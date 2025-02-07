package shuttle

import (
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"runtime"
	"time"

	"github.com/rcrowley/go-metrics"
)

const (
	// EOFRetrySleep is the amount of time to sleep between retries caused by an io.EOF, in ms.
	EOFRetrySleep = 100
	// OtherRetrySleep is the time to sleep between retries for any other error, in ms.
	OtherRetrySleep = 1000
	// DepthHighWatermark is the high watermark, beyond which the outlet looses batches instead of retrying.
	DepthHighWatermark = 0.6
	// RetryWithTypeFormat if the format string for retries that also have a type
	RetryWithTypeFormat = "at=post retry=%t msgcount=%d inbox.length=%d request_id=%q attempts=%d error=%q errtype=\"%T\"\n"
)

// HTTPOutlet handles delivery of batches to HTTP endpoints by creating
// formatters for each request. HTTPOutlets handle retries, response parsing
// and lost counters
type HTTPOutlet struct {
	inbox            <-chan Batch
	drops            *Counter
	lost             *Counter
	lostMark         int // If len(inbox) > lostMark during error handling, don't retry
	client           *http.Client
	config           Config
	newFormatterFunc NewHTTPFormatterFunc
	userAgent        string

	// User supplied loggers
	Logger    *log.Logger
	errLogger *log.Logger

	// Various stats that we'll collect, see NewHTTPOutlet for names
	inboxLengthGauge metrics.Gauge   // The number of outstanding batches, updated every time we try a post
	postSuccessTimer metrics.Timer   // The timing data for successful posts
	postFailureTimer metrics.Timer   // The timing data for failed posts
	msgLostCount     metrics.Counter // The count of lost messages
}

// NewHTTPOutlet returns a properly constructed HTTPOutlet for the given shuttle
func NewHTTPOutlet(s *Shuttle) *HTTPOutlet {
	return &HTTPOutlet{
		drops:            s.Drops,
		lost:             s.Lost,
		lostMark:         int(float64(s.config.BackBuff) * DepthHighWatermark),
		inbox:            s.Batches,
		config:           s.config,
		newFormatterFunc: s.NewFormatterFunc,
		userAgent:        fmt.Sprintf("log-shuttle/%s (%s; %s; %s; %s)", s.config.ID, runtime.Version(), runtime.GOOS, runtime.GOARCH, runtime.Compiler),
		errLogger:        s.ErrLogger,
		Logger:           s.Logger,
		client: &http.Client{
			Timeout: s.config.Timeout,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: s.config.SkipVerify,
				},
			},
		},
		inboxLengthGauge: metrics.GetOrRegisterGauge("outlet.inbox.length", s.MetricsRegistry),
		postSuccessTimer: metrics.GetOrRegisterTimer("outlet.post.success", s.MetricsRegistry),
		postFailureTimer: metrics.GetOrRegisterTimer("outlet.post.failure", s.MetricsRegistry),
		msgLostCount:     metrics.GetOrRegisterCounter("msg.lost", s.MetricsRegistry),
	}
}

// Outlet receives batches from the inbox and submits them to logplex via HTTP.
func (h *HTTPOutlet) Outlet() {
	for batch := range h.inbox {
		h.retryPost(batch)
	}
}

// retryPost posts batch and will retry on error up to h.config.MaxAttempts times.
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

	for attempts := 1; attempts <= h.config.MaxAttempts; attempts++ {
		formatter := h.newFormatterFunc(batch, edata, &h.config)
		if h.config.UseGzip {
			formatter = NewGzipFormatter(formatter)
		}
		err := h.post(formatter)
		if err != nil {
			inboxLength := len(h.inbox)
			h.inboxLengthGauge.Update(int64(inboxLength))
			msgCount := batch.MsgCount()
			if attempts < h.config.MaxAttempts && inboxLength < h.lostMark {
				h.errLogger.Printf(RetryWithTypeFormat, true, msgCount, inboxLength, batch.UUID, attempts, err, err)
				var si time.Duration = OtherRetrySleep
				if isEOF(err) {
					si = EOFRetrySleep
				}
				time.Sleep(time.Duration(attempts) * si * time.Millisecond)
				continue
			}
			h.errLogger.Printf(RetryWithTypeFormat, false, msgCount, inboxLength, batch.UUID, attempts, err, err)
			h.lost.Add(msgCount)
			h.msgLostCount.Inc(int64(msgCount))
		}
		return
	}
}

func (h *HTTPOutlet) post(formatter HTTPFormatter) error {
	req, err := formatter.Request()
	if err != nil {
		return err
	}

	cr := &countingReader{
		reader: req.Body,
	}
	req.Body = io.NopCloser(cr)

	uuid := req.Header.Get("X-Request-Id")
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
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			h.errLogger.Printf("at=post request_id=%q content_length=%d msgcount=%d status=%d reading_body=true error=%q\n", uuid, cr.count, formatter.MsgCount(), status, err)
		} else {
			h.errLogger.Printf("at=post request_id=%q content_length=%d msgcount=%d status=%d body=%q\n", uuid, cr.count, formatter.MsgCount(), status, body)
		}

	default:
		if h.config.Verbose {
			h.Logger.Printf("at=post request_id=%q status=%d\n", uuid, status)
		}
		if rh, ok := formatter.(ResponseHandler); ok { // If the formatter is also a ResponseHandler, then handle the response
			err = rh.HandleResponse(resp)
		}
	}

	return err
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

// isEOF returns whether err is io.EOF or a *url.Error wrapping
// io.EOF.
func isEOF(err error) bool {
	if err == io.EOF {
		return true
	}

	uerr, ok := err.(*url.Error)
	return ok && uerr.Err == io.EOF
}

// countingReader stores the total bytes read from an underlying reader.
type countingReader struct {
	reader io.Reader
	count  int64
}

// Read implements the io.Reader interface.
func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.reader.Read(p)
	c.count += int64(n)
	return n, err
}
