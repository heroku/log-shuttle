package main

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type testEOFHelper struct {
	Actual  []byte
	called  int
	Headers http.Header
}

func (ts *testEOFHelper) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ts.called++
	if ts.called == 1 {
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
	th := new(testEOFHelper)
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

	batch.Write(LogLine{[]byte("Hello"), time.Now()})

	outlet.retryPost(batch)
	if th.called != 2 {
		t.Errorf("th.called != 2, == %q\n", th.called)
	}

	if batch.Lost != 0 {
		t.Errorf("batch.lost != 0, == %q\n", batch.Lost)
	}

}
