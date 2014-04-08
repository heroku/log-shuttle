package main

import (
	"io/ioutil"
	"regexp"
	"strings"
	"testing"
	"time"
)

var (
	NewLine                   = byte('\n')
	LogLineOne                = LogLine{line: []byte("Hello World\n"), when: time.Now()}
	logplexTestLineOnePattern = regexp.MustCompile(`78 <190>1 [0-9T:\+\-\.]+ shuttle token shuttle - - Hello World\n`)
	LogLineTwo                = LogLine{line: []byte("The Second Test Line \n"), when: time.Now()}
	logplexTestLineTwoPattern = regexp.MustCompile(`88 <190>1 [0-9T:\+\-\.]+ shuttle token shuttle - - The Second Test Line \n`)
	LongLogLine               = LogLine{when: time.Now()}
)

func init() {
	for i := 0; i < 2980; i++ {
		LongLogLine.line = append(LongLogLine.line, []byte{'0', '1', '2', '3', '4', '5', '6', '7', '8', '9'}...)
	}
	LongLogLine.line = append(LongLogLine.line, NewLine)
}

func TestLogplexBatchFormatter(t *testing.T) {
	b := NewNBatch(1)
	b.Add(LogLineOne)
	b.Add(LogLineTwo)
	br := NewLogplexBatchFormatter(b, &config)
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

func TestLogplexBatchFormatter_MsgCount(t *testing.T) {
	b := NewNBatch(1)
	b.Add(LogLineOne)  // 1 frame
	b.Add(LongLogLine) // 3 frames

	br := NewLogplexBatchFormatter(b, &config)

	if msgCount := br.MsgCount(); msgCount != 4 {
		t.Fatalf("Formatter's MsgCount != 4, is: %d\n", msgCount)
	}
}

func TestLogplexBatchFormatter_LongLine(t *testing.T) {
	b := NewNBatch(1)
	b.Add(LogLineOne)  // 1 frame
	b.Add(LongLogLine) // 3 frames
	b.Add(LogLineTwo)  // 1 frame

	br := NewLogplexBatchFormatter(b, &config)
	d, err := ioutil.ReadAll(br)
	if err != nil {
		t.Fatalf("Error reading everything from batch: %q", err)
	}

	if c := strings.Count(string(d), " <190>1"); c != 5 {
		t.Log("'" + string(d) + "'")
		t.Fatalf("5 frames weren't generated, %d were\n", c)
	}

	if len(d) != 30188 {
		t.Log("'" + string(d) + "'")
		t.Fatalf("Expected a length of 30044, but got %d\n", len(d))
	}
}

func TestLogplexLineFormatter_Basic(t *testing.T) {
	llr := NewLogplexLineFormatter(LogLineOne, &config)
	d, err := ioutil.ReadAll(llr)
	if err != nil {
		t.Fatalf("Error reading everything from line: %q", err)
	}

	if !logplexTestLineOnePattern.Match(d) {
		t.Fatalf("actual=%q\n", d)
	}
}
