package main

import (
	"sync"
	"time"
)

func StartBatchers(config ShuttleConfig, stats *ProgramStats, inLogs <-chan *LogLine, inBatches <-chan *Batch, outBatches chan<- *Batch) *sync.WaitGroup {
	batchWaiter := new(sync.WaitGroup)
	for i := 0; i < config.NumBatchers; i++ {
		batchWaiter.Add(1)
		go func() {
			defer batchWaiter.Done()
			batcher := NewBatcher(config, inLogs, inBatches, outBatches)
			batcher.Batch(stats)
		}()
	}

	return batchWaiter
}

type Batcher struct {
	inLogs     <-chan *LogLine // Where I get the log lines to batch from
	inBatches  <-chan *Batch   // Where I get empty batches from
	outBatches chan<- *Batch   // Where I send completed batches to
	config     ShuttleConfig
}

func NewBatcher(config ShuttleConfig, inLogs <-chan *LogLine, inBatches <-chan *Batch, outBatches chan<- *Batch) *Batcher {
	return &Batcher{inLogs: inLogs, inBatches: inBatches, outBatches: outBatches, config: config}
}

// Loops getting an empty batch and filling it.
func (batcher *Batcher) Batch(stats *ProgramStats) {

	for batch := range batcher.inBatches {
		count, closeDown := batcher.fillBatch(batch)
		if count > 0 {
			select {
			case batcher.outBatches <- batch:
			// submitted into the delivery channel,
			// nothing to do here.
			default:
				//Unable to deliver into the delivery channel,
				//increment drops
				stats.Drops.Add(uint64(count))
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
func (batcher *Batcher) fillBatch(batch *Batch) (int, bool) {
	// Fill the batch with log lines
	var line *LogLine
	open := true

	timeout := time.NewTimer(batcher.config.WaitDuration)
	timeout.Stop()       // don't timeout until we actually have a log line
	defer timeout.Stop() // ensure timer is stopped

	for {
		select {
		case <-timeout.C:
			return batch.MsgCount, !open

		case line, open = <-batcher.inLogs:
			if !open {
				return batch.MsgCount, !open
			}
			batch.Write(line)
			if batch.MsgCount == batcher.config.BatchSize {
				return batch.MsgCount, !open
			}
			if batch.MsgCount == 1 {
				timeout.Reset(batcher.config.WaitDuration)
			}
		}
	}
}
