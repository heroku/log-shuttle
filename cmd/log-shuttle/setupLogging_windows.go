package main

import (
	"fmt"
	"log"

	shuttle "github.com/heroku/log-shuttle"
)

func setupLogging(logToSyslog bool, s *shuttle.Shuttle, logger, errLogger *log.Logger) error {
	// Setup the loggers before doing anything else
	if logToSyslog {
		return fmt.Errorf("syslog unavailable on Windows")
	}
	s.Logger = logger
	s.ErrLogger = errLogger
	return nil
}
