package main

import (
	"bytes"
	"fmt"
	"time"
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

// Write a line to the batch, increment it's line counter
func (b *Batch) Write(logLine *LogLine) {
	var syslogPrefix string

	if !b.config.SkipHeaders {
		syslogPrefix = "<" + b.config.Prival + ">" + b.config.Version + " " +
			logLine.when.Format("2006-01-02T15:04:05.000000+00:00") + " " +
			b.config.Hostname + " " +
			b.config.Appname + " " +
			b.config.Procid + " " +
			b.config.Msgid + " "
	}

	fmt.Fprintf(&b.Buffer, "%d %s%s", len(logLine.line)+len(syslogPrefix), syslogPrefix, logLine.line)
	b.LineCount++
}

// Zero the line count and reset the internal buffer
func (b *Batch) Reset() {
	b.LineCount = 0
	b.Buffer.Reset()
}

// NoOpCloser
func (b *Batch) Close() error { return nil }
