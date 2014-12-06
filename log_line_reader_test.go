package shuttle

import (
	"io"
	"sync"
	"testing"

	"github.com/rcrowley/go-metrics"
)

const (
	TestProducerLines = 100000
)

var (
	TestData     = []byte("12345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890\n")
	LongTestData = make([]byte, 74568, 74568)
)

type InputProducer struct {
	Total, Curr int
	TotalBytes  int
	Data        []byte
}

func NewInputProducer(c int) *InputProducer {
	curr := 0
	tb := 0
	return &InputProducer{Total: c, Curr: curr, TotalBytes: tb, Data: TestData}
}

func (llp *InputProducer) Read(p []byte) (n int, err error) {
	if llp.Curr > llp.Total {
		return 0, io.EOF
	}
	llp.Curr++
	llp.TotalBytes += len(llp.Data)
	return copy(p, llp.Data), nil
}

func (llp InputProducer) Close() error {
	return nil
}

type TestConsumer struct {
	*sync.WaitGroup
}

func (tc TestConsumer) Consume(in <-chan LogLine) {
	tc.Add(1)
	go func() {
		defer tc.Done()
		for _ = range in {
		}
	}()
}

func doBasicLogLineReaderBenchmark(b *testing.B, frontBuffSize int) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		logs := make(chan LogLine, frontBuffSize)
		rdr := NewLogLineReader(logs, metrics.NewRegistry())
		testConsumer := TestConsumer{new(sync.WaitGroup)}
		testConsumer.Consume(logs)
		llp := NewInputProducer(TestProducerLines)
		b.StartTimer()
		rdr.ReadLogLines(llp)
		b.SetBytes(int64(llp.TotalBytes))
		close(logs)
		testConsumer.Wait()
	}
}

func BenchmarkLogLineReaderWithFrontBuffEqual0(b *testing.B) {
	doBasicLogLineReaderBenchmark(b, 0)
}

func BenchmarkLogLineReaderWithFrontBuffEqual1(b *testing.B) {
	doBasicLogLineReaderBenchmark(b, 1)
}

func BenchmarkLogLineReaderWithFrontBuffEqual100(b *testing.B) {
	doBasicLogLineReaderBenchmark(b, 100)
}

func BenchmarkLogLineReaderWithFrontBuffEqual1000(b *testing.B) {
	doBasicLogLineReaderBenchmark(b, 1000)
}

func BenchmarkLogLineReaderWithFrontBuffEqual10000(b *testing.B) {
	doBasicLogLineReaderBenchmark(b, 10000)
}

func BenchmarkLogLineReaderWithDefaultFrontBuff(b *testing.B) {
	doBasicLogLineReaderBenchmark(b, DefaultFrontBuff)
}
