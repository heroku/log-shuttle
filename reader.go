package main

import (
	"bufio"
	"io"
	"time"
)

type LogLine struct {
	line string
	when time.Time
}

type Reader struct {
	Outbox chan *LogLine
	stats  *Stats
	config ShuttleConfig
}

func NewReader(config ShuttleConfig, stats *Stats) *Reader {
	r := new(Reader)
	r.Outbox = make(chan *LogLine, config.FrontBuff)
	r.stats = stats
	r.config = config
	return r
}

func (rdr *Reader) Read(input io.ReadCloser) error {
	rdrIo := bufio.NewReader(input)
	unbuffered := cap(rdr.Outbox) == 0

	for {
		line, err := rdrIo.ReadString('\n')
		if err != nil {
			input.Close()
			return err
		}

		logLine := &LogLine{line, time.Now().UTC()}

		// If we have an unbuffered chanel, we don't want to drop lines.
		// In this case we will apply back-pressure to callers of read.
		if unbuffered {
			rdr.Outbox <- logLine
			rdr.stats.InFlight.Add(1)
			rdr.stats.Reads.Add(1)
		} else {
			select {
			case rdr.Outbox <- logLine:
				rdr.stats.InFlight.Add(1)
				rdr.stats.Reads.Add(1)

			// Drop lines if the channel is currently full
			default:
				rdr.stats.Drops.Add(1)
			}
		}
	}
	return nil
}
