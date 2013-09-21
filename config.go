package main

import (
	"flag"
	"log"
	"net/url"
	"os"
	"time"
)

var LogplexUrl = os.Getenv("LOGPLEX_URL")

type ShuttleConfig struct {
	FrontBuff    int
	BatchSize    int
	Wait         int
	WorkerCount  int
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
	StatsLayout  string
	StatInterval time.Duration
}

func (c *ShuttleConfig) ParseFlags() {
	flag.BoolVar(&c.PrintVersion, "version", false, "Print log-shuttle version.")
	flag.BoolVar(&c.Verbose, "verbose", false, "Enable verbose debug info.")
	flag.BoolVar(&c.SkipHeaders, "skip-headers", false, "Skip the prepending of rfc5424 headers.")
	flag.BoolVar(&c.SkipVerify, "skip-verify", false, "Skip the verification of HTTPS server certificate.")
	flag.StringVar(&c.Prival, "prival", "190", "The primary value of the rfc5424 header.")
	flag.StringVar(&c.Version, "syslog-version", "1", "The version of syslog.")
	flag.StringVar(&c.Procid, "procid", "shuttle", "The procid field for the syslog header.")
	flag.StringVar(&c.Appname, "appname", "shuttle", "The app-name field for the syslog header.")
	flag.StringVar(&c.Appname, "logplex-token", "token", "Secret logplex token.")
	flag.StringVar(&c.Hostname, "hostname", "shuttle", "The hostname field for the syslog header.")
	flag.StringVar(&c.Msgid, "msgid", "- -", "The msgid field for the syslog header.")
	flag.StringVar(&c.Socket, "socket", "", "Location of UNIX domain socket.")
	flag.StringVar(&c.LogsURL, "logs-url", "", "The receiver of the log data.")
	flag.IntVar(&c.WorkerCount, "workers", 1, "Number of concurrent outlet workers (and HTTP connections)")
	flag.IntVar(&c.Wait, "wait", 500, "Number of ms to flush messages to logplex")
	flag.IntVar(&c.BatchSize, "batch-size", 1, "Number of messages to pack into a logplex http request.")
	flag.IntVar(&c.FrontBuff, "front-buff", 0, "Number of messages to buffer in log-shuttle's input chanel.")
	flag.StringVar(&c.StatsLayout, "stat-layout", "reads=%d drops=%d", "L2met prints stats on reads and drops. This string determines the layout. Include at least 2 %ds in the layout.")
	flag.DurationVar(&c.StatInterval, "stat-interval", 10*time.Second, "L2met prints stats on reads and drops. This duration determines how often the stat is printed.")
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
	return !c.UseSocket()
}

func (c *ShuttleConfig) UseSocket() bool {
	if len(c.Socket) > 0 {
		return true
	}
	return false
}

func (c *ShuttleConfig) WaitDuration() time.Duration {
	return time.Millisecond * time.Duration(c.Wait)
}
