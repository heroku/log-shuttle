// +build !windows

package main

import (
	"fmt"
	"log"
	"log/syslog"

	shuttle "github.com/heroku/log-shuttle"
)

func setupLogging(logToSyslog bool, s *shuttle.Shuttle, logger, errLogger *log.Logger) error {
	if !logToSyslog {
		s.Logger = logger
		s.ErrLogger = errLogger
		return nil
	}

	var err error
	s.Logger, err = syslog.NewLogger(syslog.LOG_INFO|syslog.LOG_SYSLOG, 0)
	if err != nil {
		return fmt.Errorf(`error="Unable to setup syslog logger: %s\n"`, err)
	}
	s.ErrLogger, err = syslog.NewLogger(syslog.LOG_ERR|syslog.LOG_SYSLOG, 0)
	if err != nil {
		return fmt.Errorf(`error="Unable to setup syslog error logger: %s\n"`, err)
	}
	return nil
}
