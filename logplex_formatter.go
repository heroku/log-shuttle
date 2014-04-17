package main

import (
	"fmt"
	"io"
	"time"
)

const (
	LOGPLEX_MAX_LENGTH        = 10000                              // It's actually 10240, but leave enough space for headers
	LOGPLEX_BATCH_TIME_FORMAT = "2006-01-02T15:04:05.000000+00:00" // The format of the timestamp
)

type Lengthy interface {
	Length() int
}

// LogplexBatchFormatter implements on io.Reader that returns Logplex formatted
// log lines.  Wraps log lines in length prefixed rfc5424 formatting, splitting
// them as necessary to LOGPLEX_MAX_LENGTH
type LogplexBatchFormatter struct {
	curLogLine   int // Current Log Line
	b            Batch
	curFormatter io.Reader // Current sub formatter
	config       *ShuttleConfig
}

// Returns a new LogplexBatchFormatter wrapping the provided batch
func NewLogplexBatchFormatter(b Batch, config *ShuttleConfig) *LogplexBatchFormatter {
	return &LogplexBatchFormatter{b: b, config: config}
}

// The msgcount of the wrapped batch. Because it splits lines at
// LOGPLEX_MAX_LENGTH this may be different from the actual MsgCount of the
// batch
func (bf *LogplexBatchFormatter) MsgCount() (msgCount int) {
	if bf.config.SkipHeaders {
		msgCount = bf.b.MsgCount()
	} else {
		for _, line := range bf.b.logLines {
			msgCount += 1 + int(len(line.line)/LOGPLEX_MAX_LENGTH)
		}
	}
	return
}

//Splits the line into a batch of loglines of LOGPLEX_MAX_LENGTH length
func splitLine(ll LogLine) Batch {
	l := ll.Length()
	batch := NewBatch(int(l/LOGPLEX_MAX_LENGTH) + 1)
	for i := 0; i < l; i += LOGPLEX_MAX_LENGTH {
		t := i + LOGPLEX_MAX_LENGTH
		if t > l {
			t = l
		}
		batch.Add(LogLine{line: ll.line[i:t], when: ll.when})
	}
	return batch
}

func (bf *LogplexBatchFormatter) Length() (length int) {
	for cli := 0; cli < len(bf.b.logLines); cli++ {
		cl := bf.b.logLines[cli]
		if cll := cl.Length(); !bf.config.SkipHeaders && cll > LOGPLEX_MAX_LENGTH {
			length += NewLogplexBatchFormatter(splitLine(cl), bf.config).Length()
		} else {
			length += NewLogplexLineFormatter(cl, bf.config).Length()
		}
	}
	return
}

// Implements the io.Reader interface
func (bf *LogplexBatchFormatter) Read(p []byte) (n int, err error) {
	var copied int

	for n < len(p) && err == nil {

		// There is no currentFormatter, so figure one out
		if bf.curFormatter == nil {
			currentLine := bf.b.logLines[bf.curLogLine]

			// The current log line has headers (so we assume it's length is okay)
			// but if the not and the log line is too long, we need to split it to do
			// this we make a sub batch.
			if cll := currentLine.Length(); !bf.config.SkipHeaders && cll > LOGPLEX_MAX_LENGTH {
				// Wrap the sub batch in a formatter
				bf.curFormatter = NewLogplexBatchFormatter(splitLine(currentLine), bf.config)
			} else {
				bf.curFormatter = NewLogplexLineFormatter(currentLine, bf.config)
			}
		}

		for n < len(p) && err == nil {
			copied, err = bf.curFormatter.Read(p[n:])
			n += copied
		}

		// if we're not at the last line and the err is io.EOF
		// then we're not done reading, so ditch the current formatter
		// and move to the next log line
		if bf.curLogLine < (bf.b.MsgCount()-1) && err == io.EOF {
			err = nil
			bf.curLogLine += 1
			bf.curFormatter = nil
		}
	}

	return
}

// LogplexLineFormatter formats individual loglines into length prefixed
// rfc5424 messages via an io.Reader interface
type LogplexLineFormatter struct {
	headerPos, msgPos int    // Positions in the the parts of the log lines
	line              []byte // the raw line bytes
	header            string // the precomputed, length prefixed syslog frame header
}

// Returns a new LogplexLineFormatter wrapping the provided LogLine
func NewLogplexLineFormatter(ll LogLine, config *ShuttleConfig) *LogplexLineFormatter {
	var header string
	if config.SkipHeaders {
		header = fmt.Sprintf("%d ", len(ll.line))
	} else {
		header = fmt.Sprintf(config.syslogFrameHeaderFormat,
			config.lengthPrefixedSyslogFrameHeaderSize+len(ll.line),
			ll.when.UTC().Format(LOGPLEX_BATCH_TIME_FORMAT))
	}
	return &LogplexLineFormatter{
		line:   ll.line,
		header: header,
	}
}

func (llf *LogplexLineFormatter) Length() (lenth int) {
	return len(llf.header) + len(llf.line)
}

// Implements the io.Reader interface
// tries to fill p as full as possible before returning
func (llf *LogplexLineFormatter) Read(p []byte) (n int, err error) {
	for n < len(p) && err == nil {
		if llf.headerPos >= len(llf.header) {
			copied := copy(p[n:], llf.line[llf.msgPos:])
			llf.msgPos += copied
			n += copied
			if llf.msgPos >= len(llf.line) {
				err = io.EOF
			}
		} else {
			copied := copy(p[n:], llf.header[llf.headerPos:])
			llf.headerPos += copied
			n += copied
		}
	}
	return
}

func NewLogplexErrorFormatter(err errData, config ShuttleConfig) *LogplexLineFormatter {
	var what, code string

	switch err.eType {
	case errDrop:
		what = "dropped"
		code = "L12"
	case errLost:
		what = "lost"
		code = "L13"
	}

	msg := fmt.Sprintf("<172>%s %s heroku %s log-shuttle %s Error %s: %d messages %s since %s\n",
		config.Version,
		time.Now().UTC().Format(LOGPLEX_BATCH_TIME_FORMAT),
		config.Appname,
		config.Msgid,
		code,
		err.count,
		what,
		err.since.UTC().Format(LOGPLEX_BATCH_TIME_FORMAT))
	return &LogplexLineFormatter{
		line:   []byte(msg),
		header: fmt.Sprintf("%d ", len(msg)),
	}
}
