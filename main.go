package main

import (
	"fmt"
	"log"
	"net"
	"os"
)

const (
	VERSION = "0.3.1"
)

func main() {
	var config ShuttleConfig
	config.ParseFlags()

	if config.PrintVersion {
		fmt.Println(VERSION)
		os.Exit(0)
	}

	deliverables := make(chan *Batch)
	programStats := &Stats{}

	getBatches, returnBatches := NewBatchManager(config)

	reader := NewReader(config, programStats)

	for nb := 0; nb < config.NumBatchers; nb++ {
		go NewBatcher(config, reader.Outbox, getBatches, deliverables).Batch()
	}

	for no := 0; no < config.NumOutlets; no++ {
		go NewOutlet(config, programStats, deliverables, returnBatches).Outlet()
	}

	if config.UseStdin() {
		reader.Read(os.Stdin)
		programStats.InFlight.Wait()
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

			go reader.Read(conn)
		}
	}

}
