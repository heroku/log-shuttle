package shuttle

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"time"

	"github.com/pebbe/util"
)

const (
	// Version is the current version of the program / library
	Version = "0.9.6"
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
	DefaultLogToSyslog   = false
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
	SkipHeaders                         bool
	SkipVerify                          bool
	PrintVersion                        bool
	Verbose                             bool
	LogToSyslog                         bool
	WaitDuration                        time.Duration
	Timeout                             time.Duration
	lengthPrefixedSyslogFrameHeaderSize int
	syslogFrameHeaderFormat             string
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
		MaxAttempts:   DefaultMaxAttempts,
		InputFormat:   DefaultInputFormat,
		NumBatchers:   DefaultNumBatchers,
		NumOutlets:    DefaultNumOutlets,
		WaitDuration:  time.Duration(DefaultWaitDuration),
		BatchSize:     DefaultBatchSize,
		FrontBuff:     DefaultFrontBuff,
		BackBuff:      DefaultBackBuff,
		Timeout:       time.Duration(DefaultTimeout),
		LogToSyslog:   DefaultLogToSyslog,
	}

	shuttleConfig.ComputeHeader()

	return shuttleConfig
}

// OutletURL returns the string representation of the log url including basic
// auth encoded into the url.
func (c *Config) OutletURL() string {
	var err error
	var oURL *url.URL

	if len(c.LogsURL) > 0 {
		oURL, err = url.Parse(c.LogsURL)
		if err != nil {
			log.Fatalf("Unable to parse logs-url")
		}
	}
	if oURL.User == nil {
		oURL.User = url.UserPassword("token", c.Appname)
	}
	return oURL.String()
}

// UseStdin determines if we're using the terminal's stdin or not
func (c *Config) UseStdin() bool {
	return !util.IsTerminal(os.Stdin)
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
