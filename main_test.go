package main

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
)

type testInput struct {
	*bytes.Buffer
}

func (i *testInput) Close() error {
	return nil
}

func TestIntegration(t *testing.T) {
	var actual []byte
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error
		actual, err = ioutil.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
	}))
	defer ts.Close()

	conf := new(ShuttleConfig)
	conf.ParseFlags()
	conf.BatchSize = 2
	conf.LogsURL = ts.URL

	reader := NewReader(conf)
	outlet := NewOutlet(conf, reader.Outbox, reader.InFlight)

	go outlet.Transfer()
	go outlet.Outlet()

	reader.Input = &testInput{bytes.NewBufferString("Hello World\nTest Line 2\n")}
	reader.Read()

	//This could possibly race with Read()
	outlet.InFlight.Wait()

	pat1 := regexp.MustCompile(`78 <190>1 [0-9T:\+\-\.]+ shuttle token shuttle - - Hello World`)
	pat2 := regexp.MustCompile(`78 <190>1 [0-9T:\+\-\.]+ shuttle token shuttle - - Test Line 2`)

	if !pat1.Match(actual) {
		t.Fatalf("actual=%s\n", string(actual))
	}
	if !pat2.Match(actual) {
		t.Fatalf("actual=%s\n", string(actual))
	}
}
