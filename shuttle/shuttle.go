
package shuttle

import (
	"sync"
)

type Shuttle struct{
	Reader Reader
	deliverableBatches chan Batch
	programStats *ProgramStats
	bWaiter, oWaiter *sync.WaitGroup
}

func NewShuttle(config ShuttleConfig) (*Shuttle) {
	s := &Shuttle{}
	s.programStats = NewProgramStats(config.StatsAddr, config.StatsBuff)
	s.programStats.Listen()
	go EmitStats(s.programStats, config.StatsInterval, config.StatsSource)

	s.deliverableBatches = make(chan Batch, config.BackBuff)
	// Start outlets, then batches (reverse of Shutdown)
	s.Reader = NewReader(config.FrontBuff, s.programStats.Input)
	s.oWaiter = StartOutlets(config, s.programStats.Drops, s.programStats.Lost, s.programStats.Input, s.deliverableBatches)
	s.bWaiter = StartBatchers(config, s.programStats.Drops, s.programStats.Input, s.Reader.Outbox, s.deliverableBatches)

	return s;
}


func (s *Shuttle) Shutdown() {
	deliverableLogs := s.Reader.Outbox
	stats := s.programStats.Input

	close(deliverableLogs)    // Close the log line channel, all of the batchers will stop once they are done
	s.bWaiter.Wait()            // Wait for them to be done
	close(s.deliverableBatches) // Close the batch channel, all of the outlets will stop once they are done
	s.oWaiter.Wait()            // Wait for them to be done
	close(stats)              // Close the stats channel to shut down any goroutines using it
}
