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
	var requestBody []byte
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error
		requestBody, err = ioutil.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
	}))
	defer ts.Close()

	conf := new(ShuttleConfig)
	conf.ParseFlags()
	conf.LogsURL = ts.URL

	reader := NewReader(conf)
	outlet := NewOutlet(conf, reader.Outbox)

	go outlet.Transfer()
	go outlet.Outlet()

	reader.Input = &testInput{bytes.NewBufferString("Hello World\n")}
	reader.Read()
	outlet.InFLight.Wait()

	pat := regexp.MustCompile(`71 <190>1 [0-9T:\\+\\-]+ shuttle token shuttle - - Hello World`)
	if !pat.Match(requestBody) {
		t.Fatalf("actual=%s\n", string(requestBody))
	}
}
