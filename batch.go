package shuttle

import "github.com/nu7hatch/gouuid"

// Batch holds incoming log lines and provides some helpers for dealing with
// the grouping of logLines
type Batch struct {
	logLines []LogLine
	UUID     *uuid.UUID
}

// NewBatch returns a new batch with a capacity pre-set
func NewBatch(capacity int) Batch {
	uuid, err := uuid.NewV4()
	if err != nil {
		ErrLogger.Printf("at=new_batch.generate_uuid err=%q\n", err)
	}
	return Batch{
		logLines: make([]LogLine, 0, capacity),
		UUID:     uuid,
	}
}

// Add a logline to the batch
func (b *Batch) Add(ll LogLine) {
	b.logLines = append(b.logLines, ll)
}

// MsgCount returns the count of msgs in the batch
func (b *Batch) MsgCount() int {
	return len(b.logLines)
}
