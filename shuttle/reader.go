package shuttle

import (
	"bufio"
	"io"
	"time"

	metrics "github.com/rcrowley/go-metrics"
)

// Reader performs the reading of lines from an io.ReadCloser, encapsulating
// lines into a LogLine and emitting them on outbox
type Reader struct {
	outbox          chan<- LogLine
	delayTimeMetric metrics.Timer
}

// NewReader constructs a new reader that will use the provided outbox.
func NewReader(outbox chan<- LogLine, mRegistry metrics.Registry) Reader {
	return Reader{
		outbox:          outbox,
		delayTimeMetric: metrics.GetOrRegisterTimer("reader.line.delay.time", mRegistry),
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

		rdr.outbox <- logLine
		rdr.delayTimeMetric.Update(currentLogTime.Sub(lastLogTime))
		lastLogTime = currentLogTime
	}
}
