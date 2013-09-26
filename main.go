package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"sync"
)

const (
	VERSION = "0.2.2"
)

func main() {
	conf := new(ShuttleConfig)
	conf.ParseFlags()

	if conf.PrintVersion {
		fmt.Println(VERSION)
		os.Exit(0)
	}

	inFlight := new(sync.WaitGroup)
	drops := new(Counter)
	frontBuff := make(chan string, conf.FrontBuff)

	outlet := NewOutlet(conf, inFlight, drops, frontBuff)
	reader := NewReader(conf, inFlight, drops, frontBuff)

	go outlet.Transfer()
	go outlet.Outlet()

	if conf.UseStdin() {
		reader.Read(os.Stdin)
		reader.InFlight.Wait()
		os.Exit(0)
	}

	if conf.UseSocket() {
		l, err := net.Listen("unix", conf.Socket)
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
