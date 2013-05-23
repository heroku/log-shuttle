package main

import (
	"bufio"
	"io"
	"sync/atomic"
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
