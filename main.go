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

func Exists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

func cleanUpSocket(path string) error {
	if Exists(path) {
		return os.Remove(path)
	}
	return nil
}

func SetupSocket(path string) *net.UnixConn {
	ua, err := net.ResolveUnixAddr(SOCKET_TYPE, path)
	if err != nil {
		log.Fatal("Resolving Unix Address: ", err)
	}

	err = cleanUpSocket(path)
	if err != nil {
		log.Fatal("Removing old socket: ", err)
	}

	l, err := net.ListenUnixgram(SOCKET_TYPE, ua)
	if err != nil {
		log.Fatal("Listening on Socket: ", err)
	}

	//Change permissions so anyone can write to it
	err = os.Chmod(path, SOCKET_PERMS)
	if err != nil {
		log.Fatal("Chmoding Socket: ", err)
	}

	return l
}

func CleanupSocket(path string) {
	err := os.Remove(path)
	if err != nil {
		log.Println("Error removing socket: ", err)
	}
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

	reader, stats, _, _, logs, deliverableBatches, _, batchWaiter, outletWaiter := MakeBasicBits(config)

	if config.UseStdin() {
		stdinWaiter.Add(1)
		go func() {
			reader.Read(os.Stdin)
			stdinWaiter.Done()
		}()
	}

	if config.UseSocket() {

		socket := SetupSocket(config.Socket)

		unixgramCloseChannel = make(chan bool)
		socketWaiter.Add(1)
		go func() {
			reader.ReadUnixgram(socket, unixgramCloseChannel)
			socketWaiter.Done()
		}()
	}

	if config.UseStdin() {
		stdinWaiter.Wait()
		if config.UseSocket() {
			unixgramCloseChannel <- true
		}
	}

	// TODO(edwardam): Signal handler to gracefully shutdown the socket listener on SIGTERM
	if config.UseSocket() {
		socketWaiter.Wait()
		CleanupSocket(config.Socket)
	}

	// Shutdown everything else.
	Shutdown(logs, stats, deliverableBatches, batchWaiter, outletWaiter)
}
