package shuttle

import (
	"log"
	"os"
)

var (
	Logger    = log.New(os.Stdout, "log-shuttle: ", log.LstdFlags)
	ErrLogger = log.New(os.Stderr, "log-shuttle: ", log.LstdFlags)
)

const (
	VERSION = "0.9.6"
)
