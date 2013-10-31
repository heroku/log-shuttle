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
	ticker := time.Tick(batcher.config.WaitDuration())

	for batch := range batcher.inBatches {
		closeDown := batcher.fillBatch(ticker, batch)
		if batch.LineCount > 0 {
			select {
			case batcher.outBatches <- batch:
			// submitted into the delivery channel,
			// nothing to do here.
			default:
				//Unable to deliver into the delivery channel,
				//increment
				stats.Drops.Add(uint64(batch.LineCount))
			}
		}
		if closeDown {
			break
		}
	}
}

// fillBatch coalesces individual log lines into batches. Delivery of the
// batch happens on ticker timeout or when the batch is full.
func (batcher *Batcher) fillBatch(ticker <-chan time.Time, batch *Batch) (closed bool) {
	// Fill the batch with log lines
	var line *LogLine

	for open := true; open; {
		select {
		case <-ticker:
			if batch.LineCount > 0 { // Stay here, unless we have some lines
				return !open
			}

		case line, open = <-batcher.inLogs:
			if !open {
				return true
			}
			batch.Write(line)
			if batch.LineCount == batcher.config.BatchSize {
				return
			}
		}
	}

	return true

}
