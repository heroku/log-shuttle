package main

import (
	"sync"
	"time"
)

func StartBatchers(count int, config ShuttleConfig, inLogs <-chan *LogLine, inBatches <-chan *Batch, outBatches chan<- *Batch) *sync.WaitGroup {
	batchWaiter := new(sync.WaitGroup)
	for ; count > 0; count-- {
		batchWaiter.Add(1)
		go func() {
			defer batchWaiter.Done()
			batcher := NewBatcher(config, inLogs, inBatches, outBatches)
			batcher.Batch()
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

// Loops forever getting an empty batch and filling it.
func (batcher *Batcher) Batch() {
	ticker := time.Tick(batcher.config.WaitDuration())

	for batch := range batcher.inBatches {
		closeDown := batcher.fillBatch(ticker, batch)
		if batch.LineCount > 0 {
			batcher.outBatches <- batch
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
