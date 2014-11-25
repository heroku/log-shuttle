package shuttle

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

const (
	// LogplexBatchTimeFormat is the format of timestamps as expected by Logplex
	LogplexBatchTimeFormat = "2006-01-02T15:04:05.000000+00:00"
)

// LogplexBatchFormatter implements on io.Reader that returns Logplex formatted
// log lines.  Wraps log lines in length prefixed rfc5424 formatting, splitting
// them as necessary to config.MaxLineLength
type LogplexBatchFormatter struct {
	curFormatter  int
	headers       http.Header
	stringURL     string
	msgCount      int
	contentLength int64
	io.Reader
}

// NewLogplexBatchFormatter returns a new LogplexBatchFormatter wrapping the provided batch
func NewLogplexBatchFormatter(b Batch, eData []errData, config *Config) HTTPFormatter {
	bf := &LogplexBatchFormatter{
		headers:   make(http.Header),
		stringURL: config.OutletURL(),
	}

	bf.headers.Add("Content-Type", "application/logplex-1")

	var r SubFormatter
	readers := make([]io.Reader, 0, b.MsgCount()+len(eData))

	// Process any errData that we were passed first so it's at the top of the batch
	for _, edata := range eData {
		switch edata.eType {
		case errDrop:
			bf.headers.Add("Logshuttle-Drops", strconv.Itoa(edata.count))
		case errLost:
			bf.headers.Add("Logshuttle-Lost", strconv.Itoa(edata.count))
		}

		r = NewLogplexErrorFormatter(edata, *config)
		readers = append(readers, io.Reader(r))
		bf.msgCount += r.MsgCount()
		bf.contentLength += r.ContentLength()
	}

	// Process the logLine sub-batching them as necessary
	for _, l := range b.logLines {
		if !config.SkipHeaders && len(l.line) > config.MaxLineLength {
			r = NewLogplexBatchFormatter(splitLine(l, config.MaxLineLength), make([]errData, 0), config)
		} else {
			r = NewLogplexLineFormatter(l, config)
		}
		readers = append(readers, io.Reader(r))
		bf.msgCount += r.MsgCount()
		bf.contentLength += r.ContentLength()
	}

	// Take the msg count after the formatters are created so we have the right count
	bf.headers.Add("Logplex-Msg-Count", strconv.Itoa(bf.MsgCount()))

	// Dispatch reading the body to an io.MultiReader
	bf.Reader = io.MultiReader(readers...)

	return bf
}

// Request returns a properly constructed *http.Request, complete with headers
// and ContentLength set.
func (bf *LogplexBatchFormatter) Request() (*http.Request, error) {
	req, err := http.NewRequest("POST", bf.stringURL, bf)
	if err != nil {
		return nil, err
	}

	req.ContentLength = bf.ContentLength()
	req.Header = bf.headers

	return req, nil
}

// MsgCount of the wrapped batch.
func (bf *LogplexBatchFormatter) MsgCount() int {
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

// ContentLength of the batch as formatted by the Formatter
func (bf *LogplexBatchFormatter) ContentLength() int64 {
	return bf.contentLength
}

// LogplexLineFormatter formats individual loglines into length prefixed
// rfc5424 messages via an io.Reader interface
type LogplexLineFormatter struct {
	headerPos, msgPos int    // Positions in the the parts of the log lines
	line              []byte // the raw line bytes
	header            string // the precomputed, length prefixed syslog frame header
}

// NewLogplexLineFormatter returns a new LogplexLineFormatter wrapping the provided LogLine
func NewLogplexLineFormatter(ll LogLine, config *Config) *LogplexLineFormatter {
	var header string
	if config.SkipHeaders {
		header = fmt.Sprintf("%d ", len(ll.line))
	} else {
		header = fmt.Sprintf(config.syslogFrameHeaderFormat,
			config.lengthPrefixedSyslogFrameHeaderSize+len(ll.line),
			ll.when.UTC().Format(LogplexBatchTimeFormat))
	}
	return &LogplexLineFormatter{
		line:   ll.line,
		header: header,
	}
}

// ContentLength of the line, as formatted by this formatter
func (llf *LogplexLineFormatter) ContentLength() int64 {
	return int64(len(llf.header) + len(llf.line))
}

// MsgCount is always 1 for a Line
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

// NewLogplexErrorFormatter returns a LogplexLineFormatter for the error data.
// These can be used to inject error data into the log stream
func NewLogplexErrorFormatter(err errData, config Config) *LogplexLineFormatter {
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
		time.Now().UTC().Format(LogplexBatchTimeFormat),
		config.Appname,
		config.Msgid,
		code,
		err.count,
		what,
		err.since.UTC().Format(LogplexBatchTimeFormat))
	return &LogplexLineFormatter{
		line:   []byte(msg),
		header: fmt.Sprintf("%d ", len(msg)),
	}
}
