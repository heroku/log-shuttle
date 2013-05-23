package main

import (
	"fmt"
	"log"
	"net"
	"os"
)

var LogShuttleVersion = "0.2"

func main() {
	conf := new(ShuttleConfig)
	conf.ParseFlags()

	if conf.PrintVersion {
		fmt.Println(LogShuttleVersion)
		os.Exit(0)
	}

	reader := NewReader(conf)
	outlet := NewOutlet(conf, reader.Outbox)

	go outlet.Transfer()
	go outlet.Outlet()

	if conf.UseStdin() {
		reader.Input = os.Stdin
		reader.Read()
		outlet.InFLight.Wait()
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
			reader.Input = conn
			go reader.Read()
		}
	}
}
