package shuttle

import (
	"bufio"
	"io"
	"sync"
	"time"

	"github.com/rcrowley/go-metrics"
)

// LogLineReader performs the reading of lines from an io.ReadCloser, encapsulating
// lines into a LogLine and emitting them on outbox
type LogLineReader struct {
	input     io.ReadCloser // The input to read from
	out       chan<- Batch  // Where to send batches
	close     chan struct{}
	batchSize int           // size of new batches
	timeOut   time.Duration // batch timeout
	timer     *time.Timer   // timer to actually enforce timeout
	drops     *Counter
	drop      bool // Should we drop or block

	linesRead         metrics.Counter
	linesBatchedCount metrics.Counter
	linesDroppedCount metrics.Counter
	batchFillTime     metrics.Timer

	mu sync.Mutex // protects access to below
	b  Batch
}

// NewLogLineReader constructs a new reader with it's own Outbox.
func NewLogLineReader(input io.ReadCloser, s *Shuttle) *LogLineReader {
	t := time.NewTimer(time.Second)
	t.Stop() // we only need a timer running when we actually have log lines in the batch

	ll := LogLineReader{
		input:     input,
		out:       s.Batches,
		close:     make(chan struct{}),
		batchSize: s.config.BatchSize,
		timeOut:   s.config.WaitDuration,
		timer:     t,
		drops:     s.Drops,
		drop:      s.config.Drop,

		linesRead:         metrics.GetOrRegisterCounter("lines.read", s.MetricsRegistry),
		linesBatchedCount: metrics.GetOrRegisterCounter("lines.batched", s.MetricsRegistry),
		linesDroppedCount: metrics.GetOrRegisterCounter("lines.dropped", s.MetricsRegistry),
		batchFillTime:     metrics.GetOrRegisterTimer("batch.fill", s.MetricsRegistry),

		b: NewBatch(s.config.BatchSize),
	}

	go ll.expireBatches()

	return &ll
}

func (rdr *LogLineReader) expireBatches() {
	for {
		select {
		case <-rdr.close:
			return

		case <-rdr.timer.C:
			rdr.mu.Lock()
			rdr.deliverOrDropCurrent()
			rdr.mu.Unlock()
		}
	}
}

//Close the reader for input
func (rdr *LogLineReader) Close() error {
	return rdr.input.Close()
}

// ReadLines from the input created for. Return any errors
// blocks until the underlying reader is closed
func (rdr *LogLineReader) ReadLines() error {
	rdrIo := bufio.NewReader(rdr.input)

	for {
		line, err := rdrIo.ReadBytes('\n')

		if len(line) > 0 {
			currentLogTime := time.Now()
			rdr.linesRead.Inc(1)
			rdr.mu.Lock()
			if full := rdr.b.Add(LogLine{line, currentLogTime}); full {
				rdr.deliverOrDropCurrent()
			}
			if rdr.b.MsgCount() == 1 { // First line so restart the timer
				rdr.timer.Reset(rdr.timeOut)
			}
			rdr.mu.Unlock()
		}

		if err != nil {
			rdr.mu.Lock()
			rdr.deliverOrDropCurrent()
			rdr.mu.Unlock()
			close(rdr.close)
			return err
		}
	}
}

// Should only be called when rdr.mu is held
func (rdr *LogLineReader) deliverOrDropCurrent() {
	rdr.timer.Stop()
	// There is the possibility of a new batch being expired while this is happening.
	// so guard against queueing up an empty batch
	if c := rdr.b.MsgCount(); c > 0 {
		if rdr.drop {
			select {
			case rdr.out <- rdr.b:
				rdr.linesBatchedCount.Inc(int64(c))
			default:
				rdr.linesDroppedCount.Inc(int64(c))
				rdr.drops.Add(c)
			}
		} else {
			rdr.out <- rdr.b
			rdr.linesBatchedCount.Inc(int64(c))
		}
		rdr.b = NewBatch(rdr.batchSize)
	}
}
