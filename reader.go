package shuttle

import (
	"bufio"
	"io"
	"time"

	"github.com/rcrowley/go-metrics"
)

// LogLineReader performs the reading of lines from an io.ReadCloser, encapsulating
// lines into a LogLine and emitting them on outbox
type LogLineReader struct {
	outbox    chan<- LogLine
	linesRead metrics.Counter
}

// NewLogLineReader constructs a new reader with it's own Outbox.
func NewLogLineReader(o chan<- LogLine, m metrics.Registry) LogLineReader {
	return LogLineReader{
		outbox:    o,
		linesRead: metrics.GetOrRegisterCounter("lines.read", m),
	}
}

// ReadLogLines reads lines from the Reader and returns with an error if there
// is an error
func (rdr LogLineReader) ReadLogLines(input io.Reader) error {
	rdrIo := bufio.NewReader(input)

	for {
		line, err := rdrIo.ReadBytes('\n')

		if len(line) > 0 {
			currentLogTime := time.Now()
			rdr.Enqueue(LogLine{line, currentLogTime})
		}

		if err != nil {
			return err
		}
	}
}

// Enqueue a single log line and increment the line counters
func (rdr LogLineReader) Enqueue(ll LogLine) {
	rdr.outbox <- ll
	rdr.linesRead.Inc(1)
}
