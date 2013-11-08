package main

import (
	"bufio"
	"io"
	"net"
	"time"
)

const (
	UNIXGRAM_BUFFER_SIZE = 10000 //Make this a little smaller than logplex's max (10240), so we have room for headers
	READ_DEADLINE        = 2
)

type LogLine struct {
	line    []byte
	when    time.Time
	rfc3164 bool
}

type Reader struct {
	Outbox chan *LogLine
}

func NewReader(frontBuff int) *Reader {
	r := new(Reader)
	r.Outbox = make(chan *LogLine, frontBuff)
	return r
}

func (rdr *Reader) ReadUnixgram(input *net.UnixConn, stats *ProgramStats, closeChan <-chan bool) error {
	msg := make([]byte, UNIXGRAM_BUFFER_SIZE)
	for {

		// Stop reading if we get a message
		select {
		case <-closeChan:
			input.Close()
			return nil
		default:
		}

		input.SetReadDeadline(time.Now().Add(time.Second * READ_DEADLINE))
		numRead, err := input.Read(msg)
		if err != nil {
			if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() {
				// We have a timeout error, so just loop
				continue
			} else {
				input.Close()
				return err
			}
		}

		//make a new []byte of the right length and copy our read message into it
		//TODO this is ugly, is there a better way?
		thisMsg := make([]byte, numRead)
		copy(thisMsg, msg)

		rdr.Outbox <- &LogLine{thisMsg, time.Now(), true}
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
