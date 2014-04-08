package main

import "time"

type LogLine struct {
	line []byte
	when time.Time
}

func (ll LogLine) Length() int {
	return len(ll.line)
}
