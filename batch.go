package main

import (
	"time"

	"github.com/nu7hatch/gouuid"
)

// Holder of incoming lines
type Batch struct {
	logLines       []LogLine
	oldest, newest *time.Time
	UUID           *uuid.UUID
}

// Returns a new batch with a capacity pre-set
func NewBatch(capacity int) *Batch {
	uuid, err := uuid.NewV4()
	if err != nil {
		ErrLogger.Printf("at=new_batch.generate_uuid err=%q\n", err)
	}
	return &Batch{
		logLines: make([]LogLine, 0, capacity),
		UUID:     uuid,
	}
}

// Add a logline to the batch
func (b *Batch) Add(ll LogLine) {
	b.updateTimes(ll.when)
	b.logLines = append(b.logLines, ll)
}

// The count of msgs in the batch
func (b *Batch) MsgCount() int {
	return len(b.logLines)
}

func (b *Batch) MsgAgeRange() float64 {
	if b.oldest == nil || b.newest == nil {
		return 0.0
	}
	newest := *b.newest
	return newest.Sub(*b.oldest).Seconds()
}

func (b *Batch) updateTimes(t time.Time) {
	if b.oldest == nil || t.Before(*b.oldest) {
		b.oldest = &t
	}
	if b.newest == nil || t.After(*b.newest) {
		b.newest = &t
	}
	return
}
