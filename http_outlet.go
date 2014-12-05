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

	"github.com/rcrowley/go-metrics"
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

// HTTPOutlet handles delivery of batches to HTTPendpoints by creating
// formatters for the request. HTTPOutlets handle retries, response parsing and
// lost counters
type HTTPOutlet struct {
	inbox            <-chan Batch
	drops            *Counter
	lost             *Counter
	lostMark         int // If len(inbox) > lostMark during error handling, don't retry
	client           *http.Client
	config           Config
	newFormatterFunc NewHTTPFormatterFunc
	userAgent        string

	// Various stats that we'll collect, see NewHTTPOutlet for names
	inboxLengthGauge metrics.Gauge // The number of outstanding batches, reported every time after we read a batch from the channel.
	postSuccessTimer metrics.Timer // The timing data for successful posts
	postFailureTimer metrics.Timer // The timing data for failed posts
}

// NewHTTPOutlet returns a properly constructed HTTPOutlet
func NewHTTPOutlet(config Config, drops, lost *Counter, m metrics.Registry, inbox <-chan Batch, ff NewHTTPFormatterFunc) *HTTPOutlet {
	return &HTTPOutlet{
		drops:            drops,
		lost:             lost,
		lostMark:         int(float64(config.BackBuff) * DepthHighWatermark),
		inbox:            inbox,
		config:           config,
		newFormatterFunc: ff,
		userAgent:        fmt.Sprintf("log-shuttle/%s (%s; %s; %s; %s)", config.ID, runtime.Version(), runtime.GOOS, runtime.GOARCH, runtime.Compiler),
		client: &http.Client{
			Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: config.SkipVerify},
				ResponseHeaderTimeout: config.Timeout,
				Dial: func(network, address string) (net.Conn, error) {
					return net.DialTimeout(network, address, config.Timeout)
				},
			},
		},
		inboxLengthGauge: metrics.GetOrRegisterGauge("outlet.inbox.length", m),
		postSuccessTimer: metrics.GetOrRegisterTimer("outlet.post.success.time", m),
		postFailureTimer: metrics.GetOrRegisterTimer("outlet.post.failure.time", m),
	}
}

// Outlet receives batches from the inbox and submits them to logplex via HTTP.
func (h *HTTPOutlet) Outlet() {

	for batch := range h.inbox {
		h.inboxLengthGauge.Update(int64(len(h.inbox)))

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

func (h *HTTPOutlet) post(formatter HTTPFormatter, uuid string) error {
	req, err := formatter.Request()
	if err != nil {
		return err
	}

	req.Header.Add("X-Request-Id", uuid)
	req.Header.Add("User-Agent", h.userAgent)

	resp, err := h.timeRequest(req)
	// There is a way we can have an err and a resp that is not nil, so always
	// close the Body if we have a resp
	defer func() {
		if resp != nil {
			resp.Body.Close()
		}
	}()
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

	return nil
}

func (h *HTTPOutlet) timeRequest(req *http.Request) (resp *http.Response, err error) {
	defer func(t time.Time) {
		if err != nil {
			h.postFailureTimer.UpdateSince(t)
		} else {
			h.postSuccessTimer.UpdateSince(t)
		}
	}(time.Now())
	return h.client.Do(req)
}
