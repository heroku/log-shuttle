package shuttle

import (
	"io"
)

// Formatter is the interface type that outlets use
// Outlets have final say over the outlets content length,
// message count and any additional headers.
// Formatters implement io.Reader, which outlets can use to read the formatted
// batch
type Formatter interface {
	ContentLength() int64
	MsgCount() int
	Headers() map[string]string
	io.Reader
}

// NewFormatterFunc defines the function type for defining creating and
// returning a new Formatter
type NewFormatterFunc func(b Batch, eData []errData, config *Config) Formatter
