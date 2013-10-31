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
	Outbox chan *LogLine
}

func NewReader(frontBuff int) *Reader {
	r := new(Reader)
	r.Outbox = make(chan *LogLine, frontBuff)
	return r
}

func (rdr *Reader) Read(input io.ReadCloser, stats *ProgramStats) error {
	rdrIo := bufio.NewReader(input)

	for {
		line, err := rdrIo.ReadBytes('\n')

		if err != nil {
			input.Close()
			return err
		}

		logLine := &LogLine{line, time.Now().UTC()}

		rdr.Outbox <- logLine
		stats.Reads.Add(1)
	}
	return nil
}
