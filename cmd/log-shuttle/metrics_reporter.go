package main

import (
	"fmt"
	"log"
	"time"

	"github.com/heroku/log-shuttle/Godeps/_workspace/src/github.com/heroku/slog"
	metrics "github.com/heroku/log-shuttle/Godeps/_workspace/src/github.com/rcrowley/go-metrics"
)

// Percentile info for histograms and the like. These 2 arrays need to match
// length and positions are relevant to each other.
var (
	percentiles     = []float64{0.75, 0.95, 0.99}
	percentileNames = []string{"p75", "p95", "p99"}
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
				ctx[name+".1min.rate"] = s.Rate1()
				ctx[name+".5min.rate"] = s.Rate5()
				ctx[name+".15min.rate"] = s.Rate15()
				ctx[name+".mean.rate"] = s.RateMean()
			case metrics.Timer:
				s := metric.Snapshot()
				ps := s.Percentiles(percentiles)
				ctx[name+".count"] = s.Count()
				ctx[name+".min"] = s.Min()
				ctx[name+".max"] = s.Max()
				ctx[name+".mean"] = s.Mean()
				ctx[name+".stddev"] = s.StdDev()
				for i, pn := range percentileNames {
					// In ns, convert to seconds, show up to μs precision
					ctx[name+"."+pn] = fmt.Sprintf("%.6f", ps[i]/1000000000)
				}
				ctx[name+".1min.rate"] = fmt.Sprintf("%.3f", s.Rate1())
				ctx[name+".5min.rate"] = fmt.Sprintf("%.3f", s.Rate5())
				ctx[name+".15min.rate"] = fmt.Sprintf("%.3f", s.Rate15())
				ctx[name+".mean.rate"] = fmt.Sprintf("%.3f", s.RateMean())
			}
		})
		l.Println(ctx)
	}
}
