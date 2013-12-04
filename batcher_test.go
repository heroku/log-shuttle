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
		inLogs := make(chan LogLine, config.FrontBuff)
		programStats := NewProgramStats(inLogs, inBatches)
		go ConsumeNamedValues(programStats.StatsChannel)
		batcher := NewBatcher(config, inLogs, inBatches, outBatches)
		wg := new(sync.WaitGroup)
		wg.Add(1)
		b.StartTimer()
		go func() {
			defer wg.Done()
			batcher.Batch(programStats)
		}()
		ProduceLogLines(TEST_PRODUCER_LINES, inLogs)
		close(inLogs)
		wg.Wait()
		close(programStats.StatsChannel)
	}
}
