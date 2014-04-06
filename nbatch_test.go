package main

import (
	"io/ioutil"
	"regexp"
	"testing"
	"time"
)

var (
	LogLineOne                = LogLine{line: []byte("Hello World\n"), when: time.Now()}
	logplexTestLineOnePattern = regexp.MustCompile(`78 <190>1 [0-9T:\+\-\.]+ shuttle token shuttle - - Hello World\n`)
	LogLineTwo                = LogLine{line: []byte("The Second Test Line \n"), when: time.Now()}
	logplexTestLineTwoPattern = regexp.MustCompile(`88 <190>1 [0-9T:\+\-\.]+ shuttle token shuttle - - The Second Test Line \n`)
)

func TestLogplexBatchReader(t *testing.T) {
	b := NewNBatch(1)
	b.Add(LogLineOne)
	b.Add(LogLineTwo)
	br := NewLogplexBatchReader(b, &config)
	d, err := ioutil.ReadAll(br)
	if err != nil {
		t.Fatalf("Error reading everything from batch: %q", err)
	}

	if !logplexTestLineOnePattern.Match(d) {
		t.Fatalf("actual=%q\n", d)
	}

	if !logplexTestLineTwoPattern.Match(d) {
		t.Fatalf("actual=%q\n", d)
	}

	t.Logf("%q", string(d))
}

func TestLogplexLineReader_Basic(t *testing.T) {
	llr := NewLogplexLineReader(LogLineOne, &config)
	d, err := ioutil.ReadAll(llr)
	if err != nil {
		t.Fatalf("Error reading everything from line: %q", err)
	}

	if !logplexTestLineOnePattern.Match(d) {
		t.Fatalf("actual=%q\n", d)
	}
}
