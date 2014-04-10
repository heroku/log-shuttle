package main

import (
	"sync"
	"time"
)

func StartBatchers(config ShuttleConfig, drops *Counter, stats chan<- NamedValue, inLogs <-chan LogLine, outBatches chan<- *Batch) *sync.WaitGroup {
	batchWaiter := new(sync.WaitGroup)
	for i := 0; i < config.NumBatchers; i++ {
		batchWaiter.Add(1)
		go func() {
			defer batchWaiter.Done()
			batcher := NewBatcher(config.BatchSize, config.Timeout, drops, stats, inLogs, outBatches)
			batcher.Batch()
		}()
	}

	return batchWaiter
}

type Batcher struct {
	inLogs     <-chan LogLine    // Where I get the log lines to batch from
	outBatches chan<- *Batch     // Where I send completed batches to
	stats      chan<- NamedValue // Where to send measurements
	drops      *Counter          // The drops counter
	timeout    time.Duration     // How long once we have a log line before we need to flush the batch
	batchSize  int               // The size of the batches
}

func NewBatcher(batchSize int, timeout time.Duration, drops *Counter, stats chan<- NamedValue, inLogs <-chan LogLine, outBatches chan<- *Batch) *Batcher {
	return &Batcher{
		inLogs:     inLogs,
		drops:      drops,
		stats:      stats,
		outBatches: outBatches,
		timeout:    timeout,
		batchSize:  batchSize,
	}
}

// Loops getting an empty batch and filling it.
func (batcher *Batcher) Batch() {

	for {
		//Make a new batch
		batch := NewBatch(batcher.batchSize)

		closeDown := batcher.fillBatch(batch)

		if msgCount := batch.MsgCount(); msgCount > 0 {
			select {
			case batcher.outBatches <- batch:
				batcher.stats <- NewNamedValue("batch.msg.count", float64(msgCount))
			// submitted into the delivery channel,
			// nothing to do here.
			default:
				//Unable to deliver into the delivery channel,
				//increment drops
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
func (batcher *Batcher) fillBatch(batch *Batch) (chanOpen bool) {
	timeout := new(time.Timer) // Gives us a nil channel and no timeout to start with
	chanOpen = true            // Assume the channel is open
	count := 0

	for {
		select {
		case <-timeout.C:
			return !chanOpen

		case line, chanOpen := <-batcher.inLogs:
			if !chanOpen {
				return !chanOpen
			}

			// We have a line now, so set a timeout
			if timeout.C == nil {
				defer func(t time.Time) { batcher.stats <- NewNamedValue("batch.fill.time", time.Since(t).Seconds()) }(time.Now())
				timeout = time.NewTimer(batcher.timeout)
				defer timeout.Stop() // ensure timer is stopped when done
			}

			batch.Add(line)
			count += 1

			if count >= batcher.batchSize {
				return !chanOpen
			}
		}
	}
}
