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
	Input    io.ReadCloser
	Outbox   chan string
	Config   *ShuttleConfig
	InFlight *sync.WaitGroup
	Drops    Counter
	Reads    Counter
}

func NewReader(cfg *ShuttleConfig) *Reader {
	r := new(Reader)
	r.Config = cfg
	r.Outbox = make(chan string, r.Config.FrontBuff)
	r.InFlight = new(sync.WaitGroup)
	return r
}

func (rdr *Reader) Read(input io.ReadCloser) error {
	// This is here so as long as this function is running anything
	// Wait() on InFlight will actually block
	rdr.InFlight.Add(1)
	defer rdr.InFlight.Done()

	rdrIo := bufio.NewReader(input)
	for {
		line, err := rdrIo.ReadString('\n')
		if err != nil {
			input.Close()
			return err
		}

		sline := fmt.Sprintf(syslogLineLayout,
			rdr.Config.Prival,
			rdr.Config.Version,
			time.Now().UTC().Format("2006-01-02T15:04:05.000000+00:00"),
			rdr.Config.Hostname,
			rdr.Config.Appname,
			rdr.Config.Procid,
			rdr.Config.Msgid,
			line)

		// If we have an unbuffered chanel, we don't want to drop lines.
		// In this case we will apply back-pressure to callers of read.
		if cap(rdr.Outbox) == 0 {
			rdr.Outbox <- sline
			rdr.Reads.Increment()
			rdr.InFlight.Add(1)
		} else {
			select {
			case rdr.Outbox <- sline:
				rdr.Reads.Increment()
				rdr.InFlight.Add(1)
			default:
				rdr.Drops.Increment()
			}
		}
	}
	return nil
}
