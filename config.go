package main

import (
	"flag"
	"github.com/pebbe/util"
	"log"
	"net/url"
	"os"
	"time"
)

var LogplexUrl = os.Getenv("LOGPLEX_URL")

const (
	DEFAULT_FRONT_BUFF = 5000
)

type ShuttleConfig struct {
	FrontBuff    int
	BatchSize    int
	NumBatchers  int
	NumOutlets   int
	Socket       string
	LogsURL      string
	Prival       string
	Version      string
	Procid       string
	Hostname     string
	Appname      string
	Msgid        string
	SkipHeaders  bool
	SkipVerify   bool
	PrintVersion bool
	Verbose      bool
	WaitDuration time.Duration
	Timeout      time.Duration
	ReportEvery  time.Duration
}

func (c *ShuttleConfig) ParseFlags() {
	flag.BoolVar(&c.PrintVersion, "version", false, "Print log-shuttle version.")
	flag.BoolVar(&c.Verbose, "verbose", false, "Enable verbose debug info.")
	flag.BoolVar(&c.SkipHeaders, "skip-headers", false, "Skip the prepending of rfc5424 headers.")
	flag.BoolVar(&c.SkipVerify, "skip-verify", false, "Skip the verification of HTTPS server certificate.")
	flag.StringVar(&c.Prival, "prival", "190", "The primary value of the rfc5424 header.")
	flag.StringVar(&c.Version, "syslog-version", "1", "The version of syslog.")
	flag.StringVar(&c.Procid, "procid", "shuttle", "The procid field for the syslog header.")
	flag.StringVar(&c.Appname, "appname", "", "The app-name field for the syslog header.")
	flag.StringVar(&c.Appname, "logplex-token", "", "Secret logplex token.")
	flag.StringVar(&c.Hostname, "hostname", "shuttle", "The hostname field for the syslog header.")
	flag.StringVar(&c.Msgid, "msgid", "- -", "The msgid field for the syslog header.")
	flag.StringVar(&c.Socket, "socket", "", "Location of UNIX domain socket.")
	flag.StringVar(&c.LogsURL, "logs-url", "", "The receiver of the log data.")
	flag.IntVar(&c.NumBatchers, "num-batchers", 2, "The number of batchers to run.")
	flag.IntVar(&c.NumOutlets, "num-outlets", 4, "The number of outlets to run.")
	flag.DurationVar(&c.WaitDuration, "wait", time.Duration(250*time.Millisecond), "Duration to wait to flush messages to logplex")
	flag.IntVar(&c.BatchSize, "batch-size", 500, "Number of messages to pack into a logplex http request.")
	flag.IntVar(&c.FrontBuff, "front-buff", DEFAULT_FRONT_BUFF, "Number of messages to buffer in log-shuttle's input chanel.")
	flag.DurationVar(&c.Timeout, "timeout", time.Duration(2*time.Second), "Duration to wait for a response from Logplex.")
	flag.DurationVar(&c.ReportEvery, "report-every", time.Duration(5*time.Second), "How often to report stat info.")
	flag.Parse()
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

func (c *ShuttleConfig) UseSocket() bool {
	if len(c.Socket) > 0 {
		return true
	}
	return false
}
