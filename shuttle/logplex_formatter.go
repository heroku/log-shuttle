package shuttle

import (
	"fmt"
	"io"
	"strconv"
	"time"
)

const (
	LOGPLEX_BATCH_TIME_FORMAT = "2006-01-02T15:04:05.000000+00:00" // The format of the timestamp
)

// LogplexBatchFormatter implements on io.Reader that returns Logplex formatted
// log lines.  Wraps log lines in length prefixed rfc5424 formatting, splitting
// them as necessary to config.MaxLineLength
type LogplexBatchFormatter struct {
	curFormatter int
	formatters   []Formatter
	headers      map[string]string
}

// Returns a new LogplexBatchFormatter wrapping the provided batch
func NewLogplexBatchFormatter(b Batch, eData []errData, config *ShuttleConfig) Formatter {
	bf := &LogplexBatchFormatter{
		formatters: make([]Formatter, 0, b.MsgCount()+len(eData)),
		headers:    make(map[string]string),
	}
	bf.headers["Content-Type"] = "application/logplex-1"

	//Process any errData that we were passed first so it's at the top of the batch
	for _, edata := range eData {
		bf.formatters = append(bf.formatters, NewLogplexErrorFormatter(edata, *config))
		switch edata.eType {
		case errDrop:
			bf.headers["Logshuttle-Drops"] = strconv.Itoa(edata.count)
		case errLost:
			bf.headers["Logshuttle-Lost"] = strconv.Itoa(edata.count)
		}
	}

	// Make all of the sub formatters
	for cli := 0; cli < len(b.logLines); cli++ {
		cl := b.logLines[cli]
		if cll := len(cl.line); !config.SkipHeaders && cll > config.MaxLineLength {
			bf.formatters = append(bf.formatters, NewLogplexBatchFormatter(splitLine(cl, config.MaxLineLength), make([]errData, 0), config))
		} else {
			bf.formatters = append(bf.formatters, NewLogplexLineFormatter(cl, config))
		}
	}

	// Take the msg count after the formatters are created so we have the right count
	bf.headers["Logplex-Msg-Count"] = strconv.Itoa(bf.MsgCount())

	return bf
}

func (bf *LogplexBatchFormatter) Headers() map[string]string {
	return bf.headers
}

// The msgcount of the wrapped batch. We itterate over the sub forwarders to
// determine final msgcount
func (bf *LogplexBatchFormatter) MsgCount() (msgCount int) {
	for _, f := range bf.formatters {
		msgCount += f.MsgCount()
	}
	return
}

//Splits the line into a batch of loglines of max(mll) lengths
func splitLine(ll LogLine, mll int) Batch {
	l := ll.Length()
	batch := NewBatch(int(l/mll) + 1)
	for i := 0; i < l; i += mll {
		t := i + mll
		if t > l {
			t = l
		}
		batch.Add(LogLine{line: ll.line[i:t], when: ll.when})
	}
	return batch
}

func (bf *LogplexBatchFormatter) ContentLength() (length int64) {
	for _, f := range bf.formatters {
		length += f.ContentLength()
	}
	return
}

// Implements the io.Reader interface
func (bf *LogplexBatchFormatter) Read(p []byte) (n int, err error) {
	var copied int

	for n < len(p) && err == nil {
		copied, err = bf.formatters[bf.curFormatter].Read(p[n:])
		n += copied

		// if we're not at the last formatter and the err is io.EOF
		// then we're not done reading, so ditch the current formatter
		// and move to the next log line
		if err == io.EOF && bf.curFormatter < (len(bf.formatters)-1) {
			err = nil
			bf.curFormatter += 1
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

func (llf *LogplexLineFormatter) ContentLength() (lenth int64) {
	return int64(len(llf.header) + len(llf.line))
}

func (llf *LogplexLineFormatter) MsgCount() int {
	return 1
}

func (llf *LogplexLineFormatter) Headers() map[string]string {
	return make(map[string]string)
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
