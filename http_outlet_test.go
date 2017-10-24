package shuttle

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rcrowley/go-metrics"
)

type testEOFHelper struct {
	Actual            []byte
	called, maxCloses int
	Headers           http.Header
}

func (ts *testEOFHelper) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ts.called++
	if ts.called <= ts.maxCloses {
		conn, _, _ := w.(http.Hijacker).Hijack()
		conn.Close()
		return
	}

	var err error
	d, err := ioutil.ReadAll(r.Body)
	if err != nil {
		panic(err)
	}
	ts.Actual = append(ts.Actual, d...)
	ts.Headers = r.Header
}

func TestOutletEOFRetry(t *testing.T) {
	logLineText := "Hello"
	th := &testEOFHelper{maxCloses: 1}
	ts := httptest.NewTLSServer(th)
	defer ts.Close()

	config := newTestConfig()
	config.LogsURL = ts.URL
	config.SkipVerify = true

	s := NewShuttle(config)
	outlet := NewHTTPOutlet(s)

	batch := NewBatch(config.BatchSize)

	batch.Add(LogLine{[]byte(logLineText), time.Now()})

	outlet.retryPost(batch)
	if th.called != 2 {
		t.Errorf("th.called != 2, == %d\n", th.called)
	}

	if lost := s.Lost.Read(); lost != 0 {
		t.Errorf("lost != 0, == %d\n", lost)
	}

	pat := regexp.MustCompile(logLineText)
	if !pat.Match(th.Actual) {
		t.Fatalf("actual=%s, expected=%s\n", string(th.Actual), logLineText)
	}
}

func TestOutletEOFRetryMax(t *testing.T) {
	config := newTestConfig()
	th := &testEOFHelper{maxCloses: config.MaxAttempts}
	ts := httptest.NewTLSServer(th)
	defer ts.Close()
	config.LogsURL = ts.URL
	config.SkipVerify = true

	var logCapture bytes.Buffer
	s := NewShuttle(config)
	s.ErrLogger = log.New(&logCapture, "", 0)

	outlet := NewHTTPOutlet(s)

	batch := NewBatch(config.BatchSize)

	batch.Add(LogLine{[]byte("Hello"), time.Now()})

	outlet.retryPost(batch)
	if th.called != config.MaxAttempts {
		t.Errorf("th.called != %q, == %d\n", config.MaxAttempts, th.called)
	}

	if lost := s.Lost.Read(); lost != 1 {
		t.Errorf("lost != 1, == %d\n", lost)
	}

	mrLost := metrics.GetOrRegisterCounter("msg.lost", s.MetricsRegistry)
	if lost := mrLost.Count(); lost != 1 {
		t.Errorf("lost != 1, == %d\n", lost)
	}

	if msg := logCapture.Bytes(); !bytes.Contains(msg, []byte("EOF")) {
		t.Errorf("expected log message to contain `EOF`, got %q", msg)
	}
}

func TestOutletOtherRetry(t *testing.T) {
	logLineText := "Hello"

	var reqs int
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqs++
		if reqs >= 3 {
			return
		}

		h := w.(http.Hijacker)

		c, _, err := h.Hijack()
		if err != nil {
			panic(err)
		}

		// Produces a "malformed HTTP response" error.
		c.Write([]byte("bogus"))
		c.Close()
	}))
	defer ts.Close()

	config := newTestConfig()
	config.LogsURL = ts.URL
	config.SkipVerify = true

	var logCapture bytes.Buffer
	s := NewShuttle(config)
	s.ErrLogger = log.New(&logCapture, "", 0)

	outlet := NewHTTPOutlet(s)

	batch := NewBatch(config.BatchSize)

	batch.Add(LogLine{[]byte(logLineText), time.Now()})

	outlet.retryPost(batch)

	if lost := s.Lost.Read(); lost != 0 {
		t.Errorf("lost != 0, == %d\n", lost)
	}
}

func TestOutletOtherRetryMax(t *testing.T) {
	config := newTestConfig()
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	ts.Close() // Close so connection is refused.
	config.LogsURL = ts.URL
	config.SkipVerify = true

	var logCapture bytes.Buffer
	s := NewShuttle(config)
	s.ErrLogger = log.New(&logCapture, "", 0)

	outlet := NewHTTPOutlet(s)

	batch := NewBatch(config.BatchSize)

	batch.Add(LogLine{[]byte("Hello"), time.Now()})

	outlet.retryPost(batch)

	if lost := s.Lost.Read(); lost != 1 {
		t.Errorf("lost != 1, == %d\n", lost)
	}

	mrLost := metrics.GetOrRegisterCounter("msg.lost", s.MetricsRegistry)
	if lost := mrLost.Count(); lost != 1 {
		t.Errorf("lost != 1, == %d\n", lost)
	}

	if msg := logCapture.Bytes(); !bytes.Contains(msg, []byte("refused")) {
		t.Errorf("expected log message to contain `refused`, got %q", msg)
	}
}

func TestTimeout(t *testing.T) {
	var called int32
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&called, 1)
		if v := atomic.LoadInt32(&called); v < 2 {
			time.Sleep(250 * time.Millisecond)
		}
	}))
	defer ts.Close()

	config := newTestConfig()
	config.LogsURL = ts.URL
	config.SkipVerify = true
	config.Timeout = 100 * time.Millisecond

	var logCapture bytes.Buffer
	s := NewShuttle(config)
	s.ErrLogger = log.New(&logCapture, "", 0)

	outlet := NewHTTPOutlet(s)

	batch := NewBatch(config.BatchSize)
	batch.Add(LogLine{[]byte("Hello"), time.Now()})
	outlet.retryPost(batch)

	if v := atomic.LoadInt32(&called); v != 2 {
		t.Errorf("expected called to be 2, but got %d", called)
	}

	if lost := s.Lost.Read(); lost > 0 {
		t.Errorf("expected lost of 0, got %d", lost)
	}

	mrLost := metrics.GetOrRegisterCounter("msg.lost", s.MetricsRegistry)
	if lost := mrLost.Count(); lost > 0 {
		t.Errorf("expected metrics lost of 0, got %d\n", lost)
	}

	if msg := logCapture.Bytes(); !bytes.Contains(msg, []byte("retry=true")) {
		t.Errorf("expected log message to contain `retry=trye`, got %q", msg)
	}

}

func TestIsEOF(t *testing.T) {
	if !isEOF(io.EOF) {
		t.Error("got isEOF(io.EOF) = false, want true")
	}

	uerr := &url.Error{Err: io.EOF}
	if !isEOF(uerr) {
		t.Errorf("got isEOF(%+v) = false, want true", uerr)
	}

	if isEOF(errors.New("hello")) {
		t.Error("got isEOF(errors.New()) = true, want false")
	}

	uerr = &url.Error{Err: errors.New("hello")}
	if isEOF(uerr) {
		t.Errorf("got isEOF(%+v) = true, want false", uerr)
	}
}
