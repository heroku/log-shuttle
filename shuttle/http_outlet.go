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
	// EOFRetrySleep is the amount of time to sleep between retries caused by an io.EOF, in ms.
	EOFRetrySleep = 100
	// OtherRetrySleep is the tIme to sleep between retries for any other error, in ms.
	OtherRetrySleep = 1000
	// DepthHighWatermark is the high watermark, beyond which the outlet looses batches instead of retrying.
	DepthHighWatermark = 0.6
	// RetryFormat is the format string for retries
	RetryFormat = "at=post retry=%t msgcount=%d inbox.length=%d request_id=%q attempts=%d error=%q\n"
	// RetryWithTypeFormat if the format string for retries that also have a type
	RetryWithTypeFormat = "at=post retry=%t msgcount=%d inbox.length=%d request_id=%q attempts=%d error=%q errtype=\"%T\"\n"
)

var (
	userAgent = fmt.Sprintf("log-shuttle/%s (%s; %s; %s; %s)", Version, runtime.Version(), runtime.GOOS, runtime.GOARCH, runtime.Compiler)
)

// HTTPOutlet handles delivery of batches to HTTPendpoints by creating
// formatters for the request. HTTPOutlets handle retries, response parsing and
// lost counters
type HTTPOutlet struct {
	inbox            <-chan Batch
	stats            chan<- NamedValue
	drops            *Counter
	lost             *Counter
	lostMark         int // If len(inbox) > lostMark during error handling, don't retry
	client           *http.Client
	config           Config
	newFormatterFunc NewFormatterFunc
}

// NewHTTPOutlet returns a properly constructed HTTPOutlet
func NewHTTPOutlet(config Config, drops, lost *Counter, stats chan<- NamedValue, inbox <-chan Batch, ff NewFormatterFunc) *HTTPOutlet {
	return &HTTPOutlet{
		drops:            drops,
		lost:             lost,
		lostMark:         int(float64(config.BackBuff) * DepthHighWatermark),
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
func (h *HTTPOutlet) Outlet() {

	for batch := range h.inbox {
		h.stats <- NewNamedValue("outlet.inbox.length", float64(len(h.inbox)))

		h.retryPost(batch)
	}
}

// Retry io.EOF errors h.config.MaxAttempts times
func (h *HTTPOutlet) retryPost(batch Batch) {
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
					ErrLogger.Printf(RetryWithTypeFormat, true, msgCount, inboxLength, uuid, attempts, err, err.Err)
					if err.Err == io.EOF {
						time.Sleep(time.Duration(attempts) * EOFRetrySleep * time.Millisecond)
					} else {
						time.Sleep(time.Duration(attempts) * OtherRetrySleep * time.Millisecond)
					}
					continue
				}
			}
			ErrLogger.Printf(RetryWithTypeFormat, false, msgCount, inboxLength, uuid, attempts, err, err)
			h.lost.Add(msgCount)
		}
		return
	}
}

func (h *HTTPOutlet) post(formatter Formatter, uuid string) error {

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

func (h *HTTPOutlet) timeRequest(req *http.Request) (resp *http.Response, err error) {
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
