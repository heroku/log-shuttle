package main

import (
	"fmt"
	"io"
)

type NBatch struct {
	logLines []LogLine
}

func NewNBatch(capacity int) *NBatch {
	return &NBatch{logLines: make([]LogLine, 0, capacity)}
}

// Add a logline to the batch
func (nb *NBatch) Add(ll LogLine) {
	nb.logLines = append(nb.logLines, ll)
}

// The count of msgs in the batch
func (nb *NBatch) MsgCount() int {
	return len(nb.logLines)
}

type LogplexBatchFormatter struct {
	curLogLine   int // Current Log Line
	b            *NBatch
	curFormatter io.Reader // Current sub formatter
	config       *ShuttleConfig
}

func NewLogplexBatchFormatter(b *NBatch, config *ShuttleConfig) *LogplexBatchFormatter {
	return &LogplexBatchFormatter{b: b, config: config}
}

func (bf *LogplexBatchFormatter) MsgCount() (msgCount int) {
	for _, line := range bf.b.logLines {
		msgCount += 1 + int(len(line.line)/LOGPLEX_MAX_LENGTH)
	}
	return
}

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

type LogplexLineFormatter struct {
	totalPos, headerPos, msgPos int // Positions in the the parts of the log lines
	headerLength, msgLength     int // Header and Message Lengths
	ll                          LogLine
	header                      string
}

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
