package main

import (
	"bufio"
	"fmt"
	"io"
	"sync"
	"time"
)

var syslogLineLayout = "<%s>%s %s %s %s %s %s %s"

type Reader struct {
	Outbox   chan<- string
	Config   *ShuttleConfig
	inFlight *sync.WaitGroup
	Drops    *Counter
	Reads    *Counter
}

func NewReader(cfg *ShuttleConfig, inFlight *sync.WaitGroup, drops *Counter, outbox chan<- string) *Reader {
	r := new(Reader)
	r.Config = cfg
	r.inFlight = inFlight
	r.Drops = drops
	r.Outbox = outbox
	r.Reads = new(Counter)
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

		// If we have an unbuffered chanel, we don't want to drop lines.
		// In this case we will apply back-pressure to callers of read.
		if unbuffered {
			rdr.Outbox <- rdr.FormatLine(line)
			rdr.inFlight.Add(1)
			rdr.Reads.Increment()
		} else {
			select {
			case rdr.Outbox <- rdr.FormatLine(line):
				rdr.inFlight.Add(1)
				rdr.Reads.Increment()

			// Drop lines if the channel is currently full
			default:
				rdr.Drops.Increment()
			}
		}
	}
	return nil
}

func (rdr *Reader) FormatLine(line string) string {
	if rdr.Config.SkipHeaders {
		return line
	} else {
		return fmt.Sprintf(
			syslogLineLayout,
			rdr.Config.Prival,
			rdr.Config.Version,
			time.Now().UTC().Format("2006-01-02T15:04:05.000000+00:00"),
			rdr.Config.Hostname,
			rdr.Config.Appname,
			rdr.Config.Procid,
			rdr.Config.Msgid,
			line)
	}
}
