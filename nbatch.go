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
	c                    int
	b                    *NBatch
	currentLineFormatter *LogplexLineFormatter
	config               *ShuttleConfig
}

func NewLogplexBatchReader(b *NBatch, config *ShuttleConfig) *LogplexBatchReader {
	return &LogplexBatchReader{b: b, config: config}
}

func (br *LogplexBatchReader) Read(p []byte) (n int, err error) {
	if br.c >= br.b.MsgCount() {
		return 0, io.EOF
	}

	if br.currentLineFormatter == nil {
		br.currentLineFormatter = NewLogplexLineFormatter(br.b.logLines[br.c], br.config)
	}

	b, err := br.currentLineFormatter.Read(p)
	if err == io.EOF {
		br.currentLineFormatter = nil
		br.c += 1
		return b, nil
	}
	return b, err

}

type LogplexLineFormatter struct {
	totalCounter, headerCounter, msgCounter int // Total Counter, Header Counter, Message Counter
	headerLength, msgLength                 int // Header and Message Lengths
	ll                                      LogLine
	header                                  string
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
	if llf.totalCounter >= llf.headerLength {
		n = copy(p, llf.ll.line[llf.msgCounter:])
		llf.msgCounter += n
		llf.totalCounter += n
		if llf.msgCounter >= llf.msgLength {
			err = io.EOF
		}
		return
	} else {
		n = copy(p, llf.header[llf.headerCounter:])
		llf.headerCounter += n
		llf.totalCounter += n
		return
	}
}
