package main

import (
	"github.com/bmizerany/perks/quantile"
	"log"
	"log/syslog"
	"time"
)

type NamedValue struct {
	value float64
	name  string
}

type ProgramStats struct {
	Lost     Counter
	Drops    Counter
	stats    map[string]*quantile.Stream
	receiver chan NamedValue
}

func NewProgramStats(stats chan NamedValue) *ProgramStats {
	return &ProgramStats{
		receiver: stats,
		stats:    make(map[string]*quantile.Stream),
	}
}

func (stats *ProgramStats) StartPeriodicReporter(config ShuttleConfig) {
	logger, err := syslog.NewLogger(syslog.LOG_SYSLOG|syslog.LOG_NOTICE, log.LstdFlags)
	if err != nil {
		log.Fatal("Unable to setup periodic reporting logger")
	}

	go func() {
		ticker := time.Tick(config.ReportEvery)
		var lastLost, lastDrops uint64
		var sample *quantile.Stream
		var ok bool

		for {
			select {
			case namedValue, open := <-stats.receiver:
				if !open {
					return
				}

				sample, ok = stats.stats[namedValue.name]
				if !ok {
					sample = quantile.NewTargeted(0.50, 0.95, 0.99)
				}
				sample.Insert(namedValue.value)
				stats.stats[namedValue.name] = sample

			case <-ticker:
				logger.Printf("source=%s count#log-shuttle.lost=%d count#log-shuttle.drops=%d\n",
					config.Appname,
					diffUp(stats.Lost.AllTime(), &lastLost),
					diffUp(stats.Drops.AllTime(), &lastDrops),
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
