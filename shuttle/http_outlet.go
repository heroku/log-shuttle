package shuttle

import (
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"runtime"
	"time"
)

const (
	EOF_RETRY_SLEEP        = 100 // will be in ms
	OTHER_RETRY_SLEEP      = 1000
	DEPTH_WATERMARK        = 0.6
	RETRY_FORMAT           = "at=post retry=%t msgcount=%d inbox.length=%d request_id=%q attempts=%d error=%q\n"
	RETRY_WITH_TYPE_FORMAT = "at=post retry=%t msgcount=%d inbox.length=%d request_id=%q attempts=%d error=%q errtype=\"%T\"\n"
)

var (
	userAgent = fmt.Sprintf("log-shuttle/%s (%s; %s; %s; %s)", VERSION, runtime.Version(), runtime.GOOS, runtime.GOARCH, runtime.Compiler)
)

type HttpOutlet struct {
	inbox            <-chan Batch
	stats            chan<- NamedValue
	drops            *Counter
	lost             *Counter
	lostMark         int // If len(inbox) > lostMark during error handling, don't retry
	client           *http.Client
	config           ShuttleConfig
	newFormatterFunc NewFormatterFunc
}

func NewHttpOutlet(config ShuttleConfig, drops, lost *Counter, stats chan<- NamedValue, inbox <-chan Batch, ff NewFormatterFunc) *HttpOutlet {
	return &HttpOutlet{
		drops:            drops,
		lost:             lost,
		lostMark:         int(float64(config.BackBuff) * DEPTH_WATERMARK),
		stats:            stats,
		inbox:            inbox,
		config:           config,
		newFormatterFunc: ff,
		client: &http.Client{
			Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: config.SkipVerify},
				ResponseHeaderTimeout: config.Timeout,
				Dial: func(network, address string) (net.Conn, error) {
					return net.DialTimeout(network, address, config.Timeout)
				},
			},
		},
	}
}

// Outlet receives batches from the inbox and submits them to logplex via HTTP.
func (h *HttpOutlet) Outlet() {

	for batch := range h.inbox {
		h.stats <- NewNamedValue("outlet.inbox.length", float64(len(h.inbox)))

		h.retryPost(batch)
	}
}

// Retry io.EOF errors h.config.MaxAttempts times
func (h *HttpOutlet) retryPost(batch Batch) {
	var dropData, lostData errData

	edata := make([]errData, 0, 2)

	dropData.count, dropData.since = h.drops.ReadAndReset()
	if dropData.count > 0 {
		dropData.eType = errDrop
		edata = append(edata, dropData)
	}

	lostData.count, lostData.since = h.lost.ReadAndReset()
	if lostData.count > 0 {
		lostData.eType = errLost
		edata = append(edata, lostData)
	}

	uuid := batch.UUID.String()

	for attempts := 1; attempts <= h.config.MaxAttempts; attempts++ {
		formatter := h.newFormatterFunc(batch, edata, &h.config)
		err := h.post(formatter, uuid)
		if err != nil {
			inboxLength := len(h.inbox)
			msgCount := batch.MsgCount()
			err, ok := err.(*url.Error)
			if ok {
				if attempts < h.config.MaxAttempts && inboxLength < h.lostMark {
					ErrLogger.Printf(RETRY_WITH_TYPE_FORMAT, true, msgCount, inboxLength, uuid, attempts, err, err.Err)
					if err.Err == io.EOF {
						time.Sleep(time.Duration(attempts) * EOF_RETRY_SLEEP * time.Millisecond)
					} else {
						time.Sleep(time.Duration(attempts) * OTHER_RETRY_SLEEP * time.Millisecond)
					}
					continue
				}
			}
			ErrLogger.Printf(RETRY_WITH_TYPE_FORMAT, false, msgCount, inboxLength, uuid, attempts, err, err)
			h.lost.Add(msgCount)
		}
		return
	}
}

func (h *HttpOutlet) post(formatter Formatter, uuid string) error {

	req, err := http.NewRequest("POST", h.config.OutletURL(), formatter)
	if err != nil {
		return err
	}

	if cl := formatter.ContentLength(); cl > 0 {
		req.ContentLength = cl
	}
	req.Header.Add("X-Request-Id", uuid)
	req.Header.Add("User-Agent", userAgent)

	for k, v := range formatter.Headers() {
		req.Header.Add(k, v)
	}

	resp, err := h.timeRequest(req)
	if err != nil {
		return err
	}

	switch status := resp.StatusCode; {
	case status >= 400:
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			ErrLogger.Printf("at=post request_id=%q status=%d error_reading_body=%q\n", uuid, status, err)
		} else {
			ErrLogger.Printf("at=post request_id=%q status=%d body=%q\n", uuid, status, body)
		}

	default:
		if h.config.Verbose {
			ErrLogger.Printf("at=post request_id=%q status=%d\n", uuid, status)
		}
	}

	resp.Body.Close()
	return nil
}

func (h *HttpOutlet) timeRequest(req *http.Request) (resp *http.Response, err error) {
	defer func(t time.Time) {
		name := "outlet.post.time"
		if err != nil {
			name += ".failure"
		} else {
			name += ".success"
		}
		h.stats <- NewNamedValue(name, time.Since(t).Seconds())
	}(time.Now())
	return h.client.Do(req)
}
