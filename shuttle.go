package shuttle

import (
	"io"
	"io/ioutil"
	"log"
	"sync"

	metrics "github.com/heroku/log-shuttle/Godeps/_workspace/src/github.com/rcrowley/go-metrics"
)

// Default logger to /dev/null
var (
	discardLogger = log.New(ioutil.Discard, "", 0)
)

// Shuttle is the main entry point into the library
type Shuttle struct {
	LogLineReader
	config                    Config
	LogLines                  chan LogLine
	Batches                   chan Batch
	readers                   []io.ReadCloser
	MetricsRegistry           metrics.Registry
	bWaiter, oWaiter, rWaiter *sync.WaitGroup
	Drops, Lost               *Counter
	NewFormatterFunc          NewHTTPFormatterFunc
	Logger                    *log.Logger
	ErrLogger                 *log.Logger
}

// NewShuttle returns a properly constructed Shuttle with a given config
func NewShuttle(config Config) *Shuttle {
	ll := make(chan LogLine, config.FrontBuff)
	mr := metrics.NewRegistry()

	return &Shuttle{
		config:           config,
		LogLineReader:    NewLogLineReader(ll, mr),
		LogLines:         ll,
		Batches:          make(chan Batch, config.BackBuff),
		Drops:            NewCounter(0),
		Lost:             NewCounter(0),
		MetricsRegistry:  mr,
		NewFormatterFunc: config.FormatterFunc,
		readers:          make([]io.ReadCloser, 0),
		oWaiter:          new(sync.WaitGroup),
		bWaiter:          new(sync.WaitGroup),
		rWaiter:          new(sync.WaitGroup),
		Logger:           discardLogger,
		ErrLogger:        discardLogger,
	}
}

// Launch a shuttle by spawing it's outlets and batchers (in that order), which
// is the reverse of shutdown.
func (s *Shuttle) Launch() {
	s.startOutlets()
	s.startBatchers()
}

// startOutlet launches config.NumOutlets number of outlets. When inbox is
// closed the outlets will finish up their output and exit.
func (s *Shuttle) startOutlets() {
	for i := 0; i < s.config.NumOutlets; i++ {
		s.oWaiter.Add(1)
		go func() {
			defer s.oWaiter.Done()
			outlet := NewHTTPOutlet(s)
			outlet.Outlet()
		}()
	}
}

// startBatchers starts config.NumBatchers number of batchers.  When inLogs is
// closed the batchers will finsih up and exit.
func (s *Shuttle) startBatchers() {
	for i := 0; i < s.config.NumBatchers; i++ {
		s.bWaiter.Add(1)
		go func() {
			defer s.bWaiter.Done()
			batcher := NewBatcher(s)
			batcher.Batch()
		}()
	}
}

// LoadReader into the shuttle for processing it's lines. Use this if you want
// log-shuttle to track the readers for you.
func (s *Shuttle) LoadReader(rdr io.ReadCloser) {
	s.rWaiter.Add(1)
	s.readers = append(s.readers, rdr)
	go func() {
		s.ReadLogLines(rdr)
		s.rWaiter.Done()
	}()
}

// CloseReaders closes all tracked readers.
func (s *Shuttle) CloseReaders() []error {
	errors := make([]error, 0, len(s.readers))
	for _, closer := range s.readers {
		errors = append(errors, closer.Close())
	}
	return errors
}

// DockReaders closes all tracked readers and waits for all reading go routines
// to finish.
func (s *Shuttle) DockReaders() []error {
	errors := s.CloseReaders()
	s.rWaiter.Wait()
	return errors
}

// Land gracefully terminates the shuttle instance, ensuring that anything
// read is batched and delivered. A panic is likely to happen if Land() is
// called before any readers passed to any ReadLogLines() calls aren't closed.
func (s *Shuttle) Land() {
	s.DockReaders()
	close(s.LogLines) // Close the log line channel, all of the batchers will stop once they are done
	s.bWaiter.Wait()  // Wait for them to be done
	close(s.Batches)  // Close the batch channel, all of the outlets will stop once they are done
	s.oWaiter.Wait()  // Wait for them to be done
}
