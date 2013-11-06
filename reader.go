package main

import (
	"bufio"
	"io"
	"net"
	"time"
)

const (
	UNIXGRAM_BUFFER_SIZE = 10000 //Make this a little smaller than logplex's max (10240), so we have room for headers
)

type LogLine struct {
	line     []byte
	when     time.Time
	unixgram bool
}

type Reader struct {
	Outbox chan *LogLine
}

func NewReader(frontBuff int) *Reader {
	r := new(Reader)
	r.Outbox = make(chan *LogLine, frontBuff)
	return r
}

func (rdr *Reader) ReadUnixgram(input *net.UnixConn, stats *ProgramStats) error {
	msg := make([]byte, UNIXGRAM_BUFFER_SIZE)
	for {
		numRead, err := input.Read(msg)
		if err != nil {
			input.Close()
			return err
		}

		//make a new []byte of the right length and copy our read message into it
		thisMsg := make([]byte, numRead)
		copy(thisMsg, msg)

		//Send out a logline with the right details in it
		logLine := &LogLine{thisMsg, time.Now(), true}
		rdr.Outbox <- logLine
		stats.Reads.Add(1)
	}
}

func (rdr *Reader) Read(input io.ReadCloser, stats *ProgramStats) error {
	rdrIo := bufio.NewReader(input)

	for {
		line, err := rdrIo.ReadBytes('\n')

		if err != nil {
			input.Close()
			return err
		}

		logLine := &LogLine{line, time.Now(), false}

		rdr.Outbox <- logLine
		stats.Reads.Add(1)
	}
	return nil
}
