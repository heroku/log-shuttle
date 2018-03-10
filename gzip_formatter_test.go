/**
 * Copyright (c) 2018 Salesforce
 * All rights reserved.
 * Licensed under the BSD 3-Clause license.
 * For full license text, see LICENSE.txt file in the repo root
 *   or https://opensource.org/licenses/BSD-3-Clause
 */

package shuttle

import (
	"bytes"
	"compress/gzip"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
)

type fakeFormatter struct {
	r io.Reader
}

func (f *fakeFormatter) MsgCount() int {
	return 1
}

func (f *fakeFormatter) Request() (*http.Request, error) {
	return http.NewRequest("POST", "http://localhost/", f.r)
}

func (f *fakeFormatter) Read(p []byte) (int, error) {
	return f.r.Read(p)
}

func TestGzipFormatter(t *testing.T) {
	testString := "Hi there!"
	f := &fakeFormatter{strings.NewReader(testString)}
	gr := NewGzipFormatter(f)

	if gr.MsgCount() != 1 {
		t.Fatal(gr.MsgCount())
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
	if string(uncompressed) != testString {
		t.Fatal(string(uncompressed))
	}

}
