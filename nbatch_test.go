package main

import (
	"io/ioutil"
	"regexp"
	"testing"
	"time"
)

func TestLogplexBatchReader_Test(t *testing.T) {
	b := NewNBatch(1)
	b.Add(LogLine{line: []byte("Hello World\n"), when: time.Now()})
	b.Add(LogLine{line: []byte("The Second Test Line \n"), when: time.Now()})
	br := NewLogplexBatchReader(b, &config)
	d, err := ioutil.ReadAll(br)
	if err != nil {
		t.Fatalf("Error reading everything from batch: %q", err)
	}

	pat1 := regexp.MustCompile(`78 <190>1 [0-9T:\+\-\.]+ shuttle token shuttle - - Hello World\n`)
	pat2 := regexp.MustCompile(`88 <190>1 [0-9T:\+\-\.]+ shuttle token shuttle - - The Second Test Line \n`)

	if !pat1.Match(d) {
		t.Fatalf("actual=%q\n", d)
	}

	if !pat2.Match(d) {
		t.Fatalf("actual=%q\n", d)
	}

	t.Logf("%q", string(d))
}
