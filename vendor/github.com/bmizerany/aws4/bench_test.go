package aws4

import (
	"bytes"
	"net/http"
	"testing"
	"time"
)

func BenchmarkSign(b *testing.B) {
	b.StopTimer()

	body := bytes.NewBufferString("foo=bar")
	r, _ := http.NewRequest("POST", "http://example.com", body)
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=utf8")
	r.Header.Set("Date", time.Now().UTC().Format(http.TimeFormat))

	s := &Service{
		Name:   "iam",
		Region: "us-east-1",
	}

	k := &Keys{
		AccessKey: "AKIDEXAMPLE",
		SecretKey: "wJalrXUtnFEMI/K7MDENG+bPxRfiCYEXAMPLEKEY",
	}

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		s.Sign(k, r)
	}
}
