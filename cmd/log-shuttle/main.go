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

var detectKinesis = regexp.MustCompile(`\Akinesis.[[:alpha:]]{2}-[[:alpha:]]{2,}-[[:digit:]]\.amazonaws\.com\z`)

// Default loggers to stdout and stderr
var (
	logger    = log.New(os.Stdout, "log-shuttle: ", log.LstdFlags)
	errLogger = log.New(os.Stderr, "log-shuttle: ", log.LstdFlags)

	logToSyslog bool
)

var version = "" // log-shuttle version, set with linker

// useStdin determines if we're using the terminal's stdin or not
func useStdin() bool {
	return !util.IsTerminal(os.Stdin)
}

func mapInputFormat(i string) int {
	switch i {
	case "raw":
		return shuttle.InputFormatRaw
	case "rfc3164":
		return shuttle.InputFormatRFC3164
	case "rfc5424":
		return shuttle.InputFormatRFC5424
	default:
		log.Fatalf("Unknown input format: %s\n", i)
	}
	panic("won't get here")
}

// determineLogsURL from the various options favoring each one in turn
func determineLogsURL(logplexURL, logsURL, cmdLineURL string) string {
	var envURL string

	if len(logplexURL) > 0 {
		log.Println("Warning: $LOGPLEX_URL is deprecated, use $LOGS_URL instead")
		envURL = logplexURL
	}

	if len(logsURL) > 0 {
		if len(logplexURL) > 0 {
			log.Println("Warning: Use of both $LOGPLEX_URL & $LOGS_URL, using $LOGS_URL instead")
		}
		envURL = logsURL
	}

	if len(cmdLineURL) > 0 {
		if len(envURL) > 0 {
			log.Println("Warning: Use of both an evnironment variable ($LOGPLEX_URL or $LOGS_URL) and -logs-url, using -logs-url option")
		}
		return cmdLineURL
	}
	return envURL
}

// parseFlags overrides the properties of the given config using the provided
// command-line flags.  Any option not overridden by a flag will be untouched.
func parseFlags(c shuttle.Config) shuttle.Config {
	var skipHeaders bool

	flag.BoolVar(&c.PrintVersion, "version", c.PrintVersion, "Print log-shuttle version.")
	flag.BoolVar(&c.Verbose, "verbose", c.Verbose, "Enable verbose debug info.")
	flag.BoolVar(&c.SkipVerify, "skip-verify", c.SkipVerify, "Skip the verification of HTTPS server certificate.")
	flag.BoolVar(&c.UseGzip, "gzip", false, "POST using gzip compression")
	flag.BoolVar(&c.Drop, "drop", c.Drop, "Drop (default) logs or backup & block stdin")

	flag.BoolVar(&skipHeaders, "skip-headers", false, "Skip the prepending of rfc5424 headers.")
	flag.BoolVar(&logToSyslog, "log-to-syslog", false, "Log to syslog instead of stderr")

	var inputFormat string

	flag.StringVar(&c.Prival, "prival", c.Prival, "The primary value of the rfc5424 header.")
	flag.StringVar(&c.Version, "syslog-version", c.Version, "The version of syslog.")
	flag.StringVar(&c.Procid, "procid", c.Procid, "The procid field for the syslog header.")
	flag.StringVar(&c.Appname, "appname", c.Appname, "The app-name field for the syslog header.")
	flag.StringVar(&c.Appname, "logplex-token", c.Appname, "Secret logplex token.")
	flag.StringVar(&c.Hostname, "hostname", c.Hostname, "The hostname field for the syslog header.")
	flag.StringVar(&c.Msgid, "msgid", c.Msgid, "The msgid field for the syslog header.")
	flag.StringVar(&c.LogsURL, "logs-url", c.LogsURL, "The receiver of the log data.")
	flag.StringVar(&c.StatsSource, "stats-source", c.StatsSource, "When emitting stats, add source=<stats-source> to the stats.")

	flag.StringVar(&inputFormat, "input-format", "raw", "raw (default), rfc3164 (syslog(3)), rfc5424")

	flag.DurationVar(&c.StatsInterval, "stats-interval", c.StatsInterval, "How often to emit/reset stats.")
	flag.DurationVar(&c.WaitDuration, "wait", c.WaitDuration, "Duration to wait to flush messages to logplex")
	flag.DurationVar(&c.Timeout, "timeout", c.Timeout, "Duration to wait for a response from Logplex.")

	flag.IntVar(&c.MaxAttempts, "max-attempts", c.MaxAttempts, "Max number of retries.")
	flag.IntVar(&c.NumBatchers, "num-batchers", c.NumBatchers, "The number of batchers to run.")
	flag.IntVar(&c.NumOutlets, "num-outlets", c.NumOutlets, "The number of outlets to run.")
	flag.IntVar(&c.BatchSize, "batch-size", c.BatchSize, "Number of messages to pack into a logplex http request.")
	flag.IntVar(&c.FrontBuff, "front-buff", c.FrontBuff, "Number of messages to buffer in log-shuttle's input chanel.")
	flag.IntVar(&c.BackBuff, "back-buff", c.BackBuff, "Number of batches to buffer before dropping.")
	flag.IntVar(&c.MaxLineLength, "max-line-length", c.MaxLineLength, "Number of bytes that the backend allows per line.")

	flag.Parse()

	c.InputFormat = mapInputFormat(inputFormat)

	if skipHeaders {
		log.Println("Warning: Use of -skip-headers is deprecated, use -input-format=2 (rfc5424) instead")
		if c.InputFormat == shuttle.InputFormatRaw {
			c.InputFormat = shuttle.InputFormatRFC5424
		} else {
			log.Fatal("Cannot use -skip-headers with anything except the default input format")
		}
	}

	return c
}

func getConfig() shuttle.Config {
	c := parseFlags(shuttle.NewConfig())

	if c.MaxAttempts < 1 {
		log.Fatalf("-max-attempts must be >= 1")
	}

	c.LogsURL = determineLogsURL(os.Getenv("LOGPLEX_URL"), os.Getenv("LOGS_URL"), c.LogsURL)

	oURL, err := url.Parse(c.LogsURL)
	if err != nil {
		log.Fatalln("Error parsing -logs-url/$LOGPLEX_URL/$LOGS_URL: ", err)
	}

	if oURL.User == nil {
		oURL.User = url.UserPassword("token", c.Appname)
	}

	c.FormatterFunc = determineOutputFormatter(oURL)

	c.LogsURL = oURL.String()

	c.ComputeHeader()

	return c
}

func determineOutputFormatter(u *url.URL) shuttle.NewHTTPFormatterFunc {
	if detectKinesis.MatchString(u.Host) {
		return shuttle.NewKinesisFormatter
	}
	return shuttle.NewLogplexBatchFormatter
}

func main() {
	config := getConfig()

	var err error

	if config.PrintVersion {
		fmt.Println(version)
		os.Exit(0)
	}

	config.ID = version

	if !useStdin() {
		errLogger.Fatalln("No stdin detected.")
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
		s.Logger = logger
		s.ErrLogger = errLogger
	}

	s.Launch()

	go LogFmtMetricsEmitter(s.MetricsRegistry, config.StatsSource, config.StatsInterval, s.Logger)

	// Blocks until os.Stdin is closed
	s.ReadLogLines(os.Stdin)

	// Shutdown the shuttle.
	s.Land()
}
