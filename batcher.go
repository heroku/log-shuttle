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
	drop       bool           // Should we drop or not (backup instead)

	// Various stats that we'll collect, see NewBatcher
	msgBatchedCount metrics.Counter
	msgDroppedCount metrics.Counter
	fillTime        metrics.Timer
}

// NewBatcher created an empty Batcher for the provided shuttle
func NewBatcher(s *Shuttle) Batcher {
	return Batcher{
		drops:           s.Drops,
		outBatches:      s.Batches,
		timeout:         s.config.WaitDuration,
		batchSize:       s.config.BatchSize,
		drop:            s.config.Drop,
		msgBatchedCount: metrics.GetOrRegisterCounter("msg.batched", s.MetricsRegistry),
		msgDroppedCount: metrics.GetOrRegisterCounter("msg.dropped", s.MetricsRegistry),
		fillTime:        metrics.GetOrRegisterTimer("batch.fill", s.MetricsRegistry),
	}
}

// Batch loops getting an empty batch and filling it. Filled batcches are sent
// via the outBatches channel. If outBatches is full, then the batch is dropped
// and the drops counters is incremented by the number of messages dropped.
func (b Batcher) Batch() {

	for {
		closeDown, batch := b.fillBatch()

		if msgCount := batch.MsgCount(); msgCount > 0 {
			if b.drop {
				select {
				case b.outBatches <- batch:
					// submitted into the delivery channel, just record some stats
					b.msgBatchedCount.Inc(int64(msgCount))
				default:
					//Unable to deliver into the delivery channel, increment drops
					b.msgDroppedCount.Inc(int64(msgCount))
					b.drops.Add(msgCount)
				}
			} else {
				b.outBatches <- batch
				b.msgBatchedCount.Inc(int64(msgCount))
			}
		}

		if closeDown {
			return
		}
	}
}

// fillBatch coalesces individual log lines into batches. Delivery of the
// batch happens on timeout after at least one message is received
// or when the batch is full.
// returns the channel status, completed batch
func (b Batcher) fillBatch() (bool, Batch) {
	batch := NewBatch(b.batchSize)
	timeout := new(time.Timer) // start with a nil channel and no timeout

	for {
		select {
		case <-timeout.C:
			return false, batch

		case line, chanOpen := <-b.inLogs:
			// if the channel is closed, then line will be a zero value Line, so just
			// return the batch and signal shutdown
			if !chanOpen {
				return true, batch
			}

			// Set a timeout if we don't have one
			if timeout.C == nil {
				defer func(t time.Time) { b.fillTime.UpdateSince(t) }(time.Now())
				timeout = time.NewTimer(b.timeout)
				defer timeout.Stop() // ensure timer is stopped when done
			}

			if full := batch.Add(line); full {
				return false, batch
			}
		}
	}
}
