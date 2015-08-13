package main

import (
	"fmt"
	"log"
	"time"

	"github.com/heroku/slog"
	metrics "github.com/rcrowley/go-metrics"
)

// Percentile info for histograms and the like. These 2 arrays need to match
// length and positions are relevant to each other.
var (
	percentiles     = []float64{0.75, 0.95, 0.99}
	percentileNames = []string{"p75", "p95", "p99"}
	lastCounts      = make(map[string]int64)
)

// Given a t representing a time in ns, convert to seconds, show up to μs precision
func sec(t float64) string {
	return fmt.Sprintf("%.6f", t/1000000000)
}

func countDifference(ctx slog.Context, name string, c int64) {
	name = name + ".count"
	lc := lastCounts[name]
	ctx[name] = c - lc
	lastCounts[name] = c
}

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
				countDifference(ctx, name, metric.Count())
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
				countDifference(ctx, name, s.Count())
				ctx[name+".min"] = s.Min()
				ctx[name+".max"] = s.Max()
				ctx[name+".mean"] = s.Mean()
				ctx[name+".stddev"] = s.StdDev()
				for i, pn := range percentileNames {
					ctx[name+"."+pn] = ps[i]
				}
			case metrics.Meter:
				s := metric.Snapshot()
				countDifference(ctx, name, s.Count())
				ctx[name+".rate.1min"] = s.Rate1()
				ctx[name+".rate.5min"] = s.Rate5()
				ctx[name+".rate.15min"] = s.Rate15()
				ctx[name+".rate.mean"] = s.RateMean()
			case metrics.Timer:
				s := metric.Snapshot()
				ps := s.Percentiles(percentiles)
				countDifference(ctx, name, s.Count())
				ctx[name+".min"] = sec(float64(s.Min()))
				ctx[name+".max"] = sec(float64(s.Max()))
				ctx[name+".mean"] = sec(s.Mean())
				ctx[name+".stddev"] = sec(s.StdDev())
				for i, pn := range percentileNames {
					ctx[name+"."+pn] = sec(ps[i])
				}
				ctx[name+".rate.1min"] = fmt.Sprintf("%.3f", s.Rate1())
				ctx[name+".rate.5min"] = fmt.Sprintf("%.3f", s.Rate5())
				ctx[name+".rate.15min"] = fmt.Sprintf("%.3f", s.Rate15())
				ctx[name+".rate.mean"] = fmt.Sprintf("%.3f", s.RateMean())
			}
		})
		l.Println(ctx)
	}
}
