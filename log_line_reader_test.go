package shuttle

import (
	"io"
	"sync"
	"testing"
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
	return &InputProducer{Total: c, Data: TestData}
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
	sync.WaitGroup
}

func (tc *TestConsumer) Consume(in <-chan Batch) {
	tc.Add(1)
	go func() {
		defer tc.Done()
		for _ = range in {
		}
	}()
}

func doBasicLogLineReaderBenchmark(b *testing.B, backBuffSize int) {
	b.ResetTimer()
	var tb int
	var tc TestConsumer
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		batches := make(chan Batch, backBuffSize)
		tc.Consume(batches)
		s := NewShuttle(NewConfig())
		llp := NewInputProducer(TestProducerLines)
		rdr := NewLogLineReader(llp, s)
		b.StartTimer()

		rdr.ReadLines()
		tb += llp.TotalBytes
		close(batches)
		tc.Wait()
	}
	b.SetBytes(int64(tb / b.N))
}

func BenchmarkLogLineReaderWithBackBuffEqual0(b *testing.B) {
	doBasicLogLineReaderBenchmark(b, 0)
}

func BenchmarkLogLineReaderWithBackBuffEqual1(b *testing.B) {
	doBasicLogLineReaderBenchmark(b, 1)
}

func BenchmarkLogLineReaderWithBackBuffEqual100(b *testing.B) {
	doBasicLogLineReaderBenchmark(b, 100)
}

func BenchmarkLogLineReaderWithBackBuffEqual1000(b *testing.B) {
	doBasicLogLineReaderBenchmark(b, 1000)
}

func BenchmarkLogLineReaderWithBackBuffEqual10000(b *testing.B) {
	doBasicLogLineReaderBenchmark(b, 10000)
}

func BenchmarkLogLineReaderWithDefaultBackBuff(b *testing.B) {
	doBasicLogLineReaderBenchmark(b, DefaultBackBuff)
}
