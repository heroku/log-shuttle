package main

import (
	"sync"
	"time"
)

func StartBatchers(config ShuttleConfig, drops *Counter, stats chan<- NamedValue, inLogs <-chan LogLine, inBatches <-chan *Batch, outBatches chan<- *Batch) *sync.WaitGroup {
	batchWaiter := new(sync.WaitGroup)
	for i := 0; i < config.NumBatchers; i++ {
		batchWaiter.Add(1)
		go func() {
			defer batchWaiter.Done()
			batcher := NewBatcher(config, drops, stats, inLogs, inBatches, outBatches)
			batcher.Batch()
		}()
	}

	return batchWaiter
}

type Batcher struct {
	inLogs     <-chan LogLine    // Where I get the log lines to batch from
	inBatches  <-chan *Batch     // Where I get empty batches from
	outBatches chan<- *Batch     // Where I send completed batches to
	stats      chan<- NamedValue // Where to send measurements
	drops      *Counter          // The drops counter
	timeout    time.Duration     // How long once we have a log line before we need to flush the batch
}

func NewBatcher(config ShuttleConfig, drops *Counter, stats chan<- NamedValue, inLogs <-chan LogLine, inBatches <-chan *Batch, outBatches chan<- *Batch) *Batcher {
	return &Batcher{
		inLogs:     inLogs,
		inBatches:  inBatches,
		drops:      drops,
		stats:      stats,
		outBatches: outBatches,
		timeout:    config.WaitDuration,
	}
}

// Loops getting an empty batch and filling it.
func (batcher *Batcher) Batch() {

	for batch := range batcher.inBatches {
		batcher.stats <- NewNamedValue("batcher.inBatches.length", float64(len(batcher.inBatches)))
		batcher.stats <- NewNamedValue("batcher.inLogs.length", float64(len(batcher.inLogs)))

		closeDown := batcher.fillBatch(batch)
		if batch.MsgCount > 0 {
			batcher.stats <- NewNamedValue("batch.msg.count", float64(batch.MsgCount))
			select {
			case batcher.outBatches <- batch:
			// submitted into the delivery channel,
			// nothing to do here.
			default:
				//Unable to deliver into the delivery channel,
				//increment drops
				batcher.stats <- NewNamedValue("batch.msg.dropped", float64(batch.MsgCount))
				batcher.drops.Add(batch.MsgCount)
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

			batch.Write(line)

			if batch.Full() {
				return !chanOpen
			}
		}
	}
}
