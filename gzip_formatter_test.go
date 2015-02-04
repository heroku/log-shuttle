package shuttle

import (
	"bytes"
	"compress/gzip"
	"io"
	"io/ioutil"
	"net/http"
	"testing"
)

type fakeFormatter struct {
	b []byte
	r io.Reader
}

func (f *fakeFormatter) ContentLength() int64 {
	return int64(len(f.b))
}

func (f *fakeFormatter) MsgCount() int {
	return 1
}

func (f *fakeFormatter) Request() (*http.Request, error) {
	return nil, nil
}

func (f *fakeFormatter) Read(p []byte) (int, error) {
	return f.r.Read(p)
}

func TestGzipFormatter(t *testing.T) {
	b := []byte("Hi there!")
	f := &fakeFormatter{b, bytes.NewReader(b)}
	gr := NewGzipFormatter(f)

	if gr.MsgCount() != 1 {
		t.Fatal(gr.MsgCount)
	}
	// verify that g does not preserve content length
	if gr.ContentLength() != 0 {
		t.Fatal(gr.ContentLength())
	}
	// read the compressed bytes
	compressed, err := ioutil.ReadAll(gr)
	if err != nil {
		t.Fatal(err)
	}
	// decompress the bytes and verify the message
	gunzipper, err := gzip.NewReader(bytes.NewReader(compressed))
	if err != nil {
		t.Fatal(err)
	}
	// read the uncompressed bytes
	uncompressed, err := ioutil.ReadAll(gunzipper)
	if err != nil {
		t.Fatal(err)
	}
	if string(uncompressed) != "Hi there!" {
		t.Fatal(string(uncompressed))
	}

}
