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

type LogplexBatchReader struct {
	curLogLine        int // Current Log Line
	b                 *NBatch
	currentLineReader *LogplexLineReader
	config            *ShuttleConfig
}

func NewLogplexBatchReader(b *NBatch, config *ShuttleConfig) *LogplexBatchReader {
	return &LogplexBatchReader{b: b, config: config}
}

func (br *LogplexBatchReader) Read(p []byte) (n int, err error) {
	if br.currentLineReader == nil {
		br.currentLineReader = NewLogplexLineReader(br.b.logLines[br.curLogLine], br.config)
	}

	n, err = br.currentLineReader.Read(p)

	// if we're not at the last line and the err is io.EOF
	// then we're not done reading, so ditch the current line reader
	// and move to the next log line
	if br.curLogLine < (br.b.MsgCount()-1) && err == io.EOF {
		err = nil
		br.curLogLine += 1
		br.currentLineReader = nil
	}

	return
}

type LogplexLineReader struct {
	totalPos, headerPos, msgPos int // Positions in the the parts of the log lines
	headerLength, msgLength     int // Header and Message Lengths
	ll                          LogLine
	header                      string
}

func NewLogplexLineReader(ll LogLine, config *ShuttleConfig) *LogplexLineReader {
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
	return &LogplexLineReader{ll: ll, header: header, msgLength: msgLength, headerLength: len(header)}
}

func (llf *LogplexLineReader) Read(p []byte) (n int, err error) {
	if llf.totalPos >= llf.headerLength {
		n = copy(p, llf.ll.line[llf.msgPos:])
		llf.msgPos += n
		llf.totalPos += n
		if llf.msgPos >= llf.msgLength {
			err = io.EOF
		}
		return
	} else {
		n = copy(p, llf.header[llf.headerPos:])
		llf.headerPos += n
		llf.totalPos += n
		return
	}
}
