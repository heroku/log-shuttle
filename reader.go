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
	outbox      chan<- LogLine
	lineCounter metrics.Counter
}

// NewLogLineReader constructs a new reader with it's own Outbox.
func NewLogLineReader(o chan<- LogLine, m metrics.Registry) LogLineReader {
	return LogLineReader{
		outbox:      o,
		lineCounter: metrics.GetOrRegisterCounter("reader.line.count", m),
	}
}

// ReadLogLines reads lines from the ReadCloser, increments it's counter and
// queues them up for batching
func (rdr LogLineReader) ReadLogLines(input io.ReadCloser) error {
	rdrIo := bufio.NewReader(input)

	for {
		line, err := rdrIo.ReadBytes('\n')
		currentLogTime := time.Now()

		if err != nil {
			input.Close()
			return err
		}

		logLine := LogLine{line, currentLogTime}

		rdr.outbox <- logLine
		rdr.lineCounter.Inc(1)
	}
}
