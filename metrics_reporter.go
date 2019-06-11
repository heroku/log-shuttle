package shuttle

import (
	"fmt"
	"time"
	"log"

	"github.com/heroku/slog"
	metrics "github.com/rcrowley/go-metrics"
)

// Percentile info for histograms and the like. These 2 arrays need to match
// length and positions are relevant to each other.
var (
	percentiles     = []float64{0.75, 0.95, 0.99}
	percentileNames = []string{"p75", "p95", "p99"}
)

// Given a t representing a time in ns, convert to seconds, show up to μs precision
func sec(t float64) string {
	return fmt.Sprintf("%.6f", t/1000000000)
}

type Emitter struct {
	registry metrics.Registry
	source   string
	duration time.Duration
	logger   *log.Logger
	lastCounts map[string]int64
}

func NewEmitter(r metrics.Registry, source string, d time.Duration, l *log.Logger) *Emitter {
	return &Emitter{registry: r, source: source, duration: d, logger: l, lastCounts: make(map[string]int64)}
}

func (e Emitter) countDifference(ctx slog.Context, name string, c int64) {
	name = name + ".count"
	lc := e.lastCounts[name]
	ctx[name] = c - lc
	e.lastCounts[name] = c
}

// LogFmtMetricsEmitter emits the metrics in logfmt compatible formats every d
// duration using the provided logger. source is added to the line as
// log_shuttle_stats_source if not empty.
func (e Emitter) Emit() {
	if e.duration == 0 {
		return
	}
	for _ = range time.Tick(e.duration) {
		ctx := slog.Context{}
		if e.source != "" {
			ctx["log_shuttle_stats_source"] = e.source
		}
		e.registry.Each(func(name string, i interface{}) {
			switch metric := i.(type) {
			case metrics.Counter:
				e.countDifference(ctx, name, metric.Count())
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
				e.countDifference(ctx, name, s.Count())
				ctx[name+".min"] = s.Min()
				ctx[name+".max"] = s.Max()
				ctx[name+".mean"] = s.Mean()
				ctx[name+".stddev"] = s.StdDev()
				for i, pn := range percentileNames {
					ctx[name+"."+pn] = ps[i]
				}
			case metrics.Meter:
				s := metric.Snapshot()
				e.countDifference(ctx, name, s.Count())
				ctx[name+".rate.1min"] = s.Rate1()
				ctx[name+".rate.5min"] = s.Rate5()
				ctx[name+".rate.15min"] = s.Rate15()
				ctx[name+".rate.mean"] = s.RateMean()
			case metrics.Timer:
				s := metric.Snapshot()
				ps := s.Percentiles(percentiles)
				e.countDifference(ctx, name, s.Count())
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
		e.logger.Println(ctx)
	}
}
