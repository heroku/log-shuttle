package shuttle

import (
	"fmt"
	"io"
	"net/http"
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
	formatters   []io.Reader
	headers      http.Header
	stringURL    string
	msgCount     int
}

// Returns a new LogplexBatchFormatter wrapping the provided batch as a HttpFormatter
func NewLogplexBatchFormatter(b Batch, eData []errData, config *ShuttleConfig) HttpFormatter {
	bf := &LogplexBatchFormatter{
		formatters: make([]io.Reader, 0, b.MsgCount()+len(eData)),
		headers:    make(http.Header),
		stringURL:  config.OutletURL(),
	}

	bf.headers.Add("Content-Type", "application/logplex-1")

	//Process any errData that we were passed first so it's at the top of the batch
	for _, edata := range eData {
		bf.formatters = append(bf.formatters, NewLogplexErrorFormatter(edata, *config))
		switch edata.eType {
		case errDrop:
			bf.headers.Add("Logshuttle-Drops", strconv.Itoa(edata.count))
		case errLost:
			bf.headers.Add("Logshuttle-Lost", strconv.Itoa(edata.count))
		}
	}

	var r MsgCountReader

	// Make all of the sub formatters
	for _, l := range b.logLines {
		if !config.SkipHeaders && len(l.line) > config.MaxLineLength {
			r = NewLogplexBatchFormatter(splitLine(l, config.MaxLineLength), make([]errData, 0), config)
		} else {
			r = NewLogplexLineFormatter(l, config)
		}
		bf.formatters = append(bf.formatters, r)
		bf.msgCount += MsgCounter(r).MsgCount()
	}

	// Take the msg count after the formatters are created so we have the right count
	bf.headers.Add("Logplex-Msg-Count", strconv.Itoa(bf.msgCount))

	return bf
}

func (bf *LogplexBatchFormatter) Request() (*http.Request, error) {
	req, err := http.NewRequest("POST", bf.stringURL, bf)
	if err != nil {
		return nil, err
	}

	req.ContentLength = bf.contentLength()
	req.Header = bf.headers

	return req, nil

}

// The msgcount of the wrapped batch. We itterate over the sub forwarders to
// determine final msgcount
func (bf *LogplexBatchFormatter) MsgCount() (msgCount int) {
	return bf.msgCount
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

func (bf *LogplexBatchFormatter) contentLength() (length int64) {
	for _, f := range bf.formatters {
		switch v := f.(type) {
		case *LogplexBatchFormatter:
			length += v.contentLength()
		case *LogplexLineFormatter:
			length += v.contentLength()
		}
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
	stringURL         string
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
		line:      ll.line,
		header:    header,
		stringURL: "",
	}
}

func (llf *LogplexLineFormatter) contentLength() (lenth int64) {
	return int64(len(llf.header) + len(llf.line))
}

func (llf *LogplexLineFormatter) MsgCount() int {
	return 1
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
