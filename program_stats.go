package main

import (
	"github.com/bmizerany/perks/quantile"
	"log"
	"log/syslog"
	"sync"
	"time"
)

const (
	STATS_CHANNEL_BUFFER_SIZE = 1000
)

func NewProgramStats(bi chan LogLine, oi chan *Batch) *ProgramStats {
	return &ProgramStats{
		batchInput:   bi,
		outletInput:  oi,
		StatsChannel: make(chan NamedValue, STATS_CHANNEL_BUFFER_SIZE),
		stats:        make(map[string]*quantile.Stream),
		Mutex:        new(sync.Mutex),
	}
}

func (stats *ProgramStats) StartPeriodicReporter(config ShuttleConfig) {
	logger, err := syslog.NewLogger(syslog.LOG_SYSLOG|syslog.LOG_NOTICE, log.LstdFlags)
	if err != nil {
		log.Fatal("Unable to setup periodic reporting logger")
	}

	go func() {
		ticker := time.Tick(config.ReportEvery)
		var lastLost, lastDrops, lastSuccess, lastError uint64
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
				logger.Printf("source=%s count#log-shuttle.lost=%d count#log-shuttle.drops=%d count#log-shuttle.outlet.post.success=%d count#log-shuttle.outlet.post.error=%d sample#log-shuttle.batch.input.length=%d sample#log-shuttle.outlet.input.length=%d\n",
					config.Appname,
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
		logger.Printf("source=%s count#log-shuttle.%s.count=%d sample#log-shuttle.%s.p50=%f sample#log-shuttle.%s.p95=%f sample#log-shuttle.%s.p99=%f\n",
			source,
			thing, stats.Count(),
			thing, stats.Query(0.50),
			thing, stats.Query(0.95),
			thing, stats.Query(0.99),
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
	*sync.Mutex
}

func (ps *ProgramStats) IncrementDrops(i uint64) uint64 {
	ps.Lock()
	defer ps.Unlock()
	ps.CurrentDrops.Add(i)
	return ps.AllTimeDrops.Add(i)
}

func (ps *ProgramStats) ReadAndResetDrops() uint64 {
	ps.Lock()
	defer ps.Unlock()
	return ps.CurrentDrops.ReadAndReset()
}

func (ps *ProgramStats) IncrementLost(i uint64) uint64 {
	ps.Lock()
	defer ps.Unlock()
	ps.CurrentLost.Add(i)
	return ps.AllTimeLost.Add(i)
}

func (ps *ProgramStats) ReadAndResetLost() uint64 {
	ps.Lock()
	defer ps.Unlock()
	return ps.CurrentLost.ReadAndReset()
}
