package shuttle

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
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		s := NewShuttle(config)
		batcher := NewBatcher(s)
		wg := new(sync.WaitGroup)
		wg.Add(1)
		b.StartTimer()
		go func() {
			defer wg.Done()
			batcher.Batch()
		}()
		ProduceLogLines(TestProducerLines, s.LogLines)
		close(s.LogLines)
		wg.Wait()
	}
}
