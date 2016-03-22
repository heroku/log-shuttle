package util

import (
	"bytes"
	"io"
)

const (
	defaultReaderBufSize = 4096
	minReaderBufferSize  = 16
)

// Type implementing a robust line reader.
type Reader struct {
	rd  io.Reader
	buf []byte
	err error
	p   int // points to first unprocessed byte in buf
	n   int // points to location after last unprocessed byte in buf
	idx int // points to first \n or \r (EOL) in unprocessed part of buf
	// invariants:
	//   0 <= p <= n <= len(buf)
	//   if unprocessed bytes available: p < n
	//   if EOL found: p <= idx < n && p < n
	//   if EOL not found: idx == -1
}

// Create new reader with default buffer size.
func NewReader(rd io.Reader) *Reader {
	return NewReaderSize(rd, defaultReaderBufSize)
}

// Create new reader with specified buffer size.
func NewReaderSize(rd io.Reader, size int) *Reader {
	if size < minReaderBufferSize {
		size = minReaderBufferSize
	}
	r := Reader{
		buf: make([]byte, size),
		rd:  rd,
		p:   0,
		n:   2,
		idx: 0,
	}
	r.buf[0] = '\r' // in case first line is empty
	r.buf[1] = '\n' //
	return &r
}

// Same as ReadLine(), but returns string instead of []byte.
// Result stays valid.
func (r *Reader) ReadLineString() (line string, err error) {
	b, e := r.ReadLine()
	return string(b), e
}

// Read single line from Reader, without EOL.
// EOL can be any of: \n \r \r\n \n\r.
// Last line without EOL is allowed.
// Result is only valid until next call to ReadLine() or ReadLineString()
func (r *Reader) ReadLine() (line []byte, err error) {

	// Sets r.idx to first \n or \r, or -1 if not found.
	scan := func() {
		end := r.n
		i1 := bytes.IndexByte(r.buf[r.p:end], '\n')
		if i1 >= 0 {
			end = r.p + i1
		}
		i2 := bytes.IndexByte(r.buf[r.p:end], '\r')
		if i2 >= 0 {
			r.idx = r.p + i2
		} else if i1 >= 0 {
			r.idx = r.p + i1
		} else {
			r.idx = -1
		}
	}

	// Fills buffer if no \n or \r found and no error yet,
	// and then rescans for first \n or \r.
	fill := func() {
		if (r.idx < 0 || r.idx == r.n - 1) && r.err == nil {
			if r.p > 0 {
				// make room
				if r.p < r.n {
					copy(r.buf, r.buf[r.p:r.n])
				}
				r.n -= r.p
				r.p = 0
			}
			if r.n < len(r.buf) {
				var i int
				i, r.err = r.rd.Read(r.buf[r.n:])
				r.n += i
			}
			scan()
		}
	}

	lines := make([][]byte, 0)
	beginning := true
	for {

		fill()

		// if at beginning of line, skip EOL of previous line
		if beginning {
			var c byte
			for i := 0; i < 2; i++ {
				// 1 or 2 times \r or \n, but not \r\r or \n\n
				if r.idx == r.p && r.buf[r.p] != c {
					c = r.buf[r.p]
					r.p += 1
					scan() // reset r.idx
				} else {
					break
				}
			}
			fill()
			beginning = false
		}

		// if error and no more unscanned data, return saved parts if they exist, else return error
		if r.p == r.n && r.err != nil {
			if len(lines) > 0 {
				return buildline(lines, r.buf[0:0]), nil
			} else {
				return r.buf[0:0], r.err
			}
		}

		if r.p < r.n {
			if r.idx < 0 { // no EOL found
				buf := make([]byte, r.n-r.p)
				copy(buf, r.buf[r.p:r.n])
				lines = append(lines, buf)
				r.p = r.n
			} else { // EOL found
				p := r.p
				r.p = r.idx
				return buildline(lines, r.buf[p:r.p]), nil
			}
		}

	}
	panic("not reached")
}

func buildline(lines [][]byte, last []byte) []byte {
	if len(lines) == 0 {
		return last
	}

	i := len(last)
	for _, line := range lines {
		i += len(line)
	}
	buf := make([]byte, i)
	i = 0
	for _, line := range lines {
		copy(buf[i:], line)
		i += len(line)
	}
	copy(buf[i:], last)
	return buf
}
