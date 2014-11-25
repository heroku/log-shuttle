package shuttle

import (
	"log"
	"os"
)

// Default loggers to os.Stdouit and os.Stderr
var (
	Logger    = log.New(os.Stdout, "log-shuttle: ", log.LstdFlags)
	ErrLogger = log.New(os.Stderr, "log-shuttle: ", log.LstdFlags)
)
