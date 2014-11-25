package main

import (
	"fmt"
	"log"
	"log/syslog"
	"os"

	"github.com/heroku/log-shuttle/shuttle"
)

func main() {
	config := shuttle.NewConfig()
	config.ParseFlags()

	var err error

	if config.PrintVersion {
		fmt.Println(shuttle.VERSION)
		os.Exit(0)
	}

	if !config.UseStdin() {
		shuttle.ErrLogger.Fatalln("No stdin detected.")
	}

	if config.LogToSyslog {
		shuttle.Logger, err = syslog.NewLogger(syslog.LOG_INFO|syslog.LOG_SYSLOG, 0)
		if err != nil {
			log.Fatalf("Unable to setup syslog logger: %s\n", err)
		}
		shuttle.ErrLogger, err = syslog.NewLogger(syslog.LOG_ERR|syslog.LOG_SYSLOG, 0)
		if err != nil {
			log.Fatalf("Unable to setup syslog error logger: %s\n", err)
		}
	}

	s := shuttle.NewShuttle(config)
	s.Launch()

	// Blocks until closed
	s.Reader.Read(os.Stdin)

	// Shutdown everything else.
	s.Shutdown()
}
