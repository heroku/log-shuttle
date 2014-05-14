package main

import (
	"fmt"
	"time"
)

type LogLine struct {
	line []byte
	when time.Time
}

func (ll LogLine) Length() int {
	return len(ll.line)
}

func (ll *LogLine) Header(config *ShuttleConfig) string {
	if config.SkipHeaders {
		return fmt.Sprintf("%d ", len(ll.line))
	} else {
		s := fmt.Sprintf(config.syslogFrameHeaderFormat,
			config.lengthPrefixedSyslogFrameHeaderSize+len(ll.line),
			ll.when.UTC().Format(LOGPLEX_BATCH_TIME_FORMAT))
		return s
	}
}