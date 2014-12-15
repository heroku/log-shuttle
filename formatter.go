package shuttle

import (
	"io"
	"net/http"
)

// SubFormatter formats a complete batch or a subsection of a batch. It may
// split lines in the batch as needed by the destination, making the MsgCount()
// of the formatter different from the MsgCount of the source batch. A
// formatter may emitt more (likely) or less bytes for a given LogLine than the
// actual Logline.
type SubFormatter interface {
	MsgCount() int // MsgCount is the number of messages after formatting
	io.Reader
}

// HTTPFormatter is the interface that http outlets use to format a HTTP
// request.
type HTTPFormatter interface {
	Request() (*http.Request, error) // Request() returns a *http.Request ready to be handled by an outlet
	SubFormatter
}

// NewHTTPFormatterFunc defines the function type for defining creating and
// returning a new Formatter
type NewHTTPFormatterFunc func(b Batch, eData []errData, config *Config) HTTPFormatter
