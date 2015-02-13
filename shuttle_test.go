package shuttle

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
)

var (
	config Config
)

func init() {
	config = NewConfig() //Do this once for the test. Defaults should always be good for the tests
	config.LogsURL = "http://"
}

type TestInput struct {
	*bytes.Reader
}

func NewLongerTestInput() *TestInput {
	return &TestInput{bytes.NewReader([]byte(`Lebowski ipsum what in God's holy name are you blathering about?
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

func NewTestInput() *TestInput {
	return &TestInput{bytes.NewReader([]byte("Hello World\nTest Line 2\n"))}
}

func NewTestInputWithHeaders() *TestInput {
	return &TestInput{bytes.NewReader([]byte("<13>1 2013-09-25T01:16:49.371356+00:00 host token web.1 - [meta sequenceId=\"1\"] message 1\n<13>1 2013-09-25T01:16:49.402923+00:00 host token web.1 - [meta sequenceId=\"2\"] message 2\n"))}
}

func (i *TestInput) Close() error {
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

	shut := NewShuttle(config)
	shut.Launch()

	shut.ReadLogLines(NewTestInput())
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

	config.LogsURL = ts.URL
	config.InputFormat = InputFormatRFC5424

	shut := NewShuttle(config)
	shut.Launch()

	shut.ReadLogLines(NewTestInputWithHeaders())
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

	config.LogsURL = ts.URL
	config.InputFormat = InputFormatRaw

	shut := NewShuttle(config)
	shut.Launch()

	shut.Drops.Add(1)
	shut.Drops.Add(1)
	shut.ReadLogLines(NewTestInput())
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

	config.LogsURL = ts.URL
	config.InputFormat = InputFormatRaw

	shut := NewShuttle(config)
	shut.Launch()

	shut.Lost.Add(1)
	shut.Lost.Add(1)
	shut.ReadLogLines(NewTestInput())
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

	config.LogsURL = ts.URL
	config.InputFormat = InputFormatRaw
	config.ID = "0.1-abcde"

	shut := NewShuttle(config)
	shut.Launch()

	shut.ReadLogLines(NewTestInput())
	shut.Land()

	uaHeader, ok := th.Headers["User-Agent"]
	if !ok {
		t.Fatalf("Header User-Agent not found in response")
	}

	uaPattern := regexp.MustCompile(`^^log-shuttle/[0-9a-z-\.]+ \(go\d+(\.\d+){0,2}; \w+; \w+; \w+\)$`)
	if !uaPattern.MatchString(uaHeader[0]) {
		t.Fatalf("Header User-Agent doesn't match expected pattern. Actual: %s\n", uaHeader[0])
	}
}

func TestRequestId(t *testing.T) {
	th := new(testHelper)
	ts := httptest.NewServer(th)
	defer ts.Close()

	config.LogsURL = ts.URL
	config.InputFormat = InputFormatRaw

	shut := NewShuttle(config)
	shut.Launch()

	shut.ReadLogLines(NewTestInput())
	shut.Land()

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
	config.InputFormat = InputFormatRaw

	shut := NewShuttle(config)
	shut.Launch()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		ti := NewLongerTestInput()
		b.SetBytes(int64(ti.Len()))
		b.StartTimer()
		shut.ReadLogLines(ti)
	}
	shut.Land()
}
