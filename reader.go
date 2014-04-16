package main

import (
	"bufio"
	"io"
	"time"
)

type Reader struct {
	Outbox chan LogLine
	stats  chan<- NamedValue
}

func NewReader(frontBuff int, stats chan<- NamedValue) *Reader {
	return &Reader{
		Outbox: make(chan LogLine, frontBuff),
		stats:  stats,
	}
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

		rdr.Outbox <- logLine
		rdr.stats <- NewNamedValue("reader.line.delay.time", currentLogTime.Sub(lastLogTime).Seconds())
		lastLogTime = currentLogTime
	}
	return nil
}
