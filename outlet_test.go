package main

import (
	"bytes"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"
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
	config.LogsURL = ts.URL
	config.SkipVerify = true

	schan := make(chan NamedValue)
	go func() {
		for _ = range schan {
		}
	}()
	drops := NewCounter(0)
	lost := NewCounter(0)
	outlet := NewOutlet(config, drops, lost, schan, nil, nil)

	batch := NewBatch(&config)

	batch.Write(LogLine{[]byte(logLineText), time.Now()})

	outlet.retryPost(batch)
	if th.called != 2 {
		t.Errorf("th.called != 2, == %q\n", th.called)
	}

	if batch.Lost != 0 {
		t.Errorf("batch.lost != 0, == %q\n", batch.Lost)
	}

	pat := regexp.MustCompile(logLineText)
	if !pat.Match(th.Actual) {
		t.Fatalf("actual=%s, expected=%s\n", string(th.Actual), logLineText)
	}

}

func TestOutletEOFRetryMax(t *testing.T) {
	th := &testEOFHelper{maxCloses: config.MaxAttempts}
	ts := httptest.NewTLSServer(th)
	defer ts.Close()
	config.LogsURL = ts.URL
	config.SkipVerify = true
	logCapture := new(bytes.Buffer)
	ErrLogger = log.New(logCapture, "", 0)

	schan := make(chan NamedValue)
	go func() {
		for _ = range schan {
		}
	}()
	drops := NewCounter(0)
	lost := NewCounter(0)
	outlet := NewOutlet(config, drops, lost, schan, nil, nil)

	batch := NewBatch(&config)

	batch.Write(LogLine{[]byte("Hello"), time.Now()})

	outlet.retryPost(batch)
	if th.called != config.MaxAttempts {
		t.Errorf("th.called != %q, == %q\n", config.MaxAttempts, th.called)
	}

	if lost.Read() != 1 {
		t.Errorf("lost != 1, == %q\n", lost.Read())
	}

	logMessageCheck := regexp.MustCompile(`EOF`)
	logMessage := logCapture.Bytes()
	if !logMessageCheck.Match(logMessage) {
		t.Errorf("logMessage is wrong: %q\n", logMessage)
	}

}
