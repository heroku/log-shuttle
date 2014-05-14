package main

import (
	"io/ioutil"
	"strings"
	"testing"
)



func TestLogplexBatchFormatter(t *testing.T) {
	b := NewBatch(1)
	b.Add(LogLineOne)
	b.Add(LogLineTwo)
	br, _ := NewLogplexBatchFormatter(b, noErrData, &config)
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
	b := NewBatch(1)
	b.Add(LogLineOne)  // 1 frame
	b.Add(LongLogLine) // 3 frames

	br, _ := NewLogplexBatchFormatter(b, noErrData, &config)

	if msgCount := br.MsgCount(); msgCount != 4 {
		t.Fatalf("Formatter's MsgCount != 4, is: %d\n", msgCount)
	}
}

func TestLogplexBatchFormatter_LongLine(t *testing.T) {
	b := NewBatch(3)
	b.Add(LogLineOne)  // 1 frame
	b.Add(LongLogLine) // 3 frames
	b.Add(LogLineTwo)  // 1 frame

	br, _ := NewLogplexBatchFormatter(b, noErrData, &config)
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

	if l := br.ContentLength(); l != 30188 {
		t.Fatalf("Expected a Length() of 30188, but got %d instead\n", l)
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

	if l := llr.ContentLength(); l != 81 {
		t.Fatalf("Expected a Length of 81, got %d instead\n", l)
	}
}

func TestLogplexBatchFormatter_WithHeaders(t *testing.T) {
	b := NewBatch(2)
	b.Add(LogLineOneWithHeaders) // 1 frame
	b.Add(LogLineTwoWithHeaders) // 1 frame

	config.SkipHeaders = true
	defer func() { config.SkipHeaders = false }()

	bf, _ := NewLogplexBatchFormatter(b, noErrData, &config)
	d, err := ioutil.ReadAll(bf)
	if err != nil {
		t.Fatalf("Error reading everything from batch: %q", err)
	}

	if !logplexLineOneWithHeadersPattern.Match(d) {
		t.Fatalf("Line One actual=%q\n", d)
	}
	if !logplexLineTwoWithHeadersPattern.Match(d) {
		t.Fatalf("Line Two actual=%q\n", d)
	}
}

func BenchmarkLogplexLineFormatter(b *testing.B) {
	for i := 0; i < b.N; i++ {
		lf := NewLogplexLineFormatter(LogLineOne, &config)
		_, err := ioutil.ReadAll(lf)
		if err != nil {
			b.Fatalf("Error reading everything from line: %q", err)
		}
	}
}

func BenchmarkLogplexLineFormatter_WithHeaders(b *testing.B) {
	config.SkipHeaders = true
	defer func() { config.SkipHeaders = false }()

	for i := 0; i < b.N; i++ {
		lf := NewLogplexLineFormatter(LogLineOneWithHeaders, &config)
		_, err := ioutil.ReadAll(lf)
		if err != nil {
			b.Fatalf("Error reading everything from line: %q", err)
		}
	}
}

func BenchmarkLogplexBatchFormatter(b *testing.B) {
	batch := NewBatch(50)
	for i := 0; i < 25; i++ {
		batch.Add(LogLineOne)
		batch.Add(LogLineTwo)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bf, _ := NewLogplexBatchFormatter(batch, noErrData, &config)
		_, err := ioutil.ReadAll(bf)
		if err != nil {
			b.Fatalf("Error reading everything from line: %q", err)
		}
	}
}

func BenchmarkLogplexBatchFormatter_WithHeaders(b *testing.B) {
	config.SkipHeaders = true
	defer func() { config.SkipHeaders = false }()

	batch := NewBatch(50)
	for i := 0; i < 25; i++ {
		batch.Add(LogLineOneWithHeaders)
		batch.Add(LogLineTwoWithHeaders)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bf, _ := NewLogplexBatchFormatter(batch, noErrData, &config)
		_, err := ioutil.ReadAll(bf)
		if err != nil {
			b.Fatalf("Error reading everything from line: %q", err)
		}
	}
}
