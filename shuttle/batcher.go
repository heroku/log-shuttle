package shuttle

import (
	"time"
)

// Batcher coalesces logs coming via inLogs into batches, which are sent out
// via outBatches
type Batcher struct {
	inLogs     <-chan LogLine    // Where I get the log lines to batch from
	outBatches chan<- Batch      // Where I send completed batches to
	stats      chan<- NamedValue // Where to send measurements
	drops      *Counter          // The drops counter
	timeout    time.Duration     // How long once we have a log line before we need to flush the batch
	batchSize  int               // The size of the batches
}

// NewBatcher created an empty Batcher from the provided channels / variables
func NewBatcher(batchSize int, timeout time.Duration, drops *Counter, stats chan<- NamedValue, inLogs <-chan LogLine, outBatches chan<- Batch) Batcher {
	return Batcher{
		inLogs:     inLogs,
		drops:      drops,
		stats:      stats,
		outBatches: outBatches,
		timeout:    timeout,
		batchSize:  batchSize,
	}
}

// Batch loops getting an empty batch and filling it. Filled batcches are sent
// via the outBatches channel. If outBatches is full, then the batch is dropped
// and the drops counters is incremented by the number of messages dropped.
func (batcher Batcher) Batch() {

	for {
		closeDown, batch := batcher.fillBatch()

		if msgCount := batch.MsgCount(); msgCount > 0 {
			select {
			case batcher.outBatches <- batch:
				// submitted into the delivery channel, just record some stats
				batcher.stats <- NewNamedValue("batch.msg.count", float64(msgCount))
			default:
				//Unable to deliver into the delivery channel, increment drops
				batcher.stats <- NewNamedValue("batch.msg.dropped", float64(msgCount))
				batcher.drops.Add(msgCount)
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
func (batcher Batcher) fillBatch() (bool, Batch) {
	batch := NewBatch(batcher.batchSize) // Make a batch
	timeout := new(time.Timer)           // Gives us a nil channel and no timeout to start with
	chanOpen := true                     // Assume the channel is open
	count := 0

	for {
		select {
		case <-timeout.C:
			return !chanOpen, batch

		case line, chanOpen := <-batcher.inLogs:
			if !chanOpen {
				return !chanOpen, batch
			}

			// We have a line now, so set a timeout
			if timeout.C == nil {
				defer func(t time.Time) { batcher.stats <- NewNamedValue("batch.fill.time", time.Since(t).Seconds()) }(time.Now())
				timeout = time.NewTimer(batcher.timeout)
				defer timeout.Stop() // ensure timer is stopped when done
			}

			batch.Add(line)
			count++

			if count >= batcher.batchSize {
				return !chanOpen, batch
			}
		}
	}
}
