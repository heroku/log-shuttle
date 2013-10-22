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
	MaxRequests  int
	NumBatchers  int
	NumOutlets   int
	Wait         int
	Batches      int
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
	flag.IntVar(&c.NumBatchers, "num-batchers", 1, "The number of batchers to run.")
	flag.IntVar(&c.NumOutlets, "num-outlets", 1, "The number of outlets to run.")
	flag.IntVar(&c.Batches, "batches", 5, "Number of pending batches to buffer.")
	flag.IntVar(&c.Wait, "wait", 250, "Number of ms to flush messages to logplex")
	flag.IntVar(&c.BatchSize, "batch-size", 500, "Number of messages to pack into a logplex http request.")
	flag.IntVar(&c.MaxRequests, "max-requests", 5, "Max number of inflight requests to logplex at any moment")
	flag.IntVar(&c.FrontBuff, "front-buff", 0, "Number of messages to buffer in log-shuttle's input chanel.")
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
