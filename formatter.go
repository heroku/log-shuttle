package shuttle

import (
	"io"
	"net/http"
)

// SubFormatter describes the interface the sub formatters needs to support.
// MsgCount returns the count of messages once formatted.
// CountentLength returns the byte count of the formatted messages.
type SubFormatter interface {
	MsgCount() int
	io.Reader
}

// HTTPFormatter is the interface that http outlets use to format a HTTP
// request.
// Request() returns a *http.Request ready to be handled by an outlet
type HTTPFormatter interface {
	Request() (*http.Request, error)
	SubFormatter
}

// NewHTTPFormatterFunc defines the function type for defining creating and
// returning a new Formatter
type NewHTTPFormatterFunc func(b Batch, eData []errData, config *Config) HTTPFormatter
