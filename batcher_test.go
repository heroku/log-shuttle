package main

import (
	"sync"
	"testing"
	"time"
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
	stats := make(chan NamedValue, config.StatsBuff)
	go ConsumeNamedValues(stats)
	inBatches, outBatches := NewBatchManager(config, stats)
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		logs := make(chan LogLine, config.FrontBuff)
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
	}
}

func TestLongBatchWrite(t *testing.T) {
	batch := NewBatch(&config)
	batch.Write(LogLine{line: LongTestData, when: time.Now()})
	if batch.MsgCount != 8 {
		t.Fatalf("MsgCount should be 8, but is %d", batch.MsgCount)
	}
}
