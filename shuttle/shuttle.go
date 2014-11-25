package shuttle

import (
	"sync"

	metrics "github.com/rcrowley/go-metrics"
)

// Shuttle is the main entry point into the library
type Shuttle struct {
	Reader
	Drops, Lost     *Counter
	Batches         chan Batch
	Logs            chan LogLine
	MetricsRegistry metrics.Registry

	config                    Config
	batchWaiter, outletWaiter *sync.WaitGroup
}

// NewShuttle returns a properly constructed Shuttle with a given config
func NewShuttle(config Config) *Shuttle {
	logs := make(chan LogLine, config.FrontBuff)
	mRegistry := metrics.NewRegistry()
	return &Shuttle{
		Drops:           NewCounter(0),
		Lost:            NewCounter(0),
		Logs:            logs,
		Batches:         make(chan Batch, config.BackBuff),
		config:          config,
		MetricsRegistry: mRegistry,
		Reader:          NewReader(logs, mRegistry),
		outletWaiter:    new(sync.WaitGroup),
		batchWaiter:     new(sync.WaitGroup),
	}
}

// Launch a shuttle by spawing it's outlets & batchers
func (s *Shuttle) Launch() {
	// Start outlets, then batches (reverse of Shutdown)
	s.startOutlets()
	s.startBatchers()
}

// startOutlets launches config.NumOutlets number of outlets incrementing the
// waitgroup as necessary.
func (s *Shuttle) startOutlets() {
	for i := 0; i < s.config.NumOutlets; i++ {
		s.outletWaiter.Add(1)
		go func() {
			defer s.outletWaiter.Done()
			outlet := NewHTTPOutlet(s.config, s.Drops, s.Lost, s.MetricsRegistry, s.Batches, NewLogplexBatchFormatter)
			outlet.Outlet()
		}()
	}
}

// startBatchers starts config.NumBatchers number of batchers incremeting the
// waitgroup as necessary.
func (s *Shuttle) startBatchers() {
	for i := 0; i < s.config.NumBatchers; i++ {
		s.batchWaiter.Add(1)
		go func() {
			defer s.batchWaiter.Done()
			batcher := NewBatcher(s.config.BatchSize, s.config.WaitDuration, s.Drops, s.MetricsRegistry, s.Logs, s.Batches)
			batcher.Batch()
		}()
	}
}

// Shutdown gracefully terminates the shuttle instance, ensuring that anything
// read is batched and delivered
func (s *Shuttle) Shutdown() {

	close(s.Logs)         // Close the log line channel, all of the batchers will stop once they are done
	s.batchWaiter.Wait()  // Wait for them to be done
	close(s.Batches)      // Close the batch channel, all of the outlets will stop once they are done
	s.outletWaiter.Wait() // Wait for them to be done
}
