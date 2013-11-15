package main

import (
	"log"
	"log/syslog"
	"sync"
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
	return &ProgramStats{dropsMutex: new(sync.Mutex), lostMutex: new(sync.Mutex), batchInput: bi, outletInput: oi}
}

func StartPeriodicReporter(config ShuttleConfig, stats *ProgramStats) {
	logger, err := syslog.NewLogger(syslog.LOG_SYSLOG|syslog.LOG_NOTICE, log.LstdFlags)
	if err != nil {
		log.Fatal("Unable to setup periodic reporting logger")
	}

	go func() {
		ticker := time.Tick(config.ReportEvery)
		var lastReads, lastLost, lastDrops, lastSuccess, lastError uint64
		for {
			select {
			case <-ticker:
				logger.Printf("source=%s count#log-shuttle.reads=%d count#log-shuttle.lost=%d count#log-shuttle.drops=%d count#log-shuttle.outlet.post.success=%d count#log-shuttle.outlet.post.error=%d sample#log-shuttle.batch.input.length=%d sample#log-shuttle.outlet.input.length=%d\n",
					config.Appname,
					diffUp(stats.Reads.Read(), &lastReads),
					diffUp(stats.AllTimeLost.Read(), &lastLost),
					diffUp(stats.AllTimeDrops.Read(), &lastDrops),
					diffUp(stats.OutletPostSuccess.Read(), &lastSuccess),
					diffUp(stats.OutletPostError.Read(), &lastError),
					len(stats.batchInput),
					len(stats.outletInput),
				)

			}
		}
	}()
}

func diffUp(cv uint64, lv *uint64) uint64 {
	defer func() { *lv = cv }()
	return cv - *lv
}

type ProgramStats struct {
	Reads             Counter
	CurrentLost       Counter
	AllTimeLost       Counter
	CurrentDrops      Counter
	AllTimeDrops      Counter
	OutletPostSuccess Counter
	OutletPostError   Counter
	batchInput        chan *LogLine
	outletInput       chan *Batch
	dropsMutex        *sync.Mutex
	lostMutex         *sync.Mutex
}

func (ps *ProgramStats) IncrementDrops(i uint64) uint64 {
	ps.dropsMutex.Lock()
	defer ps.dropsMutex.Unlock()
	ps.CurrentDrops.Add(i)
	return ps.AllTimeDrops.Add(i)
}

func (ps *ProgramStats) ReadAndResetDrops() uint64 {
	ps.dropsMutex.Lock()
	defer ps.dropsMutex.Unlock()
	return ps.CurrentDrops.ReadAndReset()
}

func (ps *ProgramStats) IncrementLost(i uint64) uint64 {
	ps.lostMutex.Lock()
	defer ps.lostMutex.Unlock()
	ps.CurrentLost.Add(i)
	return ps.AllTimeLost.Add(i)
}

func (ps *ProgramStats) ReadAndResetLost() uint64 {
	ps.lostMutex.Lock()
	defer ps.lostMutex.Unlock()
	return ps.CurrentLost.ReadAndReset()
}
