package main

import (
	"bytes"
	"fmt"
)

var syslogLineLayout = "<%s>%s %s %s %s %s %s %s"

// A buffer suitable for posting with a http client
// keeps track of line's Write()n to the buffer
type Batch struct {
	lineCount int
	config    *ShuttleConfig
	bytes.Buffer
}

func NewBatch(config *ShuttleConfig) (batch *Batch) {
	return &Batch{config: config}
}

func (b *Batch) LineCount() int {
	return b.lineCount
}

func (b *Batch) FormatLine(logLine *LogLine) (int, *string) {
	if b.config.SkipHeaders {
		return len(logLine.line), &logLine.line
	} else {
		formattedLine := fmt.Sprintf(
			syslogLineLayout,
			b.config.Prival,
			b.config.Version,
			logLine.when.Format("2006-01-02T15:04:05.000000+00:00"),
			b.config.Hostname,
			b.config.Appname,
			b.config.Procid,
			b.config.Msgid,
			logLine.line)
		return len(formattedLine), &formattedLine
	}
}

// Write a line to the batch, increment it's line counter
func (b *Batch) Write(logLine *LogLine) {
	fll, fl := b.FormatLine(logLine)
	fmt.Fprintf(&b.Buffer, "%d %s", fll, fl)
	b.lineCount++
}

// Zero the line count and reset the internal buffer
func (b *Batch) Reset() {
	b.lineCount = 0
	b.Buffer.Reset()
}

// NoOpCloser
func (b *Batch) Close() error { return nil }
