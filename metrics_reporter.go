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

// MetricsReporter handles reporting of metrics to a specified source at a given duration
type MetricsReporter struct {
	registry metrics.Registry
	source   string
	logger   *log.Logger
	lastCounts map[string]int64
	doneCh chan struct{}
}

// NewMetricsReporter returns a properly constructed MetricsReporter
func NewMetricsReporter(r metrics.Registry, source string, l *log.Logger) *MetricsReporter {
	return &MetricsReporter{registry: r, source: source, logger: l, lastCounts: make(map[string]int64), doneCh: make(chan struct{})}
}

func (e MetricsReporter) countDifference(ctx slog.Context, name string, c int64) {
	name = name + ".count"
	lc := e.lastCounts[name]
	ctx[name] = c - lc
	e.lastCounts[name] = c
}

// Emit emits log-shuttle metrics in logfmt compatible formats every d
// duration using the MetricsReporter logger. source is added to the line as
// log_shuttle_stats_source if not empty. It waits for a stop signal
// and will stop emitting metrics when received. Example output:
// space=<space-id> instance=<instance-id> runc-shuttle2019/06/10 12:48:25 batch.fill.count=0
// batch.fill.max=0.000000 batch.fill.mean=0.000000 batch.fill.min=0.000000
// batch.fill.p75=0.000000 batch.fill.p95=0.000000 batch.fill.p99=0.000000 batch.fill.rate.15min=0.000
// batch.fill.rate.1min=0.000 batch.fill.rate.5min=0.000 batch.fill.rate.mean=0.000
// batch.fill.stddev=0.000000 jw-debug-metrics=true lines.batched.count=0 lines.dropped.count=0
// lines.read.count=0 msg.lost.count=0 outlet.inbox.length=0 outlet.post.failure.count=0
// outlet.post.failure.max=0.000000 outlet.post.failure.mean=0.000000 outlet.post.failure.min=0.000000
// outlet.post.failure.p75=0.000000 outlet.post.failure.p95=0.000000 outlet.post.failure.p99=0.000000
// outlet.post.failure.rate.15min=0.000 outlet.post.failure.rate.1min=0.000 outlet.post.failure.rate.5min=0.000
// outlet.post.failure.rate.mean=0.000 outlet.post.failure.stddev=0.000000 outlet.post.success.count=0
// outlet.post.success.max=0.000000 outlet.post.success.mean=0.000000 outlet.post.success.min=0.000000
// outlet.post.success.p75=0.000000 outlet.post.success.p95=0.000000 outlet.post.success.p99=0.000000
// outlet.post.success.rate.15min=0.000 outlet.post.success.rate.1min=0.000 outlet.post.success.rate.5min=0.000
// outlet.post.success.rate.mean=0.000 outlet.post.success.stddev=0.000000
func (e MetricsReporter) Emit(d time.Duration) {
	if d == 0 {
		return
	}
	ticker := time.NewTicker(d)
	defer ticker.Stop()

	for {
		select {
			case <- ticker.C:
				e.emit()
			case <- e.doneCh:
				e.logger.Printf("log_shuttle_stats_source=%s at=Emit msg=closed", e.source)
				return
		}
	}
}

// Stop stops a MetricsEmitter from emitting log to its logger
func (e MetricsReporter) Stop() {
	close(e.doneCh)
}

func (e MetricsReporter) emit() {
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
