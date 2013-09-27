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
	Outbox   chan *string
	InFlight *sync.WaitGroup
	Drops    *Counter
	Reads    *Counter
	config   ShuttleConfig
}

func NewReader(config ShuttleConfig) *Reader {
	r := new(Reader)
	r.Outbox = make(chan *string, config.FrontBuff)
	r.InFlight = new(sync.WaitGroup)
	r.Drops = new(Counter)
	r.Reads = new(Counter)
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
		receivedTime := time.Now().UTC()

		// If we have an unbuffered chanel, we don't want to drop lines.
		// In this case we will apply back-pressure to callers of read.
		if unbuffered {
			rdr.Outbox <- rdr.FormatLine(&line, &receivedTime)
			rdr.InFlight.Add(1)
			rdr.Reads.Increment()
		} else {
			select {
			case rdr.Outbox <- rdr.FormatLine(&line, &receivedTime):
				rdr.InFlight.Add(1)
				rdr.Reads.Increment()

			// Drop lines if the channel is currently full
			default:
				rdr.Drops.Increment()
			}
		}
	}
	return nil
}

func (rdr *Reader) FormatLine(line *string, receivedTime *time.Time) *string {
	if rdr.config.SkipHeaders {
		return line
	} else {
		formattedLine := fmt.Sprintf(
			syslogLineLayout,
			rdr.config.Prival,
			rdr.config.Version,
			receivedTime.Format("2006-01-02T15:04:05.000000+00:00"),
			rdr.config.Hostname,
			rdr.config.Appname,
			rdr.config.Procid,
			rdr.config.Msgid,
			*line)
		return &formattedLine
	}
}
