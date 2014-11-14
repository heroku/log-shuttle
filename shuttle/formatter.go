package shuttle

import (
	"io"
	"net/http"
)

// Formatter is the interface type that outlets use
// Outlets have final say over the outlets content length,
// message count and any additional headers.
// Formatters implement io.Reader, which outlets can use to read the formatted
// batch
type HttpFormatter interface {
	Request(string) (*http.Request, error)
	MsgCount() int
	io.Reader
}

type NewHttpFormatterFunc func(b Batch, eData []errData, config *ShuttleConfig) HttpFormatter
