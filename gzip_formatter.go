package shuttle

import (
	"compress/gzip"
	"io"
	"net/http"
)

// a GzipFormatter is an HTTPFormatter that is built with a
// delegate HTTPFormatter but which compresses the request body
type GzipFormatter struct {
	delegate HTTPFormatter
	reader   *io.PipeReader
	writer   *io.PipeWriter
}

// NewGzipFormatter builds a new GzipFormatter with the supplied delegate
func NewGzipFormatter(delegate HTTPFormatter) *GzipFormatter {
	reader, writer := io.Pipe()
	f := &GzipFormatter{
		delegate: delegate,
		reader:   reader,
		writer:   writer,
	}
	go f.writeGzip()
	return f
}

func (g *GzipFormatter) writeGzip() {
	gw := gzip.NewWriter(g.writer)
	_, err := io.Copy(gw, g.delegate)
	gw.Close()
	if err != nil {
		g.writer.CloseWithError(err)
	} else {
		g.writer.Close()
	}
}

func (g *GzipFormatter) MsgCount() int {
	return g.delegate.MsgCount()
}

func (g *GzipFormatter) Request() (*http.Request, error) {
	request, err := g.delegate.Request()
	if err != nil {
		return request, err
	}
	request.Header.Add("Content-Encoding", "gzip")
	return request, nil
}

func (g *GzipFormatter) Read(p []byte) (int, error) {
	return g.reader.Read(p)
}

func (g *GzipFormatter) Close() error {
	return g.reader.Close()
}
