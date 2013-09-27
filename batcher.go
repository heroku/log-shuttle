package main

import (
	"time"
)

type Batcher struct {
	Batches chan *Batch
	Outbox  chan *Batch
	inbox   <-chan string
	config  ShuttleConfig
}

func NewBatcher(config ShuttleConfig, inbox <-chan string) *Batcher {
	batcher := new(Batcher)
	batcher.Batches = make(chan *Batch, config.Batches)
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

// Batch coalesces individual log lines into batches.
// If there is high volume traffic on the inbox channel, we send batches based
// on BatchSize. For low volume traffic, we create batches based on a time
// interval.
func (batcher *Batcher) Batch() error {
	ticker := time.Tick(batcher.config.WaitDuration())

	for {
		// Get an emptu batch
		batch := batcher.GetEmptyBatch()

	NEWBATCH:

		// Fill the batch with log lines
		for {
			select {
			case <-ticker:
				if batch.LineCount() > 0 {
					batcher.Outbox <- batch
					break NEWBATCH
				}

			case line := <-batcher.inbox:
				batch.Write(line)
				if batch.LineCount() == batcher.config.BatchSize {
					batcher.Outbox <- batch
					break NEWBATCH
				}
			}
		}
	}

}
