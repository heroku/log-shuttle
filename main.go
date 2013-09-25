package main

import (
	"fmt"
	"os"
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

	reader := NewReader(conf)
	outlet := NewOutlet(conf, reader.Outbox, reader.InFlight, &reader.Drops)

	go outlet.Transfer()
	go outlet.Outlet()

	reader.Read(os.Stdin)
	reader.InFlight.Wait()
	os.Exit(0)

}
