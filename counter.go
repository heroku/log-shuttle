package main

import (
	"log"
	"log/syslog"
	"sync/atomic"
	"time"
)

type Counter struct {
	value         uint64
	LastIncrement time.Time
}

func (c *Counter) Read() uint64 {
	return c.value
}

func (c *Counter) ReadAndReset() uint64 {
	for {
		oldCount := c.value
		if atomic.CompareAndSwapUint64(&c.value, oldCount, 0) {
			return oldCount
		}
	}
}

func (c *Counter) Add(u uint64) uint64 {
	c.LastIncrement = time.Now()
	return atomic.AddUint64(&c.value, u)
}

func NewProgramStats(bi chan *LogLine, oi chan *Batch) *ProgramStats {
	return &ProgramStats{batchInput: bi, outletInput: oi}
}

func PeriodicReporter(config ShuttleConfig, stats *ProgramStats) {
	logger, err := syslog.NewLogger(syslog.LOG_USER|syslog.LOG_DEBUG, log.LstdFlags)
	if err != nil {
		log.Fatal("Unable to setup periodic reporting logger")
	}
	go func() {
		ticker := time.Tick(config.ReportEvery)
		for {
			select {
			case <-ticker:
				logger.Println("hello")
			}
		}
	}()
}

type ProgramStats struct {
	Reads             Counter
	Drops             Counter
	OutletPostSuccess Counter
	OutletPostError   Counter
	batchInput        chan *LogLine
	outletInput       chan *Batch
}
