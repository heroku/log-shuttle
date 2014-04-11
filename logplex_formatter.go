package main

import (
	"fmt"
	"io"
)

const (
	LOGPLEX_MAX_LENGTH = 10000 // It's actually 10240, but leave enough space for headers
	//TODO: Move this and the stuff that uses it to LogLine
	BATCH_TIME_FORMAT = "2006-01-02T15:04:05.000000+00:00"
)

type MsgCounter interface {
	MsgCount() int
}

// A LogplexBatchWithHeadersFormatter implements an io.Reader that returns
// Logplex formatted log lines.  Assumes the loglines in the batch are already
// syslog rfc5424 formatted and less than LOGPLEX_MAX_LENGTH
type LogplexBatchWithHeadersFormatter struct {
	curLogLine   int // Current Log Line
	curLinePos   int // Current position in the current line
	curPrefixPos int
	b            *Batch
	config       *ShuttleConfig
}

// Returns a new LogplexBatchWithHeadersFormatter, wrapping the existing Batch
func NewLogplexBatchWithHeadersFormatter(b *Batch, config *ShuttleConfig) *LogplexBatchWithHeadersFormatter {
	return &LogplexBatchWithHeadersFormatter{b: b, config: config}
}

// Return the number of formatted messages in the batch.
func (bf *LogplexBatchWithHeadersFormatter) MsgCount() (msgCount int) {
	return bf.b.MsgCount()
}

// io.Reader implementation Returns io.EOF when done.
func (bf *LogplexBatchWithHeadersFormatter) Read(p []byte) (n int, err error) {
	//fmt.Printf("cll: %d\n", bf.curLogLine)
	//fmt.Printf("bf.b: %+v\n", bf.b)
	//fmt.Printf("len(p): %d\n", len(p))
	if bf.curLogLine > (bf.b.MsgCount() - 1) {
		return 0, io.EOF
	}
	cl := bf.b.logLines[bf.curLogLine].line
	prefix := fmt.Sprintf("%d ", len(cl))
	//fmt.Printf("len(prefix): %d\n", len(prefix))
	//fmt.Printf("curPrefixPos: %d\n", bf.curPrefixPos)
	if (len(prefix) - 1) > bf.curPrefixPos {
		copied := copy(p, prefix[bf.curPrefixPos:])
		n += copied
		bf.curPrefixPos += copied
		//fmt.Printf("prefix p: %s\n", string(p))
		//fmt.Printf("prefix n: %d\n", n)
		if n >= len(p) {
			//fmt.Println("prefix returning")
			return
		}
	}
	//fmt.Printf("post lp n: %d\n", n)
	copied := copy(p[n:], cl[bf.curLinePos:])
	bf.curLinePos += copied
	//fmt.Printf("curLinePos: %d\n", bf.curLinePos)
	//fmt.Printf("len(cl): %d\n", len(cl))
	n += copied
	if bf.curLinePos >= len(cl) {
		bf.curLinePos = 0
		bf.curPrefixPos = 0
		bf.curLogLine += 1
	}
	//fmt.Printf("curLogLine: %d\n", bf.curLogLine)
	//fmt.Printf("b.MsgCount(): %d\n", bf.b.MsgCount())
	if bf.curLogLine > (bf.b.MsgCount() - 1) {
		err = io.EOF
	}
	//fmt.Printf("n: %d, err: %v\n", n, err)
	//fmt.Printf("p: %s\n", string(p))
	return
}

// LogplexBatchFormatter implements on io.Reader that returns Logplex formatted
// log lines.  Wraps log lines in length prefixed rfc5424 formatting, splitting
// them as necessary to LOGPLEX_MAX_LENGTH
type LogplexBatchFormatter struct {
	curLogLine   int // Current Log Line
	b            *Batch
	curFormatter io.Reader // Current sub formatter
	config       *ShuttleConfig
}

// Returns a new LogplexBatchFormatter wrapping the provided batch
func NewLogplexBatchFormatter(b *Batch, config *ShuttleConfig) *LogplexBatchFormatter {
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
			subBatch := NewBatch(int(cll/LOGPLEX_MAX_LENGTH) + 1)

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
	headerPos, msgPos int // Positions in the the parts of the log lines
	line              []byte
	header            string
}

// Returns a new LogplexLineFormatter wrapping the provided LogLine
func NewLogplexLineFormatter(ll LogLine, config *ShuttleConfig) *LogplexLineFormatter {
	return &LogplexLineFormatter{
		line: ll.line,
		header: fmt.Sprintf(config.syslogFrameHeaderFormat,
			config.lengthPrefixedSyslogFrameHeaderSize+len(ll.line),
			ll.when.UTC().Format(BATCH_TIME_FORMAT)),
	}
}

// Implements the io.Reader interface
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
