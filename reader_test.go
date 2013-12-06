package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"sync"
	"testing"
	"time"
)

const (
	TEST_PRODUCER_LINES = 100000
)

var (
	TestData = []byte("12345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890\n")
)

type LogDgramProducer struct {
	Total      int
	TotalBytes int
	Data       []byte
}

func NewLogDgramProducer(c int) *LogDgramProducer {
	return &LogDgramProducer{Total: c, Data: TestData}
}

func writeDgram(conn net.Conn, data []byte) (int, error) {
	return conn.Write(data)
}

func (ldp *LogDgramProducer) Run(fileName string) {
	conn, err := net.Dial("unixgram", fileName)
	if err != nil {
		panic("unable to open: " + fileName)
	}
	for i := 0; i < ldp.Total; i++ {
		for {
			_, err := writeDgram(conn, ldp.Data)
			if err != nil {
				// We seem to send faster than the other goroutine can consume
				// TODO: figure out a better way to catch this error
				if opErr, ok := err.(*net.OpError); ok && opErr.Error() == "write unixgram "+fileName+": no buffer space available" {
					time.Sleep(1 * time.Microsecond)
				} else {
					panic(fmt.Sprintf("error sending line %d: ", i) + err.Error())
				}
			} else {
				break
			}
		}
		ldp.TotalBytes += len(ldp.Data)
	}
	err = conn.Close()
	if err != nil {
		panic("error closing")
	}
}

type InputProducer struct {
	Total, Curr *int
	TotalBytes  *int
	Data        []byte
}

func NewInputProducer(c int) InputProducer {
	curr := 0
	tb := 0
	return InputProducer{Total: &c, Curr: &curr, TotalBytes: &tb, Data: TestData}
}

func (llp InputProducer) Read(p []byte) (n int, err error) {
	if *llp.Curr > *llp.Total {
		return 0, io.EOF
	} else {
		*llp.Curr += 1
		*llp.TotalBytes += len(llp.Data)
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
		logs := make(chan LogLine, frontBuffSize)
		stats := make(chan NamedValue, config.StatsBuff)
		rdr := NewReader(logs, stats)
		testConsumer := TestConsumer{new(sync.WaitGroup)}
		testConsumer.Consume(logs, stats)
		llp := NewInputProducer(TEST_PRODUCER_LINES)
		b.StartTimer()
		rdr.Read(llp)
		b.SetBytes(int64(*llp.TotalBytes))
		close(logs)
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

func doDgramReaderBenchmark(b *testing.B, frontBuffSize int) {
	b.ResetTimer()
	tmpDir := os.Getenv("TMPDIR")
	if tmpDir == "" {
		tmpDir = "/tmp"
	}
	tmpDir, err := ioutil.TempDir(tmpDir, "reader_test")
	if err != nil {
		panic("unable to setup tmpDir: " + tmpDir)
	}

	for i := 0; i < b.N; i++ {
		tmpSocket := fmt.Sprintf("%s/%d", tmpDir, i)
		b.StopTimer()
		logs := make(chan LogLine, frontBuffSize)
		stats := make(chan NamedValue, config.StatsBuff)
		rdr := NewReader(logs, stats)
		cc := make(chan bool)
		testConsumer := TestConsumer{new(sync.WaitGroup)}
		testConsumer.Consume(logs, stats)

		socket := SetupSocket(tmpSocket)

		b.StartTimer()
		go func() {
			rdr.ReadUnixgram(socket, cc)
		}()

		ldp := NewLogDgramProducer(TEST_PRODUCER_LINES)
		ldp.Run(tmpSocket)
		cc <- true

		b.SetBytes(int64(ldp.TotalBytes))
		close(logs)
		close(stats)
		testConsumer.Wait()
		CleanupSocket(tmpSocket)
	}
}

func BenchmarkUnixgramReaderWithFrontBuffEqual0(b *testing.B) {
	doDgramReaderBenchmark(b, 0)
}

func BenchmarkUnixgramReaderWithFrontBuffEqual1(b *testing.B) {
	doDgramReaderBenchmark(b, 1)
}

func BenchmarkUnixgramReaderWithFrontBuffEqual100(b *testing.B) {
	doDgramReaderBenchmark(b, 100)
}

func BenchmarkUnixgramReaderWithFrontBuffEqual1000(b *testing.B) {
	doDgramReaderBenchmark(b, 1000)
}

func BenchmarkUnixgramReaderWithFrontBuffEqual10000(b *testing.B) {
	doDgramReaderBenchmark(b, 10000)
}

func BenchmarkUnixgramReaderWithDefaultFrontBuff(b *testing.B) {
	doDgramReaderBenchmark(b, DEFAULT_FRONT_BUFF)
}
