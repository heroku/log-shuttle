package main

import (
	"bytes"
	"fmt"
	"time"
)

const (
	SYSLOG_TIME_LENGTH = 15
)

var (
	PRIVAL_END = []byte(">")
)

// A buffer suitable for posting with a http client
// keeps track of line's Write()n to the buffer
type Batch struct {
	LineCount int
	config    *ShuttleConfig
	bytes.Buffer
}

func NewBatch(config *ShuttleConfig) (batch *Batch) {
	return &Batch{config: config}
}

func (b *Batch) WriteDrops(drops int) {
	now := time.Now().UTC().Format("2006-01-02T15:04:05.000000+00:00")
	line := fmt.Sprintf("<172>%s %s heroku %s log-shuttle %s Error L12: %d messages dropped since %s.",
		b.config.Version,
		now,
		b.config.Appname,
		b.config.Msgid,
		drops,
		now,
	)
	fmt.Fprintf(&b.Buffer, "%d %s", len(line), line)
	b.LineCount++
}

func (b *Batch) writeLine(prefix *string, msg []byte) {
	fmt.Fprintf(&b.Buffer, "%d %s%s", len(*prefix)+len(msg), *prefix, msg)
}

// Write an RFC5424 line to the buffer from the RFC3164 formatted line
//TODO: Punt on time manipulation for now, use received time
//TODO: Punt on host/tag/pid for now, use value from config
func (b *Batch) writeRFC3164(logLine *LogLine) {
	var msg []byte

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
		logLine.when.UTC().Format("2006-01-02T15:04:05.000000+00:00") + " " +
		b.config.Hostname + " " +
		b.config.Appname + " " +
		b.config.Procid + " " +
		b.config.Msgid + " "

	b.writeLine(&syslogPrefix, msg)
}

// Write a line to the batch, increment it's line counter
func (b *Batch) Write(logLine *LogLine) {

	if logLine.unixgram {
		b.writeRFC3164(logLine)
	} else {
		var syslogPrefix string

		if !b.config.SkipHeaders {
			syslogPrefix = "<" + b.config.Prival + ">" + b.config.Version + " " +
				logLine.when.UTC().Format("2006-01-02T15:04:05.000000+00:00") + " " +
				b.config.Hostname + " " +
				b.config.Appname + " " +
				b.config.Procid + " " +
				b.config.Msgid + " "
		}

		b.writeLine(&syslogPrefix, logLine.line)
	}

	b.LineCount++
}

// Zero the line count and reset the internal buffer
func (b *Batch) Reset() {
	b.LineCount = 0
	b.Buffer.Reset()
}

// NoOpCloser
func (b *Batch) Close() error { return nil }

func SyslogFields(r rune) bool {
	if r == '<' || r == '>' || r == ' ' {
		return true
	}
	return false
}
