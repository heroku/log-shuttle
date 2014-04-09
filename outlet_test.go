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

const (
	outletTestInput = `Lebowski ipsum what in God's holy name are you blathering about?
Dolor sit amet, consectetur adipiscing elit praesent ac magna justo.
They're nihilists.
Pellentesque ac lectus quis elit blandit fringilla a ut turpis praesent.
Mein nommen iss Karl.
Is hard to verk in zese clozes.
Felis ligula, malesuada suscipit malesuada non, ultrices non.
Shomer shabbos.
Urna sed orci ipsum, placerat id condimentum rutrum, rhoncus.
Yeah man, it really tied the room together.
Ac lorem aliquam placerat.`
)

/*
var (
	config ShuttleConfig
)

func init() {
	config.ParseFlags() //Load defaults. Why is there no seperate function for this?
	config.Appname = "token"
	// Some test defaults
}
*/

type testEOFHelper struct {
	Actual            []byte
	called, maxCloses int
	Headers           http.Header
}

/*
type noopTestHelper struct{}

func (th *noopTestHelper) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	_, err := ioutil.ReadAll(r.Body)
	if err != nil {
		panic(err)
	}
}
*/

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

func BenchmarkOutlet(b *testing.B) {
	// This boilerplate harness startup was yanked from main_test.go.
	// May want to consolidate it in the future.
	th := new(noopTestHelper)
	ts := httptest.NewServer(th)
	defer ts.Close()

	config.LogsURL = ts.URL
	config.SkipHeaders = true
	config.SkipVerify = true

	devNull := make(chan NamedValue)
	outbox := make(chan *Batch)

	getBatches, returnBatches := NewBatchManager(config, devNull)
	outlet := NewOutlet(config, &Counter{}, &Counter{}, devNull, outbox, returnBatches)

	// pipe everything from the stats channel to a black hole, we don't care
	// this is probably wrong... what about shutdown?
	go func() {
		for {
			<-devNull
		}
	}()

	go outlet.Outlet()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		b.SetBytes(int64(len(outletTestInput)))
		b.StartTimer()
		batch := <-getBatches
		batch.Write(LogLine{[]byte(outletTestInput), time.Now()})
		outbox <- batch
	}
}
