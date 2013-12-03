package main

import (
	"io"
	"sync"
	"testing"
)

const (
	TEST_PRODUCER_LINES = 100000
)

type LogLineProducer struct {
	total, curr *int
	data        []byte
}

func NewLogLineProducer(c int) LogLineProducer {
	var curr = 0
	return LogLineProducer{total: &c, curr: &curr, data: []byte("Dolor sit amet, consectetur adipiscing elit praesent ac magna justo.\n")}
}

func (llp LogLineProducer) Read(p []byte) (n int, err error) {
	if *llp.curr > *llp.total {
		return 0, io.EOF
	} else {
		*llp.curr += 1
		return copy(p, llp.data), nil
	}
}

func (llp LogLineProducer) Close() error {
	return nil
}

type TestConsumer struct {
	*sync.WaitGroup
}

func (tc TestConsumer) Consume(in <-chan LogLine) {
	defer tc.Done()
	for _ = range in {
	}
}

func doBasicReaderBenchmark(b *testing.B, rdr *Reader) {
	testConsumer := TestConsumer{new(sync.WaitGroup)}
	testConsumer.Add(1)
	go testConsumer.Consume(rdr.Outbox)
	programStats := &ProgramStats{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		llp := NewLogLineProducer(TEST_PRODUCER_LINES)
		b.SetBytes(int64(len(llp.data) * *llp.total))
		b.StartTimer()
		rdr.Read(llp, programStats)
	}
	close(rdr.Outbox)
	testConsumer.Wait()
}

func BenchmarkReaderWithFrontBuffEqual0(b *testing.B) {
	reader := NewReader(0)
	doBasicReaderBenchmark(b, reader)
}

func BenchmarkReaderWithFrontBuffEqual1(b *testing.B) {
	reader := NewReader(1)
	doBasicReaderBenchmark(b, reader)
}

func BenchmarkReaderWithFrontBuffEqual100(b *testing.B) {
	reader := NewReader(100)
	doBasicReaderBenchmark(b, reader)
}

func BenchmarkReaderWithFrontBuffEqual1000(b *testing.B) {
	reader := NewReader(1000)
	doBasicReaderBenchmark(b, reader)
}

func BenchmarkReaderWithFrontBuffEqual10000(b *testing.B) {
	reader := NewReader(10000)
	doBasicReaderBenchmark(b, reader)
}

func BenchmarkReaderWithDefaultFrontBuff(b *testing.B) {
	var config ShuttleConfig
	reader := NewReader(config.FrontBuff)
	doBasicReaderBenchmark(b, reader)
}
