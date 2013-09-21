package main

import (
	"bufio"
	"fmt"
	"io"
	"sync/atomic"
	"time"
)

type Reader struct {
	Input  io.ReadCloser
	Outbox chan string
	Config *ShuttleConfig
	reads  uint64
	drops  uint64
}

func NewReader(cfg *ShuttleConfig) *Reader {
	r := new(Reader)
	r.Config = cfg
	r.Outbox = make(chan string, r.Config.FrontBuff)
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
		// If we have an unbuffered chanel, we don't want to drop lines.
		// In this case we will apply back-pressure to callers of read.
		if cap(rdr.Outbox) == 0 {
			rdr.Outbox <- line
			atomic.AddUint64(&rdr.reads, 1)
		} else {
			select {
			case rdr.Outbox <- line:
				atomic.AddUint64(&rdr.reads, 1)
			default:
				atomic.AddUint64(&rdr.drops, 1)
			}
		}
	}
	return nil
}

func (rdr *Reader) Report() {
	for _ = range time.Tick(rdr.Config.StatInterval) {
		r := atomic.LoadUint64(&rdr.reads)
		d := atomic.LoadUint64(&rdr.drops)
		atomic.AddUint64(&rdr.reads, -r)
		atomic.AddUint64(&rdr.drops, -d)
		rdr.Outbox <- fmt.Sprintf(rdr.Config.StatsLayout, r, d)
	}
}
