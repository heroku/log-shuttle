package shuttle

import (
	"sync"
)

type Shuttle struct {
	config             ShuttleConfig
	Reader             Reader
	deliverableBatches chan Batch
	programStats       *ProgramStats
	bWaiter, oWaiter   *sync.WaitGroup
}

func NewShuttle(config ShuttleConfig) *Shuttle {
	s := &Shuttle{}
	s.config = config
	return s
}

func (s *Shuttle) Launch() {
	s.programStats = NewProgramStats(s.config.StatsAddr, s.config.StatsBuff)
	s.programStats.Listen()
	go EmitStats(s.programStats, s.config.StatsInterval, s.config.StatsSource)

	s.deliverableBatches = make(chan Batch, s.config.BackBuff)
	// Start outlets, then batches (reverse of Shutdown)
	s.Reader = NewReader(s.config.FrontBuff, s.programStats.Input)
	s.oWaiter = StartOutlets(s.config, s.programStats.Drops, s.programStats.Lost, s.programStats.Input, s.deliverableBatches, NewLogplexBatchFormatter)
	s.bWaiter = StartBatchers(s.config, s.programStats.Drops, s.programStats.Input, s.Reader.Outbox, s.deliverableBatches)
}

func (s *Shuttle) Shutdown() {
	deliverableLogs := s.Reader.Outbox
	stats := s.programStats.Input

	close(deliverableLogs)      // Close the log line channel, all of the batchers will stop once they are done
	s.bWaiter.Wait()            // Wait for them to be done
	close(s.deliverableBatches) // Close the batch channel, all of the outlets will stop once they are done
	s.oWaiter.Wait()            // Wait for them to be done
	close(stats)                // Close the stats channel to shut down any goroutines using it
}
