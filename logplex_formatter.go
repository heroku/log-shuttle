package main

import (
	"fmt"
	"io"
)

// A LogplexBatchWithHeadersFormatter implements an io.Reader that returns
// Logplex formatted log lines.  Assumes the loglines in the batch are already
// syslog rfc5424 formatted and less than LOGPLEX_MAX_LENGTH
type LogplexBatchWithHeadersFormatter struct {
	curLogLine int // Current Log Line
	curLinePos int // Current position in the current line
	b          *NBatch
	config     *ShuttleConfig
}

// Returns a new LogplexBatchWithHeadersFormatter, wrapping the existing Batch
func NewLogplexBatchWithHeadersFormatter(b *NBatch, config *ShuttleConfig) *LogplexBatchWithHeadersFormatter {
	return &LogplexBatchWithHeadersFormatter{b: b, config: config}
}

// Return the number of formatted messages in the batch.
func (bf *LogplexBatchWithHeadersFormatter) MsgCount() (msgCount int) {
	return bf.b.MsgCount()
}

// io.Reader implementation Returns io.EOF when done.
func (bf *LogplexBatchWithHeadersFormatter) Read(p []byte) (n int, err error) {
	cl := bf.b.logLines[bf.curLogLine].line
	if bf.curLinePos == 0 {
		n = copy(p[n:], fmt.Sprintf("%d ", len(cl)))
	}
	copied := copy(p[n:], cl[bf.curLinePos:])
	bf.curLinePos += copied
	n += copied
	if bf.curLinePos >= len(cl) {
		bf.curLinePos = 0
		bf.curLogLine += 1
	}
	if bf.curLogLine >= bf.b.MsgCount() {
		err = io.EOF
	}
	return
}

// LogplexBatchFormatter implements on io.Reader that returns Logplex formatted
// log lines.  Wraps log lines in length prefixed rfc5424 formatting, splitting
// them as necessary to LOGPLEX_MAX_LENGTH
type LogplexBatchFormatter struct {
	curLogLine   int // Current Log Line
	b            *NBatch
	curFormatter io.Reader // Current sub formatter
	config       *ShuttleConfig
}

// Returns a new LogplexBatchFormatter wrapping the provided batch
func NewLogplexBatchFormatter(b *NBatch, config *ShuttleConfig) *LogplexBatchFormatter {
	return &LogplexBatchFormatter{b: b, config: config}
}

// The msgcount of the wrapped batch. Because it splits lines at
// LOGPLEX_MAX_LENGTH this may be different from the actual MsgCount of the
// batch
func (bf *LogplexBatchFormatter) MsgCount() (msgCount int) {
	for _, line := range bf.b.logLines {
		msgCount += 1 + int(len(line.line)/LOGPLEX_MAX_LENGTH)
	}
	return
}

// Implements the io.Reader interface
func (bf *LogplexBatchFormatter) Read(p []byte) (n int, err error) {
	// There is no currentFormatter, so figure one out
	if bf.curFormatter == nil {
		currentLine := bf.b.logLines[bf.curLogLine]

		// The current line is too long, so make a sub batch
		if cll := currentLine.Length(); cll > LOGPLEX_MAX_LENGTH {
			subBatch := NewNBatch(int(cll/LOGPLEX_MAX_LENGTH) + 1)

			for i := 0; i < cll; i += LOGPLEX_MAX_LENGTH {
				target := i + LOGPLEX_MAX_LENGTH
				if target > cll {
					target = cll
				}

				subBatch.Add(LogLine{line: currentLine.line[i:target], when: currentLine.when})
			}

			// Wrap the sub batch in a formatter
			bf.curFormatter = NewLogplexBatchFormatter(subBatch, bf.config)
		} else {
			bf.curFormatter = NewLogplexLineFormatter(currentLine, bf.config)
		}
	}

	copied := 0
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

	return
}

// LogplexLineFormatter formats individual loglines into length prefixed
// rfc5424 messages via an io.Reader interface
type LogplexLineFormatter struct {
	totalPos, headerPos, msgPos int // Positions in the the parts of the log lines
	headerLength, msgLength     int // Header and Message Lengths
	ll                          LogLine
	header                      string
}

// Returns a new LogplexLineFormatter wrapping the provided LogLine
func NewLogplexLineFormatter(ll LogLine, config *ShuttleConfig) *LogplexLineFormatter {
	syslogFrameHeader := fmt.Sprintf("<%s>%s %s %s %s %s %s ",
		config.Prival,
		config.Version,
		ll.when.UTC().Format(BATCH_TIME_FORMAT),
		config.Hostname,
		config.Appname,
		config.Procid,
		config.Msgid,
	)
	msgLength := len(ll.line)
	header := fmt.Sprintf("%d %s", len(syslogFrameHeader)+msgLength, syslogFrameHeader)
	return &LogplexLineFormatter{ll: ll, header: header, msgLength: msgLength, headerLength: len(header)}
}

// Implements the io.Reader interface
func (llf *LogplexLineFormatter) Read(p []byte) (n int, err error) {
	for n < len(p) && err == nil {
		if llf.totalPos >= llf.headerLength {
			copied := copy(p[n:], llf.ll.line[llf.msgPos:])
			llf.msgPos += copied
			llf.totalPos += copied
			n += copied
			if llf.msgPos >= llf.msgLength {
				err = io.EOF
			}
		} else {
			copied := copy(p[n:], llf.header[llf.headerPos:])
			llf.headerPos += copied
			llf.totalPos += copied
			n += copied
		}
	}
	return
}
