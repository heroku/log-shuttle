package main

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync"
	"testing"
)

var (
	conf *ShuttleConfig
)

func init() {
	conf = new(ShuttleConfig)
	conf.ParseFlags()
}

type testInput struct {
	*bytes.Buffer
}

func NewTestInput() *testInput {
	return &testInput{bytes.NewBufferString("Hello World\nTest Line 2\n")}
}

func NewTestInputWithHeaders() *testInput {
	return &testInput{bytes.NewBufferString("<13>1 2013-09-25T01:16:49.371356+00:00 host token web.1 - [meta sequenceId=\"1\"] message 1\n<13>1 2013-09-25T01:16:49.402923+00:00 host token web.1 - [meta sequenceId=\"2\"] message 2\n")}
}

func (i *testInput) Close() error {
	return nil
}

type testHelper struct {
	Actual  []byte
	Headers http.Header
}

func (ts *testHelper) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var err error
	ts.Actual, err = ioutil.ReadAll(r.Body)
	if err != nil {
		panic(err)
	}
	ts.Headers = r.Header
}

func TestIntegration(t *testing.T) {
	th := new(testHelper)
	ts := httptest.NewServer(th)
	defer ts.Close()

	conf.BatchSize = 2
	conf.LogsURL = ts.URL

	inFlight := new(sync.WaitGroup)
	drops := new(Counter)
	frontBuff := make(chan string, conf.FrontBuff)

	outlet := NewOutlet(conf, inFlight, drops, frontBuff)
	reader := NewReader(conf, inFlight, drops, frontBuff)

	go outlet.Transfer()
	go outlet.Outlet()

	reader.Read(NewTestInput())
	reader.InFlight.Wait()

	pat1 := regexp.MustCompile(`78 <190>1 [0-9T:\+\-\.]+ shuttle token shuttle - - Hello World`)
	pat2 := regexp.MustCompile(`78 <190>1 [0-9T:\+\-\.]+ shuttle token shuttle - - Test Line 2`)

	if !pat1.Match(th.Actual) {
		t.Fatalf("actual=%s\n", string(th.Actual))
	}
	if !pat2.Match(th.Actual) {
		t.Fatalf("actual=%s\n", string(th.Actual))
	}

	dropHeader, ok := th.Headers["Logshuttle-Drops"]
	if !ok {
		t.Fatalf("Header Logshuttle-Drops not found in response")
	}

	if dropHeader[0] != "0" {
		t.Fatalf("Logshuttle-Drops=%s\n", dropHeader[0])
	}

	if afterDrops := reader.Drops.ReadAndReset(); afterDrops != 0 {
		t.Fatalf("afterDrops=%d\n", afterDrops)
	}

}

func TestSkipHeaders(t *testing.T) {
	th := new(testHelper)
	ts := httptest.NewServer(th)
	defer ts.Close()

	conf := new(ShuttleConfig)
	conf.LogsURL = ts.URL
	conf.BatchSize = 2
	conf.SkipHeaders = true

	inFlight := new(sync.WaitGroup)
	drops := new(Counter)
	frontBuff := make(chan string, conf.FrontBuff)

	outlet := NewOutlet(conf, inFlight, drops, frontBuff)
	reader := NewReader(conf, inFlight, drops, frontBuff)

	go outlet.Transfer()
	go outlet.Outlet()

	reader.Read(NewTestInputWithHeaders())
	reader.InFlight.Wait()

	pat1 := regexp.MustCompile(`90 <13>1 2013-09-25T01:16:49\.371356\+00:00 host token web\.1 - \[meta sequenceId="1"\] message 1`)
	pat2 := regexp.MustCompile(`90 <13>1 2013-09-25T01:16:49\.402923\+00:00 host token web\.1 - \[meta sequenceId="2"\] message 2`)

	if !pat1.Match(th.Actual) {
		t.Fatalf("actual=%s\n", string(th.Actual))
	}
	if !pat2.Match(th.Actual) {
		t.Fatalf("actual=%s\n", string(th.Actual))
	}
}

func TestDrops(t *testing.T) {
	th := new(testHelper)
	ts := httptest.NewServer(th)
	defer ts.Close()

	conf.BatchSize = 2
	conf.LogsURL = ts.URL

	inFlight := new(sync.WaitGroup)
	drops := new(Counter)
	frontBuff := make(chan string, conf.FrontBuff)

	outlet := NewOutlet(conf, inFlight, drops, frontBuff)
	reader := NewReader(conf, inFlight, drops, frontBuff)

	go outlet.Transfer()
	go outlet.Outlet()

	reader.Drops.Increment()
	reader.Drops.Increment()
	reader.Read(NewTestInput())
	reader.InFlight.Wait()

	dropHeader, ok := th.Headers["Logshuttle-Drops"]
	if !ok {
		t.Fatalf("Header Logshuttle-Drops not found in response")
	}

	if dropHeader[0] != "2" {
		t.Fatalf("LogShuttle-Drops=%s\n", dropHeader[0])
	}

	//Should be 0 because it was reset during delivery to the testHelper
	if afterDrops := reader.Drops.ReadAndReset(); afterDrops != 0 {
		t.Fatalf("afterDrops=%d\n", afterDrops)
	}
}
