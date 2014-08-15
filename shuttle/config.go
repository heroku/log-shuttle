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

var LogplexUrl = os.Getenv("LOGPLEX_URL")

const (
	INPUT_FORMAT_RAW     = iota
	INPUT_FORMAT_RFC3164 = iota
)

const (
	DEFAULT_INPUT_FORMAT   = INPUT_FORMAT_RAW
	DEFAULT_FRONT_BUFF     = 1000
	DEFAULT_BACK_BUFF      = 50
	DEFAULT_STATS_BUFF     = 5000
	DEFAULT_STATS_ADDR     = ""
	DEFAULT_TIMEOUT        = 5 * time.Second
	DEFAULT_WAIT_DURATION  = 250 * time.Millisecond
	DEFAULT_MAX_ATTEMPTS   = 3
	DEFAULT_STATS_INTERVAL = 0 * time.Second
	DEFAULT_STATS_SOURCE   = ""
	DEFAULT_PRINT_VERSION = false
	DEFAULT_VERBOSE = false
	DEFAULT_SKIP_HEADERS = false
	DEFAULT_SKIP_VERIFY = false
	DEFAULT_PRIVAL = "190"
	DEFAULT_VERSION = "1"
	DEFAULT_PROCID = "shuttle"
	DEFAULT_APPNAME = "token"
	DEFAULT_HOSTNAME = "shuttle"
	DEFAULT_MSGID = "- -"
	DEFAULT_LOGS_URL = ""
	DEFAULT_NUM_BATCHERS = 2
	DEFAULT_NUM_OUTLETS = 4
	DEFAULT_BATCH_SIZE = 500
	DEFAULT_LOG_TO_SYSLOG = false

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

type ShuttleConfig struct {
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

// Create a new config using the defaults.
func NewConfig () ShuttleConfig {
	var shuttleConfig ShuttleConfig
	shuttleConfig.PrintVersion = DEFAULT_PRINT_VERSION
	shuttleConfig.Verbose = DEFAULT_VERBOSE
	shuttleConfig.SkipHeaders = DEFAULT_SKIP_HEADERS
	shuttleConfig.SkipVerify = DEFAULT_SKIP_VERIFY
	shuttleConfig.Prival = DEFAULT_PRIVAL
	shuttleConfig.Version = DEFAULT_VERSION
	shuttleConfig.Procid = DEFAULT_PROCID
	shuttleConfig.Appname = DEFAULT_APPNAME
	shuttleConfig.Hostname = DEFAULT_HOSTNAME
	shuttleConfig.Msgid = DEFAULT_MSGID
	shuttleConfig.LogsURL = DEFAULT_LOGS_URL
	shuttleConfig.StatsAddr = DEFAULT_STATS_ADDR
	shuttleConfig.StatsSource = DEFAULT_STATS_SOURCE
	shuttleConfig.StatsInterval = time.Duration(DEFAULT_STATS_INTERVAL)
	shuttleConfig.MaxAttempts = DEFAULT_MAX_ATTEMPTS
	shuttleConfig.InputFormat = DEFAULT_INPUT_FORMAT
	shuttleConfig.NumBatchers = DEFAULT_NUM_BATCHERS
	shuttleConfig.NumOutlets = DEFAULT_NUM_OUTLETS
	shuttleConfig.WaitDuration = time.Duration(DEFAULT_WAIT_DURATION)
	shuttleConfig.BatchSize = DEFAULT_BATCH_SIZE
	shuttleConfig.FrontBuff = DEFAULT_FRONT_BUFF
	shuttleConfig.BackBuff = DEFAULT_BACK_BUFF
	shuttleConfig.StatsBuff = DEFAULT_STATS_BUFF
	shuttleConfig.Timeout = time.Duration(DEFAULT_TIMEOUT)
	shuttleConfig.LogToSyslog = DEFAULT_LOG_TO_SYSLOG

	shuttleConfig.ComputeHeader()

	return shuttleConfig
}

// Overrides the properties of the given config using the provided command-line flags.
// Any option not overridden by a flag will be untouched.
func (c *ShuttleConfig) ParseFlags() {
	flag.BoolVar(&c.PrintVersion, "version", c.PrintVersion, "Print log-shuttle version.")
	flag.BoolVar(&c.Verbose, "verbose", c.Verbose, "Enable verbose debug info.")
	flag.BoolVar(&c.SkipHeaders, "skip-headers", c.SkipHeaders, "Skip the prepending of rfc5424 headers.")
	flag.BoolVar(&c.SkipVerify, "skip-verify", c.SkipVerify, "Skip the verification of HTTPS server certificate.")
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
	flag.IntVar(&c.MaxAttempts, "max-attempts", c.MaxAttempts, "Max number of retries.")
	flag.IntVar(&c.InputFormat, "input-format", c.InputFormat, "0=raw (default), 1=rfc3164 (syslog(3))")
	flag.IntVar(&c.NumBatchers, "num-batchers", c.NumBatchers, "The number of batchers to run.")
	flag.IntVar(&c.NumOutlets, "num-outlets", c.NumOutlets, "The number of outlets to run.")
	flag.DurationVar(&c.WaitDuration, "wait", c.WaitDuration, "Duration to wait to flush messages to logplex")
	flag.IntVar(&c.BatchSize, "batch-size", c.BatchSize, "Number of messages to pack into a logplex http request.")
	flag.IntVar(&c.FrontBuff, "front-buff", c.FrontBuff, "Number of messages to buffer in log-shuttle's input chanel.")
	flag.IntVar(&c.BackBuff, "back-buff", c.BackBuff, "Number of batches to buffer before dropping.")
	flag.IntVar(&c.StatsBuff, "stats-buff", c.StatsBuff, "Number of stats to buffer.")
	flag.DurationVar(&c.Timeout, "timeout", c.Timeout, "Duration to wait for a response from Logplex.")
	flag.BoolVar(&c.LogToSyslog, "log-to-syslog", c.LogToSyslog, "Log to syslog instead of stderr")
	flag.Parse()

	if c.MaxAttempts < 1 {
		log.Fatalf("-max-attempts must be >= 1")
	}

	c.ComputeHeader()
}

func (c *ShuttleConfig) OutletURL() string {
	var err error
	var oUrl *url.URL

	if len(c.LogsURL) > 0 {
		oUrl, err = url.Parse(c.LogsURL)
		if err != nil {
			log.Fatalf("Unable to parse logs-url")
		}
	}
	if len(LogplexUrl) > 0 {
		oUrl, err = url.Parse(LogplexUrl)
		if err != nil {
			log.Fatalf("Unable to parse $LOGPLEX_URL")
		}
	}

	if oUrl == nil {
		log.Fatalf("Must set -logs-url or $LOGPLEX_URL.")
	}

	if oUrl.User == nil {
		oUrl.User = url.UserPassword("token", c.Appname)
	}
	return oUrl.String()
}

func (c *ShuttleConfig) UseStdin() bool {
	return !util.IsTerminal(os.Stdin)
}

func (c *ShuttleConfig) ComputeHeader() {
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
