/**
 * Copyright (c) 2018 Salesforce
 * All rights reserved.
 * Licensed under the BSD 3-Clause license.
 * For full license text, see LICENSE.txt file in the repo root
 *   or https://opensource.org/licenses/BSD-3-Clause
 */

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
	// LogplexContentType is the content type logplex expects
	LogplexContentType = "application/logplex-1"
)

// LogplexBatchFormatter implements on io.Reader that returns Logplex formatted
// log lines.  Wraps log lines in length prefixed rfc5424 formatting, splitting
// them as necessary to config.MaxLineLength
type LogplexBatchFormatter struct {
	headers   http.Header
	stringURL string
	msgCount  int
	io.Reader
}

// NewLogplexBatchFormatter returns a new LogplexBatchFormatter wrapping the provided batch
func NewLogplexBatchFormatter(b Batch, eData []errData, config *Config) HTTPFormatter {
	bf := &LogplexBatchFormatter{
		headers:   make(http.Header),
		stringURL: config.LogsURL,
	}

	bf.headers.Add("Content-Type", LogplexContentType)
	bf.headers.Add("X-Request-Id", b.UUID)

	var r SubFormatter
	readers := make([]io.Reader, 0, b.MsgCount()+len(eData))

	// Process any errData that we were passed first so it's at the top of the batch
	for _, edata := range eData {
		switch edata.eType {
		case errDrop:
			bf.headers.Add("Logplex-Drop-Count", strconv.Itoa(edata.count))
		case errLost:
			bf.headers.Add("Logplex-Lost-Count", strconv.Itoa(edata.count))
		}

		r = NewLogplexErrorFormatter(edata, config)
		readers = append(readers, r)
		bf.msgCount += r.MsgCount()
	}

	// Process the logLine sub-batching them as necessary
	for _, l := range b.logLines {
		if config.InputFormat == InputFormatRaw && len(l.line) > config.MaxLineLength {
			r = NewLogplexBatchFormatter(splitLine(l, config.MaxLineLength), nil, config)
		} else {
			r = NewLogplexLineFormatter(l, config)
		}
		readers = append(readers, r)
		bf.msgCount += r.MsgCount()
	}

	// Take the msg count after the formatters are created so we have the right count
	bf.headers.Add("Logplex-Msg-Count", strconv.Itoa(bf.MsgCount()))

	// Dispatch reading of the body to an io.MultiReader
	bf.Reader = io.MultiReader(readers...)

	return bf
}

// Request returns a properly constructed *http.Request, complete with headers
// and ContentLength set.
func (bf *LogplexBatchFormatter) Request() (*http.Request, error) {
	u, user, pass, err := extractCredentials(bf.stringURL)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", u.String(), bf)
	if err != nil {
		return nil, err
	}

	// Assign headers before we potentially BasicAuth
	req.Header = bf.headers

	if user != "" || pass != "" {
		req.SetBasicAuth(user, pass)
	}

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

// LogplexLineFormatter formats individual loglines into length prefixed
// rfc5424 messages via an io.Reader interface
type LogplexLineFormatter struct {
	headerPos, msgPos int    // Positions in the the parts of the log lines
	line              []byte // the raw line bytes
	header            string // the precomputed, length prefixed syslog frame header
	inputFormat       int
}

// NewLogplexLineFormatter returns a new LogplexLineFormatter wrapping the provided LogLine
func NewLogplexLineFormatter(ll LogLine, config *Config) *LogplexLineFormatter {
	var header string
	switch config.InputFormat {
	case InputFormatRaw:
		//fmt.Sprintf induces an extra allocation
		header = strconv.Itoa(len(ll.line)+config.lengthPrefixedSyslogFrameHeaderSize) + " " +
			"<" + config.Prival + ">" + config.Version + " " +
			ll.when.UTC().Format(LogplexBatchTimeFormat) + " " +
			config.Hostname + " " +
			config.Appname + " " +
			config.Procid + " " +
			config.Msgid + " "
	case InputFormatLengthPrefixedRFC5424:
		//NOOP, the message should already be in the right format. *\o/*
		//TODO: should we ensure the message is in the right format?
	case InputFormatRFC5424:
		header = strconv.Itoa(len(ll.line)) + " "
	}

	return &LogplexLineFormatter{
		line:        ll.line,
		header:      header,
		inputFormat: config.InputFormat,
	}
}

// MsgCount is always 1 for a Line
func (llf *LogplexLineFormatter) MsgCount() int {
	return 1
}

// Reset the reader so that the log line can be re-read
func (llf *LogplexLineFormatter) Reset() {
	llf.headerPos = 0
	llf.msgPos = 0
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

// fourth space seperated field in the []byte
func fourthField(l []byte) string {
	var start, found int
	for end := 0; end < len(l); end++ {
		if l[end] == ' ' {
			found++
			switch found {
			case 3:
				start = end + 1
				continue
			case 4:
				return string(l[start:end])
			}
		}
	}
	return ""
}

// AppName returns the name of app name field based on the inputFormat
// For use in syslog framing
func (llf *LogplexLineFormatter) AppName() string {
	switch llf.inputFormat {
	case InputFormatRaw:
		return fourthField([]byte(llf.header))
	case InputFormatRFC5424:
		return fourthField(llf.line)
	}
	panic("Unknown input format, or can't get appname reliably for input format")
}

// NewLogplexErrorFormatter returns a LogplexLineFormatter for the error data.
// These can be used to inject error data into the log stream
func NewLogplexErrorFormatter(err errData, config *Config) *LogplexLineFormatter {
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
		line:        []byte(msg),
		header:      fmt.Sprintf("%d ", len(msg)),
		inputFormat: InputFormatRFC5424,
	}
}
