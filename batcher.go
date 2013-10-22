package main

import (
	"time"
)

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
func (batcher *Batcher) Batch() error {
	ticker := time.Tick(batcher.config.WaitDuration())

	for {
		batch := <-batcher.inBatches
		batcher.fillBatch(ticker, batch)
	}
}

// fillBatch coalesces individual log lines into batches. Delivery of the
// batch happens on ticker timeout or when the batch is full.
func (batcher *Batcher) fillBatch(ticker <-chan time.Time, batch *Batch) {
	// Fill the batch with log lines
	for {
		select {
		case <-ticker:
			if batch.LineCount() > 0 {
				batcher.outBatches <- batch
				return
			}

		case line := <-batcher.inLogs:
			batch.Write(line)
			if batch.LineCount() == batcher.config.BatchSize {
				batcher.outBatches <- batch
				return
			}
		}
	}
}
