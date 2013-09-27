package main

import (
	"fmt"
	"log"
	"net"
	"os"
)

const (
	VERSION = "0.2.2"
)

func main() {
	var config ShuttleConfig
	config.ParseFlags()

	if config.PrintVersion {
		fmt.Println(VERSION)
		os.Exit(0)
	}

	reader := NewReader(config)
	batcher := NewBatcher(config, reader.Outbox)
	outlet := NewOutlet(config, reader.InFlight, reader.Drops, batcher.Outbox, batcher.Batches)

	go batcher.Batch()
	go outlet.Outlet()

	if config.UseStdin() {
		reader.Read(os.Stdin)
		reader.InFlight.Wait()
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
