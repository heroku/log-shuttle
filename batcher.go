package main

import (
	"time"
)

type Batcher struct {
	Outbox chan []string
	inbox  <-chan string
	config ShuttleConfig
}

func NewBatcher(config ShuttleConfig, inbox <-chan string) *Batcher {
	b := new(Batcher)
	b.Outbox = make(chan []string, config.BatchSize)
	b.inbox = inbox
	b.config = config
	return b
}

func (b *Batcher) NewBuffer() []string {
	return make([]string, 0, b.config.BatchSize)
}

// Batch coalesces individual log lines into batches.
// If there is high volume traffic on the inbox channel, we create batches
// based on the batchSize flag. For low volume traffic, we create batches based
// on a time interval.
func (b *Batcher) Batch() error {
	ticker := time.Tick(b.config.WaitDuration())
	batch := b.NewBuffer()
	for {
		select {
		case <-ticker:
			if len(batch) > 0 {
				b.Outbox <- batch
				batch = b.NewBuffer()
			}
		case l := <-b.inbox:
			batch = append(batch, l)
			if len(batch) == cap(batch) {
				b.Outbox <- batch
				batch = b.NewBuffer()
			}
		}
	}
}
