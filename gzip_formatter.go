package shuttle

import (
	"compress/gzip"
	"io"
	"io/ioutil"
	"net/http"
	"sync"
)

// a GzipFormatter is an HTTPFormatter that is built with a
// delegate HTTPFormatter but which compresses the request body
type GzipFormatter struct {
	delegate HTTPFormatter
	reader   *io.PipeReader
	writer   *io.PipeWriter
	once     *sync.Once
}

// NewGzipFormatter builds a new GzipFormatter with the supplied delegate
func NewGzipFormatter(delegate HTTPFormatter) *GzipFormatter {
	reader, writer := io.Pipe()
	f := &GzipFormatter{
		delegate: delegate,
		reader:   reader,
		writer:   writer,
		once:     new(sync.Once),
	}
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

	request.Body = ioutil.NopCloser(g)
	request.Header.Add("Content-Encoding", "gzip")
	return request, nil
}

func (g *GzipFormatter) Read(p []byte) (int, error) {
	g.once.Do(func() {
		go g.writeGzip()
	})
	return g.reader.Read(p)
}

func (g *GzipFormatter) Close() error {
	return g.reader.Close()
}
