package main

import (
	"sync"
	"testing"
	"time"
)

func ProduceLogLines(count int, c chan<- LogLine) {
	ll := LogLine{
		line:    []byte("Dolor sit amet, consectetur adipiscing elit praesent ac magna justo.\n"),
		when:    time.Now(),
		rfc3164: false,
	}
	for i := 0; i < count; i++ {
		c <- ll
	}
}

func BenchmarkBatcher(b *testing.B) {
	b.ResetTimer()
	inBatches, outBatches := NewBatchManager(config)
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		logs := make(chan LogLine, config.FrontBuff)
		stats := make(chan NamedValue, config.StatsBuff)
		go ConsumeNamedValues(stats)
		drops := NewCounter(0)
		batcher := NewBatcher(config, drops, stats, logs, inBatches, outBatches)
		wg := new(sync.WaitGroup)
		wg.Add(1)
		b.StartTimer()
		go func() {
			defer wg.Done()
			batcher.Batch()
		}()
		ProduceLogLines(TEST_PRODUCER_LINES, logs)
		close(logs)
		wg.Wait()
		close(stats)
	}
}
