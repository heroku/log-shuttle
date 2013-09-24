package main

import (
	"fmt"
	"log"
	"net"
	"os"
)

const (
	VERSION = "0.2.1"
)

func main() {
	conf := new(ShuttleConfig)
	conf.ParseFlags()

	if conf.PrintVersion {
		fmt.Println(VERSION)
		os.Exit(0)
	}

	reader := NewReader(conf)
	outlet := NewOutlet(conf, reader.Outbox, reader.InFlight, &reader.Drops)

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
