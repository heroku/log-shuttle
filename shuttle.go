package shuttle

import (
	"sync"
)

// Shuttle is the main entry point into the library
type Shuttle struct {
	config             Config
	Reader             Reader
	deliverableBatches chan Batch
	programStats       *ProgramStats
	bWaiter, oWaiter   *sync.WaitGroup
}

// NewShuttle returns a properly constructed Shuttle with a given config
func NewShuttle(config Config) *Shuttle {
	return &Shuttle{config: config}
}

// Launch a shuttle by spawing it's outlet's, batchers and stats interface
func (s *Shuttle) Launch() {
	s.programStats = NewProgramStats(s.config.StatsAddr, s.config.StatsBuff)
	s.programStats.Listen()
	go EmitStats(s.programStats, s.config.StatsInterval, s.config.StatsSource)

	s.deliverableBatches = make(chan Batch, s.config.BackBuff)
	// Start outlets, then batches (reverse of Shutdown)
	s.Reader = NewReader(s.config.FrontBuff, s.programStats.Input)
	s.oWaiter = startOutlets(s.config, s.programStats.Drops, s.programStats.Lost, s.programStats.Input, s.deliverableBatches, NewLogplexBatchFormatter)
	s.bWaiter = startBatchers(s.config, s.programStats.Drops, s.programStats.Input, s.Reader.Outbox, s.deliverableBatches)
}

// startOutlet launches config.NumOutlets number of outlets and returns a
// waitgroup you can wait on.  When inbox is closed the outlets will finish up
// their output and exit.  Per activity stats are sent via the `stats` channel
func startOutlets(config Config, drops, lost *Counter, stats chan<- NamedValue, inbox <-chan Batch, ff NewHTTPFormatterFunc) *sync.WaitGroup {
	outletWaiter := new(sync.WaitGroup)

	for i := 0; i < config.NumOutlets; i++ {
		outletWaiter.Add(1)
		go func() {
			defer outletWaiter.Done()
			outlet := NewHTTPOutlet(config, drops, lost, stats, inbox, ff)
			outlet.Outlet()
		}()
	}

	return outletWaiter
}

// startBatchers starts config.NumBatchers number of batchers and returns a WaitGroup that you wan wait on.
// When inLogs is closed the batchers will finsih up and exit.
// Per batcher stats are sent via the `stats` channel.
func startBatchers(config Config, drops *Counter, stats chan<- NamedValue, inLogs <-chan LogLine, outBatches chan<- Batch) *sync.WaitGroup {
	batchWaiter := new(sync.WaitGroup)
	for i := 0; i < config.NumBatchers; i++ {
		batchWaiter.Add(1)
		go func() {
			defer batchWaiter.Done()
			batcher := NewBatcher(config.BatchSize, config.WaitDuration, drops, stats, inLogs, outBatches)
			batcher.Batch()
		}()
	}

	return batchWaiter
}

// Shutdown gracefully terminates the shuttle instance, ensuring that anything
// read is batched and delivered
func (s *Shuttle) Shutdown() {
	deliverableLogs := s.Reader.Outbox
	stats := s.programStats.Input

	close(deliverableLogs)      // Close the log line channel, all of the batchers will stop once they are done
	s.bWaiter.Wait()            // Wait for them to be done
	close(s.deliverableBatches) // Close the batch channel, all of the outlets will stop once they are done
	s.oWaiter.Wait()            // Wait for them to be done
	close(stats)                // Close the stats channel to shut down any goroutines using it
}
