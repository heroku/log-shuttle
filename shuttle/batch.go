package shuttle

import "github.com/nu7hatch/gouuid"

// Holder of incoming lines
type Batch struct {
	logLines []LogLine
	UUID     *uuid.UUID
}

// Returns a new batch with a capacity pre-set
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

// The count of msgs in the batch
func (b *Batch) MsgCount() int {
	return len(b.logLines)
}

func (b *Batch) Encode() (message []byte, err error) {

	return
}
