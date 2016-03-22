package shuttle

import "github.com/pborman/uuid"

// Batch holds incoming log lines and provides some helpers for dealing with
// the grouping of logLines
type Batch struct {
	logLines []LogLine
	UUID     string
}

// NewBatch returns a new batch with a capacity pre-set
func NewBatch(capacity int) Batch {
	return Batch{
		logLines: make([]LogLine, 0, capacity),
		UUID:     uuid.New(),
	}
}

// Add a logline to the batch and return a boolean indicating if the batch is
// full or not
func (b *Batch) Add(ll LogLine) bool {
	b.logLines = append(b.logLines, ll)
	return len(b.logLines) == cap(b.logLines)
}

// MsgCount returns the number of msgs in the batch
func (b *Batch) MsgCount() int {
	return len(b.logLines)
}
