package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"

	"github.com/nu7hatch/gouuid"
	"github.com/bmizerany/lpx"
)

const (
	JSON_MARSHAL_FORMAT = "at=NewElasticSearchBatchFormatter type=%s error=%q errtype=\"%T\"\n"
	LPX_FAIL_FORMAT = "at=NewElasticSearchDocument error=%q errtype=\"%T\"\n"
	PRIVAL_FACILITY_FORMAT = "at=parsePrival facility-index=%d error=%s\n"
	PRIVAL_INVALID_FORMAT = "at=parsePrival raw=\"%s\" error=%s\n"
	SEARCH_INDEX_FAIL_FORMAT = "at=NewElasticSearchBatch error=%q errtype=\"%T\"\n"
)

// TODO(apg): This sucks
var BatchError = errors.New("Invalid batch")

// ElasticSearchBatchFormatter implements io.Reader that returns
// JSON suitable for indexes using the _bulk endpoints of ElasticSearch.
type ElasticSearchBatchFormatter struct {
	curFormatter int
	formatters []Formatter
	headers map[string]string // TODO: Maybe not needed?
}

// Returns an ElasticSearchBatchFormatter wrapping the provided batch
func NewElasticSearchBatchFormatter(b Batch, eData []errData, config *ShuttleConfig) (*ElasticSearchBatchFormatter, error) {
	bf := &ElasticSearchBatchFormatter{
		formatters: make([]Formatter, 0, b.MsgCount()),
	  headers: make(map[string]string),
	}

	// TODO: set a content-type? Not clear if this is needed or it's always assumed to be application/json
	bf.headers["Content-Type"] = "application/json"

	for cli := 0; cli < len(b.logLines); cli++ {
		cl := b.logLines[cli]
    f, err := NewElasticSearchIndexFormatter(cl, config, b.UUID, cli)
		if err != nil {
			ErrLogger.Printf(SEARCH_INDEX_FAIL_FORMAT, err)
		} else {
			bf.formatters = append(bf.formatters, f)
    }
	}

	if len(bf.formatters) > 0 {
		return bf, nil
	} else {
		return nil, BatchError
	}
}

func (bf *ElasticSearchBatchFormatter) Headers() map[string]string {
	return bf.headers
}

// The msgcount of the wrapped batch.
func (bf *ElasticSearchBatchFormatter) MsgCount() int {
	return len(bf.formatters)
}

func (bf *ElasticSearchBatchFormatter) ContentLength() (length int64) {
	for _, f := range bf.formatters {
		length += f.ContentLength()
	}
	return
}

// Implements the io.Reader interface
func (bf *ElasticSearchBatchFormatter) Read(p []byte) (n int, err error) {
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

// ElasticSearchLineFormatter formats individual loglines into a pair
// of action_metadata, document attributes, for use by the /_bulk
// endpoint
type ElasticSearchIndexFormatter struct {
	actionPos, docPos int
	action []byte // serialized JSON
	document []byte // serialized JSON
}

type ElasticSearchIndexActionBody struct {
	Id string `json:"_id"`
	Timestamp string `json:"_timestamp,omitempty"` 
}

type ElasticSearchIndexAction struct {
	Index ElasticSearchIndexActionBody `json:"index"`
}

type ElasticSearchDocument struct {
	Facility      string `json:"facility"`
	Level         string `json:"level"`
	Version       int `json:"version,int"`
	Time          string `json:"time"`
	Hostname      string `json:"hostname"`
	Name          string `json:"name"`
	Procid        string `json:"procid"`
	Msgid         string `json:"msgid"`
  Msg           string `json:"msg"`
}

// Returns a new LogplexLineFormatter wrapping the provided LogLine
func NewElasticSearchIndexFormatter(ll LogLine, config *ShuttleConfig, batchId *uuid.UUID, index int) (f *ElasticSearchIndexFormatter, err error) {
	var mAction, mDocument []byte

	actionBody := ElasticSearchIndexActionBody{
		Id: fmt.Sprintf("%s:%d", batchId.String(), index),
		Timestamp: ll.when.UTC().Format(LOGPLEX_BATCH_TIME_FORMAT),
	}
	action := ElasticSearchIndexAction{actionBody}
	if mAction, err = json.Marshal(action); err != nil {
		ErrLogger.Printf(JSON_MARSHAL_FORMAT, "ElasticSearchIndexAction", err, err)
		f = nil
		return
	}

	document := NewElasticSearchDocument(ll, config)
	if mDocument, err = json.Marshal(document); err != nil {
		ErrLogger.Printf(JSON_MARSHAL_FORMAT, "ElasticSearchDocument", err, err)
		f = nil
		return
	}

	// Add a newline to mAction, which is needed to separate
	// the action and the document being inserted
	mAction = append(mAction, '\n')
  mDocument = append(mDocument, '\n')

	f = &ElasticSearchIndexFormatter{ action: mAction, document: mDocument }
	return
}

func (esf *ElasticSearchIndexFormatter) ContentLength() (lenth int64) {
	return int64(len(esf.action) + len(esf.document))
}

func (esf *ElasticSearchIndexFormatter) MsgCount() int {
	return 1
}

func (esf *ElasticSearchIndexFormatter) Headers() map[string]string {
	return make(map[string]string)
}

// Implements the io.Reader interface
// tries to fill p as full as possible before returning
func (esf *ElasticSearchIndexFormatter) Read(p []byte) (n int, err error) {
	for n < len(p) && err == nil {
		if esf.actionPos >= len(esf.action) {
			copied := copy(p[n:], esf.document[esf.docPos:])
			esf.docPos += copied
			n += copied
			if esf.docPos >= len(esf.document) {
				err = io.EOF
			}
		} else {
			copied := copy(p[n:], esf.action[esf.actionPos:])
			esf.actionPos += copied
			n += copied
		}
	}
	return
}

// Parses a Logplex formatted logline into an ElasticSearchDocument
// This is the data we submit to ElasticSearch for indexing
func NewElasticSearchDocument(ll LogLine, config *ShuttleConfig) *ElasticSearchDocument {
	line := ll.Header(config) + string(ll.line)

	r := lpx.NewReader(bytes.NewBuffer([]byte(line)))
	r.Next()
	if err := r.Err(); err != nil {
		ErrLogger.Printf(LPX_FAIL_FORMAT, err, err)

		return nil
	}

	facility, level, version := parsePrivalVersion(string(r.Header().PrivalVersion))

	return &ElasticSearchDocument{
		Facility: facility,
		Level: level,
		Version: version,
		Time: string(r.Header().Time),
		Hostname: string(r.Header().Hostname),
		Name: string(r.Header().Name),
		Procid: string(r.Header().Procid),
		Msgid: string(r.Header().Msgid),
		Msg: string(r.Bytes()),
	}
}

// TODO(apg): determine a better place for this stuff...
var SYSLOG_FACILITIES = [24]string{
	"kern",
	"user",
	"mail",
	"daemon",
	"auth",
	"syslog",
	"lpr",
	"news",
	"uucp",
	"clock",
	"authpriv",
	"ftp",
	"-", // unused
	"-", // unused
	"-", // unused
	"cron",
	"local0",
	"local1",
	"local2",
	"local3",
	"local4",
	"local5",
	"local6",
	"local7",
}

var SYSLOG_LEVELS = [8]string{
	"emerg",
	"alert",
	"crit",
	"err",
	"warning",
	"notice",
	"info",
	"debug",
}

var SYSLOG_PRIVER_REGEXP = regexp.MustCompile("<([0-9]{1,3})>([0-9])")

func parsePrivalVersion(raw string) (facility string, level string, version int) {
	if matches := SYSLOG_PRIVER_REGEXP.FindStringSubmatch(raw); len(matches) == 3 {
		primary, _ := strconv.Atoi(matches[1])
		if f := primary / 8; f < len(SYSLOG_FACILITIES) {
			facility = SYSLOG_FACILITIES[f]
		} else {
			ErrLogger.Printf(PRIVAL_FACILITY_FORMAT, f, "invalid facility index")
		}
		level = SYSLOG_LEVELS[primary % 8]
		version, _ = strconv.Atoi(matches[2])
	} else {
		ErrLogger.Printf(PRIVAL_INVALID_FORMAT, raw, "did not match")
	}

	return
}
