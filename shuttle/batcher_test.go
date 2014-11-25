package shuttle

import (
	"sync"
	"testing"
	"time"

	metrics "github.com/rcrowley/go-metrics"
)

func ProduceLogLines(count int, c chan<- LogLine) {
	ll := LogLine{
		line: TestData,
		when: time.Now(),
	}
	for i := 0; i < count; i++ {
		c <- ll
	}
}

func BenchmarkBatcher(b *testing.B) {
	b.ResetTimer()
	outBatches := make(chan Batch)
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		logs := make(chan LogLine, config.FrontBuff)
		batcher := NewBatcher(config.BatchSize, config.Timeout, NewCounter(0), metrics.NewRegistry(), logs, outBatches)
		wg := new(sync.WaitGroup)
		wg.Add(1)
		b.StartTimer()
		go func() {
			defer wg.Done()
			batcher.Batch()
		}()
		ProduceLogLines(TestProducerLines, logs)
		close(logs)
		wg.Wait()
	}
}
