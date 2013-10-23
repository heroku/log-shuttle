package main

import (
	"container/list"
	"time"
)

type queued struct {
	when  time.Time
	batch *Batch
}

func BatchManager(getBatches, returnBatches chan *Batch, config *ShuttleConfig) {
	q := new(list.List)
	ticker := time.Tick(time.Minute)

	for {
		if q.Len() == 0 {
			q.PushFront(queued{when: time.Now(), batch: NewBatch(config)})
		}

		e := q.Front()

		select {
		case batch := <-returnBatches:
			//I've been given a batch back, queue it
			batch.Reset()
			q.PushFront(queued{when: time.Now(), batch: batch})

		case getBatches <- e.Value.(queued).batch:
			//I've sent the current batch out, remove it from the queue
			q.Remove(e)

		case <-ticker:
			//Periodically go through the queued batches and throw
			//out ones that have been queued for too long in an effort
			//to expire old batches that were created because of bursts
			for e := q.Front(); e != nil; e = e.Next() {
				if time.Since(e.Value.(queued).when) > time.Minute {
					q.Remove(e)
					e.Value = nil
				}
			}
		}
	}
}

func NewBatchManager(config ShuttleConfig) (getBatches, returnBatches chan *Batch) {
	getBatches = make(chan *Batch)
	returnBatches = make(chan *Batch)

	go BatchManager(getBatches, returnBatches, &config)

	return
}
