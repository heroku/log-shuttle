package main

import (
	"bufio"
	"io"
	"time"
)

type LogLine struct {
	line []byte
	when time.Time
}

type Reader struct {
	outbox chan<- LogLine
	stats  chan<- NamedValue
}

func NewReader(out chan<- LogLine, stats chan<- NamedValue) *Reader {
	return &Reader{outbox: out, stats: stats}
}

func (rdr *Reader) Read(input io.ReadCloser) error {
	rdrIo := bufio.NewReader(input)

	lastLogTime := time.Now()

	for {
		line, err := rdrIo.ReadBytes('\n')
		currentLogTime := time.Now()

		if err != nil {
			input.Close()
			return err
		}

		logLine := LogLine{line, currentLogTime}

		rdr.outbox <- logLine
		rdr.stats <- NewNamedValue("reader.line.delay", currentLogTime.Sub(lastLogTime).Seconds())
		lastLogTime = currentLogTime
	}
	return nil
}
