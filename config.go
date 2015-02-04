package shuttle

import (
	"fmt"
	"log"
	"time"
)

// Input format constants.
// TODO: ensure these are really used properly
const (
	InputFormatRaw = iota
	InputFormatRFC3164
)

// Default option values
const (
	DefaultMaxLineLength = 10000 // Logplex max is 10000 bytes, so default to that
	DefaultInputFormat   = InputFormatRaw
	DefaultFrontBuff     = 1000
	DefaultBackBuff      = 50
	DefaultTimeout       = 5 * time.Second
	DefaultWaitDuration  = 250 * time.Millisecond
	DefaultMaxAttempts   = 3
	DefaultStatsInterval = 0 * time.Second
	DefaultStatsSource   = ""
	DefaultPrintVersion  = false
	DefaultVerbose       = false
	DefaultSkipHeaders   = false
	DefaultSkipVerify    = false
	DefaultPriVal        = "190"
	DefaultVersion       = "1"
	DefaultProcID        = "shuttle"
	DefaultAppName       = "token"
	DefaultHostname      = "shuttle"
	DefaultMsgID         = "- -"
	DefaultLogsURL       = ""
	DefaultNumBatchers   = 2
	DefaultNumOutlets    = 4
	DefaultBatchSize     = 500
	DefaultID            = ""
	DefaultSocket        = ""
)

const (
	errDrop errType = iota
	errLost
)

type errType int

type errData struct {
	count int
	since time.Time
	eType errType
}

// Config holds the various config options for a shuttle
type Config struct {
	MaxLineLength                       int
	BackBuff                            int
	FrontBuff                           int
	BatchSize                           int
	NumBatchers                         int
	NumOutlets                          int
	InputFormat                         int
	MaxAttempts                         int
	LogsURL                             string
	Prival                              string
	Version                             string
	Procid                              string
	Hostname                            string
	Appname                             string
	Msgid                               string
	StatsSource                         string
	SkipHeaders                         bool
	SkipVerify                          bool
	PrintVersion                        bool
	Verbose                             bool
	WaitDuration                        time.Duration
	Timeout                             time.Duration
	StatsInterval                       time.Duration
	lengthPrefixedSyslogFrameHeaderSize int
	syslogFrameHeaderFormat             string
	ID                                  string
	Socket                              string
	// Loggers
	Logger    *log.Logger
	ErrLogger *log.Logger
}

// NewConfig returns a newly created Config, filled in with defaults
func NewConfig() Config {
	shuttleConfig := Config{
		MaxLineLength: DefaultMaxLineLength,
		PrintVersion:  DefaultPrintVersion,
		Verbose:       DefaultVerbose,
		SkipHeaders:   DefaultSkipHeaders,
		SkipVerify:    DefaultSkipVerify,
		Prival:        DefaultPriVal,
		Version:       DefaultVersion,
		Procid:        DefaultProcID,
		Appname:       DefaultAppName,
		Hostname:      DefaultHostname,
		Msgid:         DefaultMsgID,
		LogsURL:       DefaultLogsURL,
		StatsSource:   DefaultStatsSource,
		StatsInterval: time.Duration(DefaultStatsInterval),
		MaxAttempts:   DefaultMaxAttempts,
		InputFormat:   DefaultInputFormat,
		NumBatchers:   DefaultNumBatchers,
		NumOutlets:    DefaultNumOutlets,
		WaitDuration:  time.Duration(DefaultWaitDuration),
		BatchSize:     DefaultBatchSize,
		FrontBuff:     DefaultFrontBuff,
		BackBuff:      DefaultBackBuff,
		Timeout:       time.Duration(DefaultTimeout),
		Socket:        DefaultSocket,
		ID:            DefaultID,
		Logger:        discardLogger,
		ErrLogger:     discardLogger,
	}

	shuttleConfig.ComputeHeader()

	return shuttleConfig
}

// ComputeHeader computes the syslogFrameHeaderFormat once so we don't have to
// do that for every formatter itteration
func (c *Config) ComputeHeader() {
	// This is here to pre-compute this so other's don't have to later
	c.lengthPrefixedSyslogFrameHeaderSize = len(c.Prival) + len(c.Version) + len(LogplexBatchTimeFormat) +
		len(c.Hostname) + len(c.Appname) + len(c.Procid) + len(c.Msgid) + 8 // spaces, < & >

	c.syslogFrameHeaderFormat = fmt.Sprintf("%s <%s>%s %s %s %s %s %s ",
		"%d",
		c.Prival,
		c.Version,
		"%s", // The time should be put here
		c.Hostname,
		c.Appname,
		c.Procid,
		c.Msgid)
}
