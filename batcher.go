package shuttle

import (
	"time"

	"github.com/rcrowley/go-metrics"
)

// Batcher coalesces logs coming via inLogs into batches, which are sent out
// via outBatches
type Batcher struct {
	inLogs     <-chan LogLine // Where I get the log lines to batch from
	outBatches chan<- Batch   // Where I send completed batches to
	drops      *Counter       // The drops counter
	timeout    time.Duration  // How long once we have a log line before we need to flush the batch
	batchSize  int            // The size of the batches

	// Various stats that we'll collect, see NewBatcher
	msgBatchedCount metrics.Counter
	msgDroppedCount metrics.Counter
	fillTime        metrics.Timer
}

// NewBatcher created an empty Batcher for the provided shuttle
func NewBatcher(s *Shuttle) Batcher {
	return Batcher{
		inLogs:          s.LogLines,
		drops:           s.Drops,
		outBatches:      s.Batches,
		timeout:         s.config.WaitDuration,
		batchSize:       s.config.BatchSize,
		msgBatchedCount: metrics.GetOrRegisterCounter("batch.msg.count", s.MetricsRegistry),
		msgDroppedCount: metrics.GetOrRegisterCounter("batch.msg.dropped", s.MetricsRegistry),
		fillTime:        metrics.GetOrRegisterTimer("batch.fill.time", s.MetricsRegistry),
	}
}

// Batch loops getting an empty batch and filling it. Filled batcches are sent
// via the outBatches channel. If outBatches is full, then the batch is dropped
// and the drops counters is incremented by the number of messages dropped.
func (b Batcher) Batch() {

	for {
		closeDown, batch := b.fillBatch()

		if msgCount := batch.MsgCount(); msgCount > 0 {
			select {
			case b.outBatches <- batch:
				// submitted into the delivery channel, just record some stats
				b.msgBatchedCount.Inc(int64(msgCount))
			default:
				//Unable to deliver into the delivery channel, increment drops
				b.msgDroppedCount.Inc(int64(msgCount))
				b.drops.Add(msgCount)
			}
		}

		if closeDown {
			break
		}
	}
}

// fillBatch coalesces individual log lines into batches. Delivery of the
// batch happens on timeout after at least one message is received
// or when the batch is full.
// returns the channel status, completed batch
func (b Batcher) fillBatch() (bool, Batch) {
	batch := NewBatch(b.batchSize) // Make a batch
	timeout := new(time.Timer)     // Gives us a nil channel and no timeout to start with
	chanOpen := true               // Assume the channel is open
	count := 0

	for {
		select {
		case <-timeout.C:
			return !chanOpen, batch

		case line, chanOpen := <-b.inLogs:
			if !chanOpen {
				return !chanOpen, batch
			}

			// We have a line now, so set a timeout
			if timeout.C == nil {
				defer func(t time.Time) { b.fillTime.UpdateSince(t) }(time.Now())
				timeout = time.NewTimer(b.timeout)
				defer timeout.Stop() // ensure timer is stopped when done
			}

			batch.Add(line)
			count++

			if count >= b.batchSize {
				return !chanOpen, batch
			}
		}
	}
}
