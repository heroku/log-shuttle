package main

import (
	"time"
)

type Batcher struct {
	Batches chan *Batch
	Outbox  chan *Batch
	inbox   <-chan *string
	config  ShuttleConfig
}

func NewBatcher(config ShuttleConfig, inbox <-chan *string) *Batcher {
	batcher := new(Batcher)
	batcher.Batches = make(chan *Batch, config.Batches*2)
	batcher.Outbox = make(chan *Batch, config.Batches)
	batcher.inbox = inbox
	batcher.config = config
	return batcher
}

// Do what you can to return an empty batch
// Try pulling one from Batches
// If not, make one
func (batcher *Batcher) GetEmptyBatch() (batch *Batch) {
	select {
	case batch = <-batcher.Batches:
		// Got one; Reset it
		batch.Reset()
	default:
		batch = new(Batch)
	}

	return batch
}

// Loops forever getting an empty batch and filling it.
func (batcher *Batcher) Batch() error {
	ticker := time.Tick(batcher.config.WaitDuration())

	for {
		// Get an empty batch
		batch := batcher.GetEmptyBatch()
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
				batcher.Outbox <- batch
				return
			}

		case line := <-batcher.inbox:
			batch.Write(line)
			if batch.LineCount() == batcher.config.BatchSize {
				batcher.Outbox <- batch
				return
			}
		}
	}
}
