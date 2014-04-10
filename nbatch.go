package main

// Holder of incoming lines
type NBatch struct {
	logLines []LogLine
}

// Returns a new batch with a capacity pre-set
func NewNBatch(capacity int) *NBatch {
	return &NBatch{logLines: make([]LogLine, 0, capacity)}
}

// Add a logline to the batch
func (nb *NBatch) Add(ll LogLine) {
	nb.logLines = append(nb.logLines, ll)
}

// The count of msgs in the batch
func (nb *NBatch) MsgCount() int {
	return len(nb.logLines)
}
