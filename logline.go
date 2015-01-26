package shuttle

import "time"

// LogLine holds the new line terminated log messages and when shuttle received them.
type LogLine struct {
	line []byte
	when time.Time
}

// NewLogLine returns a newly constructed LogLine containing the provided line and time
func NewLogLine(line []byte, when time.Time) LogLine {
	return LogLine{line, when}
}

// Length returns the length of the raw byte of the LogLine
func (ll LogLine) Length() int {
	return len(ll.line)
}
