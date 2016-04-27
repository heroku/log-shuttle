package shuttle

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"sync"
	"testing"
	"time"
)

var longerTestData = []byte(`Lebowski ipsum what in God's holy name are you blathering about?
Dolor sit amet, consectetur adipiscing elit praesent ac magna justo.
They're nihilists.
Pellentesque ac lectus quis elit blandit fringilla a ut turpis praesent.
Mein nommen iss Karl.
Is hard to verk in zese clozes.
Felis ligula, malesuada suscipit malesuada non, ultrices non.
Shomer shabbos.
Urna sed orci ipsum, placerat id condimentum rutrum, rhoncus.
Yeah man, it really tied the room together.
Ac lorem aliquam placerat.
`)

func newTestConfig() Config {
	// Defaults should be good for most tests
	config := NewConfig()
	config.LogsURL = "http://"
	return config
}

type loopingBuffer struct {
	b     []byte
	close chan struct{}
	p     int

	mu sync.Mutex
	r  int
}

func NewLoopingBuffer(b []byte) *loopingBuffer {
	return &loopingBuffer{
		b:     b,
		close: make(chan struct{}),
	}
}

func (b *loopingBuffer) Read(p []byte) (n int, err error) {
	for n < len(p) && err == nil {
		select {
		case <-b.close:
			return n, io.EOF
		default:
		}

		c := copy(p[n:], b.b[b.p:])
		n += c
		b.p += c

		b.mu.Lock()
		b.r += c
		b.mu.Unlock()
		if b.p >= len(b.b) {
			b.p = 0
			return
		}
	}
	return
}

func (b *loopingBuffer) Close() error {
	close(b.close)
	return nil
}

// bytesRead since last time this was called
func (b *loopingBuffer) bytesRead() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	v := b.r
	b.r = 0
	return v
}

func NewLongerTestInput() *bytes.Reader {
	return bytes.NewReader(longerTestData)
}

func NewTestInput() io.ReadCloser {
	data := []byte(`Hello World
Test Line 2
`)
	return ioutil.NopCloser(bytes.NewReader(data))
}

func NewTestInputWithHeaders() io.ReadCloser {
	data := []byte(`<13>1 2013-09-25T01:16:49.371356+00:00 host token web.1 - [meta sequenceId="1"] message 1
<13>1 2013-09-25T01:16:49.402923+00:00 host token web.1 - [meta sequenceId="2"] message 2
`)
	return ioutil.NopCloser(bytes.NewReader(data))
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
	Called  int
	sync.Mutex
}

func (ts *testHelper) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var err error
	d, err := ioutil.ReadAll(r.Body)
	if err != nil {
		panic(err)
	}
	ts.Mutex.Lock()
	defer ts.Mutex.Unlock()
	ts.Called++
	// Last request wins the race
	ts.Actual = d
	ts.Headers = r.Header
}

func TestIntegration(t *testing.T) {
	th := new(testHelper)
	ts := httptest.NewServer(th)
	defer ts.Close()

	config := newTestConfig()
	config.LogsURL = ts.URL

	shut := NewShuttle(config)
	input := NewTestInput()
	shut.LoadReader(input)
	shut.Launch()
	shut.WaitForReadersToFinish()
	shut.Land()

	pat1 := regexp.MustCompile(`78 <190>1 [0-9T:\+\-\.]+ shuttle token shuttle - - Hello World`)
	pat2 := regexp.MustCompile(`78 <190>1 [0-9T:\+\-\.]+ shuttle token shuttle - - Test Line 2`)

	if !pat1.Match(th.Actual) {
		t.Fatalf("actual=%s\n", string(th.Actual))
	}
	if !pat2.Match(th.Actual) {
		t.Fatalf("actual=%s\n", string(th.Actual))
	}

	if afterDrops, _ := shut.Drops.ReadAndReset(); afterDrops != 0 {
		t.Fatalf("afterDrops=%d\n", afterDrops)
	}
}

func TestInputFormatRFC5424Integration(t *testing.T) {
	th := new(testHelper)
	ts := httptest.NewServer(th)
	defer ts.Close()

	config := newTestConfig()
	config.LogsURL = ts.URL
	config.InputFormat = InputFormatRFC5424

	shut := NewShuttle(config)
	input := NewTestInputWithHeaders()
	shut.LoadReader(input)
	shut.Launch()
	shut.WaitForReadersToFinish()
	shut.Land()

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

	config := newTestConfig()
	config.LogsURL = ts.URL
	config.InputFormat = InputFormatRaw

	shut := NewShuttle(config)
	input := NewTestInput()
	shut.LoadReader(input)
	shut.Launch()
	shut.Drops.Add(1)
	shut.Drops.Add(1)
	shut.WaitForReadersToFinish()
	shut.Land()

	pat1 := regexp.MustCompile(`138 <172>1 [0-9T:\+\-\.]+ heroku token log-shuttle - - Error L12: 2 messages dropped since [0-9T:\+\-\.]+\n`)
	if !pat1.Match(th.Actual) {
		t.Fatalf("actual=%s\n", string(th.Actual))
	}

	dropHeader, ok := th.Headers["Logplex-Drop-Count"]
	if !ok {
		t.Fatalf("Header Logplex-Drop-Count not found in response")
	}

	if dropHeader[0] != "2" {
		t.Fatalf("Logplex-Drop-Count=%s\n", dropHeader[0])
	}

	//Should be 0 because it was reset during delivery to the testHelper
	if afterDrops, _ := shut.Drops.ReadAndReset(); afterDrops != 0 {
		t.Fatalf("afterDrops=%d\n", afterDrops)
	}
}

func TestLost(t *testing.T) {
	th := new(testHelper)
	ts := httptest.NewServer(th)
	defer ts.Close()

	config := newTestConfig()
	config.LogsURL = ts.URL
	config.InputFormat = InputFormatRaw

	shut := NewShuttle(config)
	input := NewTestInput()
	shut.LoadReader(input)
	shut.Launch()

	shut.Lost.Add(1)
	shut.Lost.Add(1)

	shut.WaitForReadersToFinish()
	shut.Land()

	pat1 := regexp.MustCompile(`135 <172>1 [0-9T:\+\-\.]+ heroku token log-shuttle - - Error L13: 2 messages lost since [0-9T:\+\-\.]+\n`)
	if !pat1.Match(th.Actual) {
		t.Fatalf("actual=%s\n", string(th.Actual))
	}

	lostHeader, ok := th.Headers["Logplex-Lost-Count"]
	if !ok {
		t.Fatalf("Header Logplex-Lost-Count not found in response")
	}

	if lostHeader[0] != "2" {
		t.Fatalf("Logplex-Lost-Count=%s\n", lostHeader[0])
	}

	//Should be 0 because it was reset during delivery to the testHelper
	if afterLost, _ := shut.Lost.ReadAndReset(); afterLost != 0 {
		t.Fatalf("afterLost=%d\n", afterLost)
	}
}

func TestUserAgentHeader(t *testing.T) {
	th := new(testHelper)
	ts := httptest.NewServer(th)
	defer ts.Close()

	config := newTestConfig()
	config.LogsURL = ts.URL
	config.InputFormat = InputFormatRaw
	config.ID = "0.1-abcde"

	shut := NewShuttle(config)
	input := NewTestInput()
	shut.LoadReader(input)
	shut.Launch()

	shut.WaitForReadersToFinish()
	shut.Land()

	uaHeader, ok := th.Headers["User-Agent"]
	if !ok {
		t.Fatalf("Header User-Agent not found in response")
	}

	uaPattern := regexp.MustCompile(`^^log-shuttle/[0-9a-z-\.]+ \(go\d+(\.\d+){0,2}((beta|rc)\d)?; \w+; \w+; \w+\)$`)
	if !uaPattern.MatchString(uaHeader[0]) {
		t.Fatalf("Header User-Agent doesn't match expected pattern. Actual: %s\n", uaHeader[0])
	}
}

func TestRequestId(t *testing.T) {
	th := new(testHelper)
	ts := httptest.NewServer(th)
	defer ts.Close()

	config := newTestConfig()
	config.LogsURL = ts.URL
	config.InputFormat = InputFormatRaw

	shut := NewShuttle(config)
	input := NewTestInput()
	shut.LoadReader(input)
	shut.Launch()
	shut.WaitForReadersToFinish()
	shut.Land()

	_, ok := th.Headers["X-Request-Id"]
	if !ok {
		t.Fatalf("Header X-Request-ID not found in response")
	}
}

type lenner interface {
	Len() int
}

func BenchmarkPipeline(b *testing.B) {
	th := new(noopTestHelper)
	ts := httptest.NewServer(th)
	defer ts.Close()

	config := newTestConfig()
	config.LogsURL = ts.URL
	config.InputFormat = InputFormatRaw

	shut := NewShuttle(config)
	input := NewLoopingBuffer(longerTestData)
	shut.LoadReader(input)
	shut.Launch()
	var tb int
	for i := 0; i < b.N; i++ {
		// This sleep is here to allow the reading goroutines to make
		// progress reading. Open to better ideas.
		time.Sleep(10 * time.Microsecond)
		tb += input.bytesRead()
	}
	b.SetBytes(int64(tb / b.N))
	shut.Land()
}

func ExampleShuttle() {
	config := NewConfig()
	// Modulate the config as needed before creating a new shuttle
	s := NewShuttle(config)
	s.LoadReader(os.Stdin)
	s.Launch() // Start up the batching/delivering go routines
	s.Land()   // Spin down the batching/delivering go routines
}
