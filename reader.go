package main

import (
	"bufio"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"
)

var syslogLineLayout = "<%s>%s %s %s %s %s %s %s"

type Reader struct {
	Input    io.ReadCloser
	Outbox   chan string
	Config   *ShuttleConfig
	InFlight *sync.WaitGroup
	reads    uint64
	drops    uint64
}

func NewReader(cfg *ShuttleConfig) *Reader {
	r := new(Reader)
	r.Config = cfg
	r.Outbox = make(chan string, r.Config.FrontBuff)
	r.InFlight = new(sync.WaitGroup)
	return r
}

func (rdr *Reader) Read() error {
	rdrIo := bufio.NewReader(rdr.Input)
	for {
		line, err := rdrIo.ReadString('\n')
		if err != nil {
			rdr.Input.Close()
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
			atomic.AddUint64(&rdr.reads, 1)
			rdr.InFlight.Add(1)
		} else {
			select {
			case rdr.Outbox <- sline:
				atomic.AddUint64(&rdr.reads, 1)
				rdr.InFlight.Add(1)
			default:
				atomic.AddUint64(&rdr.drops, 1)
			}
		}
	}
	return nil
}
