package shuttle

import (
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"time"

	"github.com/pebbe/util"
)

// This is the Logplex url to connect to, default to the $LOGPLEX_URL environment variable
var LogplexURL = os.Getenv("LOGPLEX_URL")

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
	DefaultStatsBuff     = 5000
	DefaultStatsAddr     = ""
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
	StatsBuff                           int
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
	StatsAddr                           string
	StatsSource                         string
	SkipHeaders                         bool
	SkipVerify                          bool
	PrintVersion                        bool
	Verbose                             bool
	LogToSyslog                         bool
	WaitDuration                        time.Duration
	Timeout                             time.Duration
	StatsInterval                       time.Duration
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
		StatsAddr:     DefaultStatsAddr,
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
		StatsBuff:     DefaultStatsBuff,
		Timeout:       time.Duration(DefaultTimeout),
		LogToSyslog:   DefaultLogToSyslog,
	}

	shuttleConfig.ComputeHeader()

	return shuttleConfig
}

// ParseFlags overrides the properties of the given config using the provided
// command-line flags.  Any option not overridden by a flag will be untouched.
func (c *Config) ParseFlags() {
	flag.BoolVar(&c.PrintVersion, "version", c.PrintVersion, "Print log-shuttle version.")
	flag.BoolVar(&c.Verbose, "verbose", c.Verbose, "Enable verbose debug info.")
	flag.BoolVar(&c.SkipHeaders, "skip-headers", c.SkipHeaders, "Skip the prepending of rfc5424 headers.")
	flag.BoolVar(&c.SkipVerify, "skip-verify", c.SkipVerify, "Skip the verification of HTTPS server certificate.")
	flag.BoolVar(&c.LogToSyslog, "log-to-syslog", c.LogToSyslog, "Log to syslog instead of stderr")

	flag.StringVar(&c.Prival, "prival", c.Prival, "The primary value of the rfc5424 header.")
	flag.StringVar(&c.Version, "syslog-version", c.Version, "The version of syslog.")
	flag.StringVar(&c.Procid, "procid", c.Procid, "The procid field for the syslog header.")
	flag.StringVar(&c.Appname, "appname", c.Appname, "The app-name field for the syslog header.")
	flag.StringVar(&c.Appname, "logplex-token", c.Appname, "Secret logplex token.")
	flag.StringVar(&c.Hostname, "hostname", c.Hostname, "The hostname field for the syslog header.")
	flag.StringVar(&c.Msgid, "msgid", c.Msgid, "The msgid field for the syslog header.")
	flag.StringVar(&c.LogsURL, "logs-url", c.LogsURL, "The receiver of the log data.")
	flag.StringVar(&c.StatsAddr, "stats-addr", c.StatsAddr, "Where to expose stats.")
	flag.StringVar(&c.StatsSource, "stats-source", c.StatsSource, "When emitting stats, add source=<stats-source> to the stats.")

	flag.DurationVar(&c.StatsInterval, "stats-interval", c.StatsInterval, "How often to emit/reset stats.")
	flag.DurationVar(&c.WaitDuration, "wait", c.WaitDuration, "Duration to wait to flush messages to logplex")
	flag.DurationVar(&c.Timeout, "timeout", c.Timeout, "Duration to wait for a response from Logplex.")

	flag.IntVar(&c.MaxAttempts, "max-attempts", c.MaxAttempts, "Max number of retries.")
	flag.IntVar(&c.InputFormat, "input-format", c.InputFormat, "0=raw (default), 1=rfc3164 (syslog(3))")
	flag.IntVar(&c.NumBatchers, "num-batchers", c.NumBatchers, "The number of batchers to run.")
	flag.IntVar(&c.NumOutlets, "num-outlets", c.NumOutlets, "The number of outlets to run.")
	flag.IntVar(&c.BatchSize, "batch-size", c.BatchSize, "Number of messages to pack into a logplex http request.")
	flag.IntVar(&c.FrontBuff, "front-buff", c.FrontBuff, "Number of messages to buffer in log-shuttle's input chanel.")
	flag.IntVar(&c.BackBuff, "back-buff", c.BackBuff, "Number of batches to buffer before dropping.")
	flag.IntVar(&c.StatsBuff, "stats-buff", c.StatsBuff, "Number of stats to buffer.")
	flag.IntVar(&c.MaxLineLength, "max-line-length", c.MaxLineLength, "Number of bytes that the backend allows per line.")

	flag.Parse()

	if c.MaxAttempts < 1 {
		log.Fatalf("-max-attempts must be >= 1")
	}

	c.ComputeHeader()
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
	if len(LogplexURL) > 0 {
		oURL, err = url.Parse(LogplexURL)
		if err != nil {
			log.Fatalf("Unable to parse $LOGPLEX_URL")
		}
	}

	if oURL == nil {
		log.Fatalf("Must set -logs-url or $LOGPLEX_URL.")
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
	c.lengthPrefixedSyslogFrameHeaderSize = len(c.Prival) + len(c.Version) + len(LOGPLEX_BATCH_TIME_FORMAT) +
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
