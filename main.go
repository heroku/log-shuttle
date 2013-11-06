package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"sync"
)

const (
	VERSION      = "0.3.2"
	SOCKET_TYPE  = "unixgram"
	SOCKET_PERMS = 0666
)

func Shutdown(dLogLines chan *LogLine, dBatches chan *Batch, bWaiter *sync.WaitGroup, oWaiter *sync.WaitGroup) {
	close(dLogLines) // Close the log line channel, all of the batchers will stop once they are done
	bWaiter.Wait()   // Wait for them to be done
	close(dBatches)  // Close the batch channel, all of the outlet will stop once they are done
	oWaiter.Wait()   // Wait for them to be done
}

func Exists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
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
		ua, err := net.ResolveUnixAddr(SOCKET_TYPE, config.Socket)
		if err != nil {
			log.Fatal("Resolving Unix Address: ", err)
		}

		if Exists(config.Socket) {
			err := os.Remove(config.Socket)
			if err != nil {
				log.Fatal("Removing old socket: ", err)
			}
		}

		l, err := net.ListenUnixgram(SOCKET_TYPE, ua)
		if err != nil {
			log.Fatal("Listening on Socket: ", err)
		}

		//Change permissions so anyone can write to it
		err = os.Chmod(config.Socket, SOCKET_PERMS)
		if err != nil {
			log.Fatal("Chmoding Socket: ", err)
		}

		reader.ReadUnixgram(l, programStats)
		Shutdown(reader.Outbox, deliverables, batchWaiter, outletWaiter)
		os.Exit(0)
	}
}
