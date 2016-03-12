package shuttle

import (
	"io"
	"io/ioutil"
	"log"
	"sync"

	metrics "github.com/rcrowley/go-metrics"
)

// Default logger to /dev/null
var (
	discardLogger = log.New(ioutil.Discard, "", 0)
)

// Shuttle is the main entry point into the library
type Shuttle struct {
	LogLineReader
	config           Config
	Batches          chan Batch
	readers          []*LogLineReader
	MetricsRegistry  metrics.Registry
	oWaiter, rWaiter *sync.WaitGroup
	Drops, Lost      *Counter
	NewFormatterFunc NewHTTPFormatterFunc
	Logger           *log.Logger
	ErrLogger        *log.Logger
}

// NewShuttle returns a properly constructed Shuttle with a given config
func NewShuttle(config Config) *Shuttle {
	b := make(chan Batch, config.BackBuff)
	mr := metrics.NewRegistry()

	return &Shuttle{
		config:           config,
		Batches:          b,
		Drops:            NewCounter(0),
		Lost:             NewCounter(0),
		MetricsRegistry:  mr,
		NewFormatterFunc: config.FormatterFunc,
		readers:          make([]*LogLineReader, 0),
		oWaiter:          new(sync.WaitGroup),
		rWaiter:          new(sync.WaitGroup),
		Logger:           discardLogger,
		ErrLogger:        discardLogger,
	}
}

// Launch a shuttle by spawing it's outlets and batchers (in that order), which
// is the reverse of shutdown.
func (s *Shuttle) Launch() {
	s.startOutlets()
	for _, rdr := range s.readers {
		s.rWaiter.Add(1)
		go func(rdr *LogLineReader) {
			rdr.ReadLines()
			s.rWaiter.Done()
		}(rdr)
	}
}

// startOutlet launches config.NumOutlets number of outlets. When inbox is
// closed the outlets will finish up their output and exit.
func (s *Shuttle) startOutlets() {
	for i := 0; i < s.config.NumOutlets; i++ {
		s.oWaiter.Add(1)
		go func() {
			outlet := NewHTTPOutlet(s)
			outlet.Outlet()
			s.oWaiter.Done()
		}()
	}
}

// LoadReader into the shuttle for processing it's lines. Use this if you want
// log-shuttle to track the readers for you. The errors returned by ReadLogLines
// are discarded.
func (s *Shuttle) LoadReader(rdr io.ReadCloser) {
	r := NewLogLineReader(rdr, s)
	s.readers = append(s.readers, r)
}

// CloseReaders closes all tracked readers and returns any errors returned by
// Close()ing the readers
func (s *Shuttle) CloseReaders() []error {
	var errors []error
	for _, closer := range s.readers {
		if err := closer.Close(); err != nil {
			errors = append(errors, err)
		}
	}
	return errors
}

// WaitForReadersToFinish to finish reading
func (s *Shuttle) WaitForReadersToFinish() {
	s.rWaiter.Wait()
}

// DockReaders closes all tracked readers and waits for all reading go routines
// to finish.
func (s *Shuttle) DockReaders() []error {
	errors := s.CloseReaders()
	s.WaitForReadersToFinish()
	return errors
}

// Land gracefully terminates the shuttle instance, ensuring that anything
// read is batched and delivered. A panic is likely to happen if Land() is
// called before any readers passed to any ReadLogLines() calls aren't closed.
func (s *Shuttle) Land() {
	s.DockReaders()
	close(s.Batches) // Close the batch channel, all of the outlets will stop once they are done
	s.oWaiter.Wait() // Wait for them to be done
}
