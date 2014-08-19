package main

import (
	"fmt"
	"log"
	"log/syslog"
	"os"
	"sync"

	"github.com/heroku/log-shuttle/shuttle"
)

func MakeBasicBits(config shuttle.ShuttleConfig) (reader shuttle.Reader, deliverableBatches chan shuttle.Batch, programStats *shuttle.ProgramStats, bWaiter, oWaiter *sync.WaitGroup) {
	programStats = shuttle.NewProgramStats(config.StatsAddr, config.StatsBuff)
	programStats.Listen()
	go shuttle.EmitStats(programStats, config.StatsInterval, config.StatsSource)

	deliverableBatches = make(chan shuttle.Batch, config.BackBuff)
	// Start outlets, then batches (reverse of Shutdown)
	reader = shuttle.NewReader(config.FrontBuff, programStats.Input)
	oWaiter = shuttle.StartOutlets(config, programStats.Drops, programStats.Lost, programStats.Input, deliverableBatches)
	bWaiter = shuttle.StartBatchers(config, programStats.Drops, programStats.Input, reader.Outbox, deliverableBatches)
	return
}

func Shutdown(deliverableLogs chan shuttle.LogLine, stats chan shuttle.NamedValue, deliverableBatches chan shuttle.Batch, bWaiter *sync.WaitGroup, oWaiter *sync.WaitGroup) {
	close(deliverableLogs)    // Close the log line channel, all of the batchers will stop once they are done
	bWaiter.Wait()            // Wait for them to be done
	close(deliverableBatches) // Close the batch channel, all of the outlets will stop once they are done
	oWaiter.Wait()            // Wait for them to be done
	close(stats)              // Close the stats channel to shut down any goroutines using it
}

func main() {
	var config shuttle.ShuttleConfig
	var err error

	config.ParseFlags()

	if config.PrintVersion {
		fmt.Println(shuttle.VERSION)
		os.Exit(0)
	}

	if !config.UseStdin() {
		shuttle.ErrLogger.Fatalln("No stdin detected.")
	}

	if config.LogToSyslog {
		shuttle.Logger, err = syslog.NewLogger(syslog.LOG_INFO|syslog.LOG_SYSLOG, 0)
		if err != nil {
			log.Fatalf("Unable to setup syslog logger: %s\n", err)
		}
		shuttle.ErrLogger, err = syslog.NewLogger(syslog.LOG_ERR|syslog.LOG_SYSLOG, 0)
		if err != nil {
			log.Fatalf("Unable to setup syslog error logger: %s\n", err)
		}
	}

	reader, deliverableBatches, programStats, batchWaiter, outletWaiter := MakeBasicBits(config)

	// Blocks until closed
	reader.Read(os.Stdin)

	// Shutdown everything else.
	Shutdown(reader.Outbox, programStats.Input, deliverableBatches, batchWaiter, outletWaiter)
}
