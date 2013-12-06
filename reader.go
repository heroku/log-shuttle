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
	outbox chan<- LogLine
	stats  chan<- NamedValue
}

func NewReader(out chan<- LogLine, stats chan<- NamedValue) *Reader {
	return &Reader{outbox: out, stats: stats}
}

//TODO: Refactor to use net.Conn interface for testing and simplicity reasons
// See reader_test.go for comments about not being able to consume fast enough
// Switching to net.Conn should also help with the test, because we can implement the interface
// instead of actually stressing a socket
func (rdr *Reader) ReadUnixgram(input *net.UnixConn, closeChan <-chan bool) error {
	msg := make([]byte, UNIXGRAM_BUFFER_SIZE)

	lastLogTime := time.Now()

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
		currentLogTime := time.Now()
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

		rdr.outbox <- LogLine{thisMsg, currentLogTime, thisMsg[0] == '<'}
		rdr.stats <- NamedValue{value: currentLogTime.Sub(lastLogTime).Seconds(), name: "unixgramreader.msg.delay"}
		lastLogTime = currentLogTime
	}
}

func (rdr *Reader) Read(input io.ReadCloser) error {
	rdrIo := bufio.NewReader(input)

	lastLogTime := time.Now()

	for {
		line, err := rdrIo.ReadBytes('\n')
		currentLogTime := time.Now()

		if err != nil {
			input.Close()
			return err
		}

		logLine := LogLine{line, currentLogTime, false}

		rdr.outbox <- logLine
		rdr.stats <- NamedValue{value: currentLogTime.Sub(lastLogTime).Seconds(), name: "reader.line.delay"}
		lastLogTime = currentLogTime
	}
	return nil
}
