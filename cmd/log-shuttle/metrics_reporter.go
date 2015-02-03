package main

import (
	"log"
	"time"

	"github.com/heroku/log-shuttle/Godeps/_workspace/src/github.com/heroku/slog"
	metrics "github.com/heroku/log-shuttle/Godeps/_workspace/src/github.com/rcrowley/go-metrics"
)

// Percentile info for histograms and the like. These 2 arrays need to match
// length and positions are relevant to each other.
var (
	percentiles     = []float64{0.5, 0.75, 0.95, 0.99, 0.999}
	percentileNames = []string{"mean", "p75", "p95", "p99", "p999"}
)

// LogFmtMetricsEmitter emits the metrics in logfmt compatible formats every d
// duration using the provided logger. source is added to the line as
// log_shuttle_stats_source if not empty.
func LogFmtMetricsEmitter(r metrics.Registry, source string, d time.Duration, l *log.Logger) {
	if d == 0 {
		return
	}
	for _ = range time.Tick(d) {
		ctx := slog.Context{}
		if source != "" {
			ctx["log_shuttle_stats_source"] = source
		}
		r.Each(func(name string, i interface{}) {
			switch metric := i.(type) {
			case metrics.Counter:
				ctx[name] = metric.Count()
			case metrics.Gauge:
				ctx[name] = metric.Value()
			case metrics.GaugeFloat64:
				ctx[name] = metric.Value()
			case metrics.Healthcheck:
				metric.Check()
				ctx[name] = metric.Error()
			case metrics.Histogram:
				s := metric.Snapshot()
				ps := s.Percentiles(percentiles)
				ctx[name+".count"] = s.Count()
				ctx[name+".min"] = s.Min()
				ctx[name+".max"] = s.Max()
				ctx[name+".mean"] = s.Mean()
				ctx[name+".stddev"] = s.StdDev()
				for i, pn := range percentileNames {
					ctx[name+"."+pn] = ps[i]
				}
			case metrics.Meter:
				s := metric.Snapshot()
				ctx[name+".count"] = s.Count()
				ctx[name+".1min"] = s.Rate1()
				ctx[name+".5min"] = s.Rate5()
				ctx[name+".15min"] = s.Rate15()
				ctx[name+".mean"] = s.RateMean()
			case metrics.Timer:
				s := metric.Snapshot()
				ps := s.Percentiles(percentiles)
				ctx[name+".count"] = s.Count()
				ctx[name+".min"] = s.Min()
				ctx[name+".max"] = s.Max()
				ctx[name+".mean"] = s.Mean()
				ctx[name+".stddev"] = s.StdDev()
				for i, pn := range percentileNames {
					ctx[name+"."+pn] = ps[i]
				}
				ctx[name+".1min"] = s.Rate1()
				ctx[name+".5min"] = s.Rate5()
				ctx[name+".15min"] = s.Rate15()
				ctx[name+".mean"] = s.RateMean()
			}
		})
		l.Println(ctx)
	}
}
