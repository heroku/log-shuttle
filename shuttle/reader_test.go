package shuttle

import (
	"io"
	"sync"
	"testing"
)

const (
	TEST_PRODUCER_LINES = 100000
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
	} else {
		llp.Curr += 1
		llp.TotalBytes += len(llp.Data)
		return copy(p, llp.Data), nil
	}
}

func (llp InputProducer) Close() error {
	return nil
}

type TestConsumer struct {
	*sync.WaitGroup
}

func (tc TestConsumer) Consume(in <-chan LogLine, stats <-chan NamedValue) {
	tc.Add(1)
	go func() {
		defer tc.Done()
		for _ = range in {
		}
	}()
	tc.Add(1)
	go func() {
		defer tc.Done()
		ConsumeNamedValues(stats)
	}()
}

func ConsumeNamedValues(c <-chan NamedValue) {
	for _ = range c {
	}
}

func doBasicReaderBenchmark(b *testing.B, frontBuffSize int) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		stats := make(chan NamedValue, config.StatsBuff)
		rdr := NewReader(frontBuffSize, stats)
		testConsumer := TestConsumer{new(sync.WaitGroup)}
		testConsumer.Consume(rdr.Outbox, stats)
		llp := NewInputProducer(TEST_PRODUCER_LINES)
		b.StartTimer()
		rdr.Read(llp)
		b.SetBytes(int64(llp.TotalBytes))
		close(rdr.Outbox)
		close(stats)
		testConsumer.Wait()
	}
}

func BenchmarkReaderWithFrontBuffEqual0(b *testing.B) {
	doBasicReaderBenchmark(b, 0)
}

func BenchmarkReaderWithFrontBuffEqual1(b *testing.B) {
	doBasicReaderBenchmark(b, 1)
}

func BenchmarkReaderWithFrontBuffEqual100(b *testing.B) {
	doBasicReaderBenchmark(b, 100)
}

func BenchmarkReaderWithFrontBuffEqual1000(b *testing.B) {
	doBasicReaderBenchmark(b, 1000)
}

func BenchmarkReaderWithFrontBuffEqual10000(b *testing.B) {
	doBasicReaderBenchmark(b, 10000)
}

func BenchmarkReaderWithDefaultFrontBuff(b *testing.B) {
	doBasicReaderBenchmark(b, DEFAULT_FRONT_BUFF)
}
