package main

import (
	"bytes"
	"fmt"
)

// A buffer suitable for posting with a http client
// keeps track of line's Write()n to the buffer
type Batch struct {
	lineCount int
	buffer    bytes.Buffer
}

func (b *Batch) LineCount() int {
	return b.lineCount
}

// Write a line to the batch, increment it's line counter
func (b *Batch) Write(line string) {
	fmt.Fprintf(&b.buffer, "%d %s", len(line), line)
	b.lineCount++
}

// Zero the line count and reset the internal buffer
func (b *Batch) Reset() {
	b.lineCount = 0
	b.buffer.Reset()
}

// Proxy to buffer
func (b *Batch) Len() int                         { return b.buffer.Len() }
func (b *Batch) Read(p []byte) (n int, err error) { return b.buffer.Read(p) }

// NoOpCloser
func (b *Batch) Close() error { return nil }
