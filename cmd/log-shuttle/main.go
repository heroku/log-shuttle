package main

import (
	"flag"
	"fmt"
	"log"
	"log/syslog"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/heroku/log-shuttle"
	"github.com/pebbe/util"
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

func mapInputFormat(i string) (int, error) {
	switch i {
	case "raw":
		return shuttle.InputFormatRaw, nil
	case "rfc5424":
		return shuttle.InputFormatRFC5424, nil
	case "lprfc5424":
		return shuttle.InputFormatLengthPrefixedRFC5424, nil
	}
	return 0, fmt.Errorf("Unknown input format: %s", i)
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
func parseFlags(c shuttle.Config) (shuttle.Config, error) {
	var skipHeaders bool
	var statsAddr string
	var printVersion bool

	flag.BoolVar(&c.Verbose, "verbose", c.Verbose, "Enable verbose debug info.")
	flag.BoolVar(&c.SkipVerify, "skip-verify", c.SkipVerify, "Skip the verification of HTTPS server certificate.")
	flag.BoolVar(&c.UseGzip, "gzip", c.UseGzip, "POST using gzip compression.")
	flag.BoolVar(&c.Drop, "drop", c.Drop, "Drop (default) logs or backup & block stdin.")

	flag.BoolVar(&skipHeaders, "skip-headers", skipHeaders, "Skip the prepending of rfc5424 headers.")
	flag.BoolVar(&logToSyslog, "log-to-syslog", logToSyslog, "Log to syslog instead of stderr.")
	flag.BoolVar(&printVersion, "version", printVersion, "Print log-shuttle version & exit.")

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

	flag.StringVar(&inputFormat, "input-format", "raw", "'raw' (default; newline termined text), 'rfc5424' (newline terminated rfc5424), 'lprfc5424' (length prefixed rfc5424).")
	flag.StringVar(&statsAddr, "stats-addr", "", "DEPRECATED, WILL BE REMOVED, HAS NO EFFECT.")

	flag.DurationVar(&c.StatsInterval, "stats-interval", c.StatsInterval, "How often to emit/reset stats.")
	flag.DurationVar(&c.WaitDuration, "wait", c.WaitDuration, "Duration to wait to flush messages to logs-url.")
	flag.DurationVar(&c.Timeout, "timeout", c.Timeout, "Duration to wait for a response from logs-url.")

	flag.IntVar(&c.MaxAttempts, "max-attempts", c.MaxAttempts, "Max number of retries.")
	var b int
	flag.IntVar(&b, "num-batchers", b, "[NO EFFECT/REMOVED] The number of batchers to run.")
	flag.IntVar(&c.NumOutlets, "num-outlets", c.NumOutlets, "The number of outlets to run.")
	flag.IntVar(&c.BatchSize, "batch-size", c.BatchSize, "Number of messages to pack into an application/logplex-1 http request.")
	var f int
	flag.IntVar(&f, "front-buff", f, "[NO EFFECT/REMOVED] Number of messages to buffer in log-shuttle's input channel.")
	flag.IntVar(&c.BackBuff, "back-buff", c.BackBuff, "Number of batches to buffer before dropping.")
	flag.IntVar(&c.MaxLineLength, "max-line-length", c.MaxLineLength, "Number of bytes that the backend allows per line.")
	flag.IntVar(&c.KinesisShards, "kinesis-shards", c.KinesisShards, "Number of unique partition keys to use per app.")

	flag.Parse()

	if printVersion {
		fmt.Println(version)
		os.Exit(0)
	}

	if f != 0 {
		log.Println("Warning: Use of -front-buff is no longer supported. The flag has no effect and will be removed in the future.")
	}

	if b != 0 {
		log.Println("Warning: Use of -num-batchers is no longer supported. The flag has no effect and will be removed in the future.")
	}

	if statsAddr != "" {
		log.Println("Warning: Use of -stats-addr is deprecated and will be dropped in the future.")
	}

	var err error
	c.InputFormat, err = mapInputFormat(inputFormat)
	if err != nil {
		return c, err
	}

	if skipHeaders {
		log.Println("Warning: Use of -skip-headers is deprecated, use -input-format=rfc5424 instead")
		switch c.InputFormat {
		case shuttle.InputFormatRaw:
			// Massage InputFormat as that's what is used internally
			c.InputFormat = shuttle.InputFormatRFC5424
		case shuttle.InputFormatRFC5424:
			// NOOP
		default:
			return c, fmt.Errorf("Can only use -skip-headers with default input format or rfc5424")
		}
	}

	return c, nil
}

// validateURL validates the url provided as a string.
func validateURL(u string) (*url.URL, error) {
	oURL, err := url.Parse(u)
	if err != nil {
		return nil, fmt.Errorf("Error parsing logs-url/$LOGPLEX_URL/$LOGS_URL: %s", err.Error())
	}

	switch oURL.Scheme {
	case "http", "https":
		// no-op these are good
	default:
		return nil, fmt.Errorf("Invalid URL scheme in provided logs-url: %s", u)
	}

	if oURL.Host == "" {
		return nil, fmt.Errorf("No host specified in provided logs-url: %s", u)
	}

	parts := strings.Split(oURL.Host, ":")

	if len(parts) > 2 {
		return nil, fmt.Errorf("Invalid host specified in provided logs-url: %s", u)
	}

	if len(parts) == 2 {
		_, err := strconv.Atoi(parts[1])
		if err != nil {
			return nil, fmt.Errorf("Invalid port specified in provided logs-url: %s", u)
		}
	}

	return oURL, nil
}

func getConfig() (shuttle.Config, error) {
	c, err := parseFlags(shuttle.NewConfig())
	if err != nil {
		return c, err
	}

	if c.MaxAttempts < 1 {
		return c, fmt.Errorf("-max-attempts must be >= 1")
	}

	c.LogsURL = determineLogsURL(os.Getenv("LOGPLEX_URL"), os.Getenv("LOGS_URL"), c.LogsURL)
	oURL, err := validateURL(c.LogsURL)
	if err != nil {
		return c, err
	}

	if oURL.User == nil {
		oURL.User = url.UserPassword("token", c.Appname)
	}

	c.FormatterFunc = determineOutputFormatter(oURL)

	c.LogsURL = oURL.String()

	c.ComputeHeader()

	return c, nil
}

func determineOutputFormatter(u *url.URL) shuttle.NewHTTPFormatterFunc {
	if detectKinesis.MatchString(u.Host) {
		return shuttle.NewKinesisFormatter
	}
	return shuttle.NewLogplexBatchFormatter
}

func main() {
	config, err := getConfig()
	if err != nil {
		errLogger.Fatalf("error=%q\n", err)
	}

	config.ID = version

	if !useStdin() {
		errLogger.Fatalln(`error="No stdin detected."`)
	}

	s := shuttle.NewShuttle(config)

	// Setup the loggers before doing anything else
	if logToSyslog {
		s.Logger, err = syslog.NewLogger(syslog.LOG_INFO|syslog.LOG_SYSLOG, 0)
		if err != nil {
			errLogger.Fatalf(`error="Unable to setup syslog logger: %s\n"`, err)
		}
		s.ErrLogger, err = syslog.NewLogger(syslog.LOG_ERR|syslog.LOG_SYSLOG, 0)
		if err != nil {
			errLogger.Fatalf(`error="Unable to setup syslog error logger: %s\n"`, err)
		}
	} else {
		s.Logger = logger
		s.ErrLogger = errLogger
	}

	s.LoadReader(os.Stdin)

	s.Launch()
	metricsReporter := shuttle.NewMetricsReporter(s.MetricsRegistry, config.StatsSource, config.StatsInterval, s.Logger)
	go metricsReporter.Emit()

	// blocks until the readers all exit
	s.WaitForReadersToFinish()

	// Shutdown the shuttle.
	s.Land()
	metricsReporter.Stop()
}
