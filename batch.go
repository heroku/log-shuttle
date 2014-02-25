package main

//TODO(edwardam): refactor syslogPrefix bits

import (
	"bytes"
	"fmt"
	"github.com/nu7hatch/gouuid"
	"time"
)

const (
	SYSLOG_TIME_LENGTH = 15    // locally this is always 15 AFAICT, but may not be if we decide to take input from elsewhere
	LOGPLEX_MAX_LENGTH = 10000 // It's actually 10240, but leave enough space for headers
	BATCH_TIME_FORMAT  = "2006-01-02T15:04:05.000000+00:00"
)

var (
	PRIVAL_END = []byte(">")
)

// A buffer suitable for posting with a http client
// keeps track of line's Write()n to the buffer
type Batch struct {
	MsgCount    int
	config      *ShuttleConfig
	oldest      *time.Time
	newest      *time.Time
	UUID        *uuid.UUID
	Drops, Lost int
	bytes.Buffer
}

// Create a new batch
func NewBatch(config *ShuttleConfig) (batch *Batch) {
	batch = &Batch{config: config}
	batch.Reset()
	return
}

// Generates a new UUID for the batch
func (b *Batch) SetUUID() {
	rid, err := uuid.NewV4()
	if err != nil {
		ErrLogger.Printf("at=generate_uuid err=%q\n", err)
	}
	b.UUID = rid
}

// Returns the time range of the messages in the batch in seconds
func (b *Batch) MsgAgeRange() float64 {
	if b.oldest == nil || b.newest == nil {
		return 0.0
	}
	newest := *b.newest
	return newest.Sub(*b.oldest).Seconds()
}

func (b *Batch) writeError(code, codeMsg string) {
	prefix := fmt.Sprintf("<172>%s %s heroku %s log-shuttle %s ",
		b.config.Version,
		time.Now().UTC().Format(BATCH_TIME_FORMAT),
		b.config.Appname,
		b.config.Msgid,
	)
	msg := fmt.Sprintf("Error %s: %s.", code, codeMsg)
	b.writeMsg(prefix, []byte(msg))
}

func (b *Batch) WriteDrops(dropped int, since time.Time) {
	b.Drops = dropped
	b.writeError("L12", fmt.Sprintf("%d messages dropped since %s", dropped, since.UTC().Format(BATCH_TIME_FORMAT)))
}

func (b *Batch) WriteLost(lost int, since time.Time) {
	b.Lost = lost
	b.writeError("L13", fmt.Sprintf("%d messages lost since %s", lost, since.UTC().Format(BATCH_TIME_FORMAT)))
}

// Write a message into the buffer, incrementing MsgCount
// TODO(edwardam): Ensure that we can't recurse forever
func (b *Batch) writeMsg(prefix string, msg []byte) {
	msgLen := len(msg)
	if msgLen > LOGPLEX_MAX_LENGTH {
		for i := 0; i < msgLen; i += LOGPLEX_MAX_LENGTH {
			target := i + LOGPLEX_MAX_LENGTH
			if target > msgLen {
				target = msgLen
			}
			b.writeMsg(prefix, msg[i:target])
		}
	} else {
		fmt.Fprintf(&b.Buffer, "%d %s%s", len(prefix)+msgLen, prefix, msg)
		b.MsgCount++
	}
}

// Write an RFC5424 msg to the buffer from the RFC3164 formatted msg
// TODO(edwardam): Punt on time manipulation for now, use received time
// TODO(edwardam): Punt on host/tag/pid for now, use value from config
func (b *Batch) writeRFC3164Msg(logLine LogLine) {
	var msg []byte

	b.UpdateTimes(logLine.when)

	// Figure out the prival
	pe := bytes.Index(logLine.line, PRIVAL_END)
	prival := string(logLine.line[1:pe])

	//Find the first ': ' after the syslog time, naive, but meh
	logLineLength := len(logLine.line)
	for msgStart := pe + SYSLOG_TIME_LENGTH; msgStart < logLineLength; msgStart++ {
		if logLine.line[msgStart] == ':' && logLine.line[msgStart+1] == ' ' {
			msg = logLine.line[msgStart+2 : len(logLine.line)]
			break
		}
	}

	syslogPrefix := "<" + prival + ">" + b.config.Version + " " +
		logLine.when.UTC().Format(BATCH_TIME_FORMAT) + " " +
		b.config.Hostname + " " +
		b.config.Appname + " " +
		b.config.Procid + " " +
		b.config.Msgid + " "

	b.writeMsg(syslogPrefix, msg)
}

// Write a line to the batch, increment it's line counter
func (b *Batch) Write(logLine LogLine) {
	b.UpdateTimes(logLine.when)

	if b.config.InputFormat == INPUT_FORMAT_RFC3164 {
		b.writeRFC3164Msg(logLine)
	} else {
		var syslogPrefix string

		if !b.config.SkipHeaders {
			syslogPrefix = "<" + b.config.Prival + ">" + b.config.Version + " " +
				logLine.when.UTC().Format(BATCH_TIME_FORMAT) + " " +
				b.config.Hostname + " " +
				b.config.Appname + " " +
				b.config.Procid + " " +
				b.config.Msgid + " "
		}

		b.writeMsg(syslogPrefix, logLine.line)
	}
}

func (b *Batch) UpdateTimes(t time.Time) {
	if b.oldest == nil || t.Before(*b.oldest) {
		b.oldest = &t
	}
	if b.newest == nil || t.After(*b.newest) {
		b.newest = &t
	}
	return
}

// Zero the line count and reset the internal buffer
func (b *Batch) Reset() {
	b.MsgCount = 0
	b.newest = nil
	b.oldest = nil
	b.Drops = 0
	b.Lost = 0
	b.SetUUID()
	b.Buffer.Reset()
}

// Is the batch full?
func (b *Batch) Full() bool {
	if b.MsgCount >= b.config.BatchSize {
		return true
	}
	return false
}

// NoOpCloser
func (b *Batch) Close() error { return nil }

func SyslogFields(r rune) bool {
	if r == '<' || r == '>' || r == ' ' {
		return true
	}
	return false
}
