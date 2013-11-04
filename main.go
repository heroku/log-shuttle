package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"sync"
)

const (
	VERSION = "0.3.2"
)

func Shutdown(dLogLines chan *LogLine, dBatches chan *Batch, bWaiter *sync.WaitGroup, oWaiter *sync.WaitGroup) {
	close(dLogLines) // Close the log line channel, all of the batchers will stop once they are done
	bWaiter.Wait()   // Wait for them to be done
	close(dBatches)  // Close the batch channel, all of the outlet will stop once they are done
	oWaiter.Wait()   // Wait for them to be done
}

func main() {
	var config ShuttleConfig
	config.ParseFlags()

	if config.PrintVersion {
		fmt.Println(VERSION)
		os.Exit(0)
	}

	deliverables := make(chan *Batch, config.NumOutlets*config.NumBatchers)
	programStats := &ProgramStats{}

	getBatches, returnBatches := NewBatchManager(config)

	reader := NewReader(config.FrontBuff)

	// Start outlets, then batches, then readers (reverse of Shutdown)
	outletWaiter := StartOutlets(config, programStats, deliverables, returnBatches)
	batchWaiter := StartBatchers(config, programStats, reader.Outbox, getBatches, deliverables)

	if config.UseStdin() {
		reader.Read(os.Stdin, programStats)
		Shutdown(reader.Outbox, deliverables, batchWaiter, outletWaiter)
		os.Exit(0)
	}

	if config.UseSocket() {
		l, err := net.Listen("unix", config.Socket)
		if err != nil {
			log.Fatal(err)
		}

		for {
			conn, err := l.Accept()
			if err != nil {
				fmt.Fprintf(os.Stderr, "accept-err=%s\n", err)
				continue
			}

			go reader.Read(conn, programStats)
		}
	}

}
