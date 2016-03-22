package util

import (
	"compress/bzip2"
	"compress/gzip"
	"io"
	"os"
	"runtime"
	"strings"
)

type ReadCloser struct {
	r       io.Reader
	fp      *os.File
	gz      *gzip.Reader
	isOpen  bool
	isGzip  bool
	isBzip2 bool
}

// Opens for reading a plain file, gzip'ed file (extension .gz), or bzip2'ed file (extension .bz2)
func Open(filename string) (rc *ReadCloser, err error) {
	fp, err := os.Open(filename)
	if err != nil {
		return nil, err
	}

	if strings.HasSuffix(filename, ".gz") {
		r := &ReadCloser{
			fp:     fp,
			isGzip: true,
			isOpen: true,
		}
		r.gz, err = gzip.NewReader(fp)
		if err != nil {
			fp.Close()
			return nil, err
		}
		r.r = r.gz
		runtime.SetFinalizer(r, (*ReadCloser).Close)
		return r, nil
	}

	if strings.HasSuffix(filename, ".bz2") {
		r := &ReadCloser{
			fp:      fp,
			r:       bzip2.NewReader(fp),
			isBzip2: true,
			isOpen:  true,
		}
		runtime.SetFinalizer(r, (*ReadCloser).Close)
		return r, nil
	}

	r := &ReadCloser{
		fp:      fp,
		r:       fp,
		isOpen:  true,
	}
	runtime.SetFinalizer(r, (*ReadCloser).Close)
	return r, nil
}

func (rc ReadCloser) Read(p []byte) (n int, err error) {
	if !rc.isOpen {
		panic("ReadCloser is closed")
	}
	return rc.r.Read(p)
}

func (rc ReadCloser) Close() (err error) {
	if rc.isOpen {
		rc.isOpen = false

		if rc.isGzip {
			err = rc.gz.Close()
		}

		e := rc.fp.Close()
		if err == nil {
			err = e
		}
	}
	return
}
