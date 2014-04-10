package main

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
)

var (
	config ShuttleConfig
)

func init() {
	config.ParseFlags() //Load defaults. Why is there no seperate function for this?
	config.Appname = "token"
	// Some test defaults
}

type testInput struct {
	*bytes.Reader
}

func NewLongerTestInput() *testInput {
	return &testInput{bytes.NewReader([]byte(`Lebowski ipsum what in God's holy name are you blathering about?
Dolor sit amet, consectetur adipiscing elit praesent ac magna justo.
They're nihilists.
Pellentesque ac lectus quis elit blandit fringilla a ut turpis praesent.
Mein nommen iss Karl.
Is hard to verk in zese clozes.
Felis ligula, malesuada suscipit malesuada non, ultrices non.
Shomer shabbos.
Urna sed orci ipsum, placerat id condimentum rutrum, rhoncus.
Yeah man, it really tied the room together.
Ac lorem aliquam placerat.`))}
}

func NewTestInput() *testInput {
	return &testInput{bytes.NewReader([]byte("Hello World\nTest Line 2\n"))}
}

func NewTestInputWithHeaders() *testInput {
	return &testInput{bytes.NewReader([]byte("<13>1 2013-09-25T01:16:49.371356+00:00 host token web.1 - [meta sequenceId=\"1\"] message 1\n<13>1 2013-09-25T01:16:49.402923+00:00 host token web.1 - [meta sequenceId=\"2\"] message 2\n"))}
}

func (i *testInput) Close() error {
	return nil
}

type noopTestHelper struct{}

func (th *noopTestHelper) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	_, err := ioutil.ReadAll(r.Body)
	if err != nil {
		panic(err)
	}
}

type testHelper struct {
	Actual  []byte
	Headers http.Header
}

func (ts *testHelper) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var err error
	d, err := ioutil.ReadAll(r.Body)
	if err != nil {
		panic(err)
	}
	ts.Actual = append(ts.Actual, d...)
	ts.Headers = r.Header
}

func TestIntegration(t *testing.T) {
	th := new(testHelper)
	ts := httptest.NewServer(th)
	defer ts.Close()

	config.LogsURL = ts.URL

	reader, deliverables, stats, bWaiter, oWaiter := MakeBasicBits(config)

	reader.Read(NewTestInput())
	Shutdown(reader.Outbox, stats.Input, deliverables, bWaiter, oWaiter)

	pat1 := regexp.MustCompile(`78 <190>1 [0-9T:\+\-\.]+ shuttle token shuttle - - Hello World`)
	pat2 := regexp.MustCompile(`78 <190>1 [0-9T:\+\-\.]+ shuttle token shuttle - - Test Line 2`)

	if !pat1.Match(th.Actual) {
		t.Fatalf("actual=%s\n", string(th.Actual))
	}
	if !pat2.Match(th.Actual) {
		t.Fatalf("actual=%s\n", string(th.Actual))
	}

	t.SkipNow()

	dropHeader, ok := th.Headers["Logshuttle-Drops"]
	if !ok {
		t.Fatalf("Header Logshuttle-Drops not found in response")
	}

	if dropHeader[0] != "0" {
		t.Fatalf("Logshuttle-Drops=%s\n", dropHeader[0])
	}

	if afterDrops, _ := stats.Drops.ReadAndReset(); afterDrops != 0 {
		t.Fatalf("afterDrops=%d\n", afterDrops)
	}

}

func TestSkipHeadersIntegration(t *testing.T) {
	th := new(testHelper)
	ts := httptest.NewServer(th)
	defer ts.Close()

	config.LogsURL = ts.URL
	config.SkipHeaders = true

	reader, deliverables, stats, bWaiter, oWaiter := MakeBasicBits(config)

	reader.Read(NewTestInputWithHeaders())
	Shutdown(reader.Outbox, stats.Input, deliverables, bWaiter, oWaiter)

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
	t.SkipNow()
	th := new(testHelper)
	ts := httptest.NewServer(th)
	defer ts.Close()

	config.LogsURL = ts.URL
	config.SkipHeaders = false

	reader, deliverables, stats, bWaiter, oWaiter := MakeBasicBits(config)

	stats.Drops.Add(1)
	stats.Drops.Add(1)
	reader.Read(NewTestInput())
	Shutdown(reader.Outbox, stats.Input, deliverables, bWaiter, oWaiter)

	pat1 := regexp.MustCompile(`138 <172>1 [0-9T:\+\-\.]+ heroku token log-shuttle - - Error L12: 2 messages dropped since [0-9T:\+\-\.]+`)
	if !pat1.Match(th.Actual) {
		t.Fatalf("actual=%s\n", string(th.Actual))
	}

	dropHeader, ok := th.Headers["Logshuttle-Drops"]
	if !ok {
		t.Fatalf("Header Logshuttle-Drops not found in response")
	}

	if dropHeader[0] != "2" {
		t.Fatalf("Logshuttle-Drops=%s\n", dropHeader[0])
	}

	//Should be 0 because it was reset during delivery to the testHelper
	if afterDrops, _ := stats.Drops.ReadAndReset(); afterDrops != 0 {
		t.Fatalf("afterDrops=%d\n", afterDrops)
	}
}

func TestRequestId(t *testing.T) {
	th := new(testHelper)
	ts := httptest.NewServer(th)
	defer ts.Close()

	config.LogsURL = ts.URL
	config.SkipHeaders = false

	reader, deliverables, stats, bWaiter, oWaiter := MakeBasicBits(config)

	reader.Read(NewTestInput())
	Shutdown(reader.Outbox, stats.Input, deliverables, bWaiter, oWaiter)

	_, ok := th.Headers["X-Request-Id"]
	if !ok {
		t.Fatalf("Header X-Request-ID not found in response")
	}
}

func BenchmarkPipeline(b *testing.B) {
	th := new(noopTestHelper)
	ts := httptest.NewServer(th)
	defer ts.Close()

	config.LogsURL = ts.URL
	config.SkipHeaders = false

	reader, deliverables, stats, bWaiter, oWaiter := MakeBasicBits(config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		ti := NewLongerTestInput()
		b.SetBytes(int64(ti.Len()))
		b.StartTimer()
		reader.Read(ti)
	}
	Shutdown(reader.Outbox, stats.Input, deliverables, bWaiter, oWaiter)
}
