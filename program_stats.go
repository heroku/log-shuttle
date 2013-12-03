package main

import (
	"github.com/bmizerany/perks/quantile"
	"log"
	"log/syslog"
	"sync"
	"time"
)

func NewProgramStats(bi chan LogLine, oi chan *Batch) *ProgramStats {
	return &ProgramStats{
		dropsMutex:   new(sync.Mutex),
		lostMutex:    new(sync.Mutex),
		batchInput:   bi,
		outletInput:  oi,
		StatsChannel: make(chan NamedValue),
		stats:        make(map[string]*quantile.Stream),
	}
}

func (stats *ProgramStats) StartPeriodicReporter(config ShuttleConfig) {
	logger, err := syslog.NewLogger(syslog.LOG_SYSLOG|syslog.LOG_NOTICE, log.LstdFlags)
	if err != nil {
		log.Fatal("Unable to setup periodic reporting logger")
	}

	go func() {
		ticker := time.Tick(config.ReportEvery)
		var lastReads, lastLost, lastDrops, lastSuccess, lastError uint64
		var sample *quantile.Stream
		var ok bool

		for {
			select {
			case namedValue := <-stats.StatsChannel:
				if sample, ok = stats.stats[namedValue.name]; ok != true {
					sample = quantile.NewTargeted(0.50, 0.95, 0.99)
				}
				sample.Insert(namedValue.value)
				stats.stats[namedValue.name] = sample

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

				for name, sample := range stats.stats {
					logStats(config.Appname, name, logger, sample)
				}
			}
		}
	}()
}

func logStats(source string, thing string, logger *log.Logger, stats *quantile.Stream) {
	if stats.Count() > 0 {
		logger.Printf("source=%s sample#log-shuttle.%s.time.p50=%fs sample#log-shuttle.%[2]s.time.p95=%[4]fs sample#log-shuttle.%[2]s.time.p99=%[5]fs\n",
			source,
			thing,
			stats.Query(0.50),
			stats.Query(0.95),
			stats.Query(0.99),
		)
		stats.Reset()
	}
}

func diffUp(cv uint64, lv *uint64) uint64 {
	defer func() { *lv = cv }()
	return cv - *lv
}

type NamedValue struct {
	value float64
	name  string
}

type ProgramStats struct {
	Reads             Counter
	CurrentLost       Counter
	AllTimeLost       Counter
	CurrentDrops      Counter
	AllTimeDrops      Counter
	OutletPostSuccess Counter
	OutletPostError   Counter
	batchInput        chan LogLine
	outletInput       chan *Batch
	stats             map[string]*quantile.Stream
	StatsChannel      chan NamedValue
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
