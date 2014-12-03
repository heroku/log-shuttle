package shuttle

import (
	"bufio"
	"io"
	"time"
)

// Reader performs the reading of lines from an io.ReadCloser, encapsulating
// lines into a LogLine and emitting them on Outbox
type Reader struct {
	Outbox chan LogLine
	stats  chan<- NamedValue
}

// NewReader constructs a new reader with it's own Outbox.
func NewReader(frontBuff int, stats chan<- NamedValue) Reader {
	return Reader{
		Outbox: make(chan LogLine, frontBuff),
		stats:  stats,
	}
}

func (rdr Reader) Read(input io.ReadCloser) error {
	rdrIo := bufio.NewReader(input)

	lastLogTime := time.Now()

	for {
		line, err := rdrIo.ReadBytes('\n')
		currentLogTime := time.Now()

		if err != nil {
			input.Close()
			return err
		}

		logLine := LogLine{line, currentLogTime}

		rdr.Outbox <- logLine
		rdr.stats <- NewNamedValue("reader.line.delay.time", currentLogTime.Sub(lastLogTime).Seconds())
		lastLogTime = currentLogTime
	}
}
