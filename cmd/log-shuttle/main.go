package main

import (
	"flag"
	"fmt"
	"log"
	"log/syslog"
	"net/url"
	"os"
	"regexp"

	"github.com/heroku/log-shuttle"
	"github.com/heroku/log-shuttle/Godeps/_workspace/src/github.com/pebbe/util"
)

// LogplexURL is the url of the logplex cluster (or work alike) to connect
// to, defaults to the $LOGPLEX_URL environment variable
var LogplexURL = os.Getenv("LOGPLEX_URL")

var detectKinesis = regexp.MustCompile(`\Akinesis.[[:alpha:]]{2}-[[:alpha:]]{2,}-[[:digit:]]\.amazonaws\.com\z`)

// Default loggers to stdout and stderr
var (
	Logger    = log.New(os.Stdout, "log-shuttle: ", log.LstdFlags)
	ErrLogger = log.New(os.Stderr, "log-shuttle: ", log.LstdFlags)

	logToSyslog bool
)

var version = "" // log-shuttle version, set with linker

// UseStdin determines if we're using the terminal's stdin or not
func UseStdin() bool {
	return !util.IsTerminal(os.Stdin)
}

// ParseFlags overrides the properties of the given config using the provided
// command-line flags.  Any option not overridden by a flag will be untouched.
func ParseFlags(c shuttle.Config) shuttle.Config {
	flag.BoolVar(&c.PrintVersion, "version", c.PrintVersion, "Print log-shuttle version.")
	flag.BoolVar(&c.Verbose, "verbose", c.Verbose, "Enable verbose debug info.")
	flag.BoolVar(&c.SkipHeaders, "skip-headers", c.SkipHeaders, "Skip the prepending of rfc5424 headers.")
	flag.BoolVar(&c.SkipVerify, "skip-verify", c.SkipVerify, "Skip the verification of HTTPS server certificate.")
	flag.BoolVar(&logToSyslog, "log-to-syslog", false, "Log to syslog instead of stderr")
	flag.BoolVar(&c.UseGzip, "gzip", false, "POST using gzip compression")
	flag.BoolVar(&c.Drop, "drop", c.Drop, "Drop (default) logs or backup & block stdin")

	flag.StringVar(&c.Prival, "prival", c.Prival, "The primary value of the rfc5424 header.")
	flag.StringVar(&c.Version, "syslog-version", c.Version, "The version of syslog.")
	flag.StringVar(&c.Procid, "procid", c.Procid, "The procid field for the syslog header.")
	flag.StringVar(&c.Appname, "appname", c.Appname, "The app-name field for the syslog header.")
	flag.StringVar(&c.Appname, "logplex-token", c.Appname, "Secret logplex token.")
	flag.StringVar(&c.Hostname, "hostname", c.Hostname, "The hostname field for the syslog header.")
	flag.StringVar(&c.Msgid, "msgid", c.Msgid, "The msgid field for the syslog header.")
	flag.StringVar(&c.LogsURL, "logs-url", c.LogsURL, "The receiver of the log data.")
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
	flag.IntVar(&c.MaxLineLength, "max-line-length", c.MaxLineLength, "Number of bytes that the backend allows per line.")

	flag.Parse()

	if c.MaxAttempts < 1 {
		log.Fatalf("-max-attempts must be >= 1")
	}

	if len(LogplexURL) > 0 {
		if c.LogsURL != shuttle.DefaultLogsURL {
			log.Println("Warning: Use of both $LOGPLEX_URL and -logs-url, defaulting to -logs-url setting.")
		} else {
			c.LogsURL = LogplexURL
		}
	}

	oURL, err := url.Parse(c.LogsURL)
	if err != nil {
		log.Fatalln("Error parsing -logs-url or $LOGPLEX_URL: ", err)
	}

	if oURL.User == nil {
		oURL.User = url.UserPassword("token", c.Appname)
	}

	if detectKinesis.MatchString(oURL.Host) {
		c.FormatterFunc = shuttle.NewKinesisFormatter
	} else {
		c.FormatterFunc = shuttle.NewLogplexBatchFormatter
	}

	c.LogsURL = oURL.String()

	c.ComputeHeader()

	return c
}

func main() {
	config := shuttle.NewConfig()
	config = ParseFlags(config)

	var err error

	if config.PrintVersion {
		fmt.Println(version)
		os.Exit(0)
	}

	config.ID = version

	if !UseStdin() {
		ErrLogger.Fatalln("No stdin detected.")
	}

	s := shuttle.NewShuttle(config)

	// Setup the loggers before doing anything else
	if logToSyslog {
		s.Logger, err = syslog.NewLogger(syslog.LOG_INFO|syslog.LOG_SYSLOG, 0)
		if err != nil {
			log.Fatalf("Unable to setup syslog logger: %s\n", err)
		}
		s.ErrLogger, err = syslog.NewLogger(syslog.LOG_ERR|syslog.LOG_SYSLOG, 0)
		if err != nil {
			log.Fatalf("Unable to setup syslog error logger: %s\n", err)
		}
	} else {
		s.Logger = Logger
		s.ErrLogger = ErrLogger
	}

	s.Launch()

	go LogFmtMetricsEmitter(s.MetricsRegistry, config.StatsSource, config.StatsInterval, s.Logger)

	// Blocks until os.Stdin is closed
	s.ReadLogLines(os.Stdin)

	// Shutdown the shuttle.
	s.Land()
}
