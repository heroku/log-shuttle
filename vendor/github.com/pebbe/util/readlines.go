package util

import (
	"compress/bzip2"
	"compress/gzip"
	"io"
	"os"
	"strings"
)

/*
### Opening a LinesReader ###

Example 1:

    r, err := util.NewLinesReaderFromFile(filename)
    util.CheckErr(err)

Example 2:

    r = util.NewLinesReaderFromReader(os.Stdin)

Example 3:

    f, e := os.Open(filename)
    util.CheckErr(e)
    defer f.Close()

    rd, e := gzip.NewReader(f)
    util.CheckErr(e)
    defer rd.Close()

    r = util.NewLinesReaderFromReader(rd)

Since NewLinesReaderFromFile also handles .gz and .bz2 directly,
you can use example 1 instead of example 3 for these files.


### Using a LinesReader ###

Example 1:

    for line := range r.ReadLines() {
        // do something with line
    }

Example 2:

    for line := range r.ReadLines() {
        // do something with line

        // if you need to stop before all lines are read:
        r.Break()
        break     // not needed at bottom of loop

        // do more things
    }
*/
type LinesReader struct {
	r         *Reader
	rz        *gzip.Reader
	bz2       io.Reader
	f         *os.File
	isOpen    bool
	isGzip    bool
	needClose bool
	interrupt chan bool
}
// Either plain text file, gzip'ed text file with name ending in .gz, or bzip2'ed text file with name ending in .bz2
func NewLinesReaderFromFile(filename string) (r *LinesReader, err error) {
	return NewLinesReaderSizeFromFile(filename, 0)
}

// Either plain text file, gzip'ed text file with name ending in .gz, or bzip2'ed text file with name ending in .bz2
func NewLinesReaderSizeFromFile(filename string, bufsize int) (r *LinesReader, err error) {
	r = &LinesReader{interrupt: make(chan bool)}
	r.f, err = os.Open(filename)
	if err != nil {
		return
	}
	if strings.HasSuffix(filename, ".gz") {
		r.rz, err = gzip.NewReader(r.f)
		if err != nil {
			r.f.Close()
			return
		}
		r.r = NewReaderSize(r.rz, bufsize)
		r.isGzip = true
	} else if strings.HasSuffix(filename, ".bz2") {
		r.bz2 = bzip2.NewReader(r.f)
		r.r = NewReader(r.bz2)
	} else {
		r.r = NewReader(r.f)
	}
	r.isOpen = true
	r.needClose = true
	return
}

func NewLinesReaderFromReader(rd io.Reader) (r *LinesReader) {
	return NewLinesReaderSizeFromReader(rd, 0)
}

func NewLinesReaderSizeFromReader(rd io.Reader, bufsize int) (r *LinesReader) {

	r = &LinesReader{interrupt: make(chan bool)}
	r.r = NewReaderSize(rd, bufsize)
	r.isOpen = true
	return
}

func (r *LinesReader) ReadLines() <-chan string {
	if !r.isOpen {
		panic("LinesReader is closed")
	}
	ch := make(chan string)
	go func() {
	ReadLinesLoop:
		for {
			if !r.isOpen {
				break ReadLinesLoop
			}
			s, err := r.r.ReadLineString()
			if err == io.EOF {
				r.close()
				break
			}
			if err != nil {
				panic(err)
			}
			if !r.isOpen && len(s) == 0 {
				break ReadLinesLoop
			}
			select {
			case ch <- s:
			case <-r.interrupt:
				break ReadLinesLoop
			}
		}
		r.close()
		close(ch)
	}()
	return ch
}

func (r *LinesReader) ReadLinesBytes() <-chan []byte {
	if !r.isOpen {
		panic("LinesReader is closed")
	}
	ch := make(chan []byte)
	go func() {
	ReadLinesLoop:
		for {
			if !r.isOpen {
				break ReadLinesLoop
			}
			line, err := r.r.ReadLine()
			if err == io.EOF {
				r.close()
				break
			}
			if err != nil {
				panic(err)
			}
			if !r.isOpen && len(line) == 0 {
				break ReadLinesLoop
			}
			s := make([]byte, len(line))
			copy(s, line)
			select {
			case ch <- s:
			case <-r.interrupt:
				break ReadLinesLoop
			}
		}
		r.close()
		close(ch)
	}()
	return ch
}

func (r *LinesReader) Break() {
	if r.isOpen {
		r.interrupt <- true
	}
}

func (r *LinesReader) close() {
	r.isOpen = false
	if r.needClose {
		r.needClose = false
		if r.isGzip {
			e := r.rz.Close()
			if e != nil {
				panic(e)
			}
		}
		e := r.f.Close()
		if e != nil {
			panic(e)
		}
	}
}
