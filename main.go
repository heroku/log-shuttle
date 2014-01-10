package main

import (
	"fmt"
	"log"
	"os"
	"sync"
)

const (
	VERSION = "0.5.1"
)

func MakeBasicBits(config ShuttleConfig) (reader *Reader, stats chan NamedValue, drops, lost *Counter, logs chan LogLine, deliverableBatches chan *Batch, programStats *ProgramStats, bWaiter, oWaiter *sync.WaitGroup) {
	deliverableBatches = make(chan *Batch, config.NumOutlets*config.NumBatchers)
	logs = make(chan LogLine, config.FrontBuff)
	stats = make(chan NamedValue, config.StatsBuff)
	drops = NewCounter(0)
	lost = NewCounter(0)
	reader = NewReader(logs, stats)
	programStats = NewProgramStats(config.StatsAddr, lost, drops, stats)
	programStats.Run()
	getBatches, returnBatches := NewBatchManager(config, stats)
	// Start outlets, then batches, then readers (reverse of Shutdown)
	oWaiter = StartOutlets(config, drops, lost, stats, deliverableBatches, returnBatches)
	bWaiter = StartBatchers(config, drops, stats, logs, getBatches, deliverableBatches)
	return
}

func Shutdown(deliverableLogs chan LogLine, stats chan NamedValue, deliverableBatches chan *Batch, bWaiter *sync.WaitGroup, oWaiter *sync.WaitGroup) {
	close(deliverableLogs)    // Close the log line channel, all of the batchers will stop once they are done
	bWaiter.Wait()            // Wait for them to be done
	close(deliverableBatches) // Close the batch channel, all of the outlet will stop once they are done
	oWaiter.Wait()            // Wait for them to be done
	close(stats)              // Close the stats channel to shut down any goroutines using it
}

func main() {
	var config ShuttleConfig

	config.ParseFlags()

	if config.PrintVersion {
		fmt.Println(VERSION)
		os.Exit(0)
	}

	if !config.UseStdin() {
		log.Fatalln("No stdin detected.")
	}

	reader, stats, _, _, logs, deliverableBatches, _, batchWaiter, outletWaiter := MakeBasicBits(config)

	reader.Read(os.Stdin)

	// Shutdown everything else.
	Shutdown(logs, stats, deliverableBatches, batchWaiter, outletWaiter)
}
