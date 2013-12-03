package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"sync"
)

const (
	VERSION      = "0.4.0"
	SOCKET_TYPE  = "unixgram"
	SOCKET_PERMS = 0666
)

func MakeBasicBits(config ShuttleConfig) (*Reader, chan *Batch, *ProgramStats, *sync.WaitGroup, *sync.WaitGroup) {
	deliverables := make(chan *Batch, config.NumOutlets*config.NumBatchers)
	programStats := &ProgramStats{}
	getBatches, returnBatches := NewBatchManager(config)
	reader := NewReader(config.FrontBuff)
	// Start outlets, then batches, then readers (reverse of Shutdown)
	oWaiter := StartOutlets(config, programStats, deliverables, returnBatches)
	bWaiter := StartBatchers(config, programStats, reader.Outbox, getBatches, deliverables)
	return reader, deliverables, programStats, bWaiter, oWaiter
}

func Shutdown(dLogLines chan LogLine, dBatches chan *Batch, bWaiter *sync.WaitGroup, oWaiter *sync.WaitGroup) {
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
	var unixgramCloseChannel chan bool

	stdinWaiter := new(sync.WaitGroup)
	socketWaiter := new(sync.WaitGroup)

	config.ParseFlags()

	if config.PrintVersion {
		fmt.Println(VERSION)
		os.Exit(0)
	}

	if !config.UseStdin() && !config.UseSocket() {
		log.Fatalln("No stdin detected or socket used.")
	}

	reader, deliverables, programStats, batchWaiter, outletWaiter := MakeBasicBits(config)

	if config.UseStdin() {
		stdinWaiter.Add(1)
		go func() {
			reader.Read(os.Stdin, programStats)
			stdinWaiter.Done()
		}()
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

		unixgramCloseChannel = make(chan bool)
		socketWaiter.Add(1)
		go func() {
			reader.ReadUnixgram(l, programStats, unixgramCloseChannel)
			socketWaiter.Done()
		}()
	}

	if config.UseStdin() {
		stdinWaiter.Wait()
		if config.UseSocket() {
			unixgramCloseChannel <- true
		}
	}

	//TODO: Signal handler to gracefully shutdown the socket listener on SIGTERM
	if config.UseSocket() {
		socketWaiter.Wait()
		err := os.Remove(config.Socket)
		if err != nil {
			log.Println("Error removing socket: ", err)
		}
	}

	// Shutdown everything else.
	Shutdown(reader.Outbox, deliverables, batchWaiter, outletWaiter)
}
