package main

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

var (
	reqInFlight sync.WaitGroup
)

// Flags
var (
	frontBuff            = *flag.Int("front-buff", 0, "Number of messages to buffer in log-shuttle's input chanel.")
	batchSize            = *flag.Int("batch-size", 50, "Number of messages to pack into a logplex http request.")
	wait                 = *flag.Int("wait", 500, "Number of ms to flush messages to logplex")
	workerCount          = *flag.Int("workers", 1, "Number of concurrent outlet workers (and HTTP connections)")
	socket               = *flag.String("socket", "", "Location of UNIX domain socket.")
	logplexToken         = *flag.String("logplex-token", "abc123", "Secret logplex token.")
	procid               = *flag.String("procid", "", "The procid for the syslog payload")
	skipHeaders          = *flag.Bool("skip-headers", false, "Skip the prepending of rfc5424 headers.")
	skipCertVerification = *flag.Bool("skip-cert-verification", false, "Disable SSL cert validation.")
)

func init() {
	flag.Parse()
		
  if workerCount < 1 {
    workerCount = 1 // workerCount needs to be >= 1
  }
}

//Env
var (
	logplexUrl       *url.URL
	logplexUrlString = os.Getenv("LOGPLEX_URL")
)

func init() {
	var err error
	logplexUrl, err = url.Parse(logplexUrlString)
	if err != nil {
		log.Fatal("Can't parse LOGPLEX_URL: ", err)
	}
	// If the username and password weren't part of the URL, use the
	// logplex-token as the password
	if logplexUrl.User == nil {
		logplexUrl.User = url.UserPassword("token", logplexToken)
	}
	if logplexUrl.Scheme == "https" {
		tr := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: skipCertVerification}}
		http.DefaultTransport = tr
	}
}

func prepare(w io.Writer, batch []string, logplexToken, procid string, skipHeaders bool) {
	for _, msg := range batch {
		if !skipHeaders {
			//http://tools.ietf.org/html/rfc5424
			//<PRIVAL>VERSION TIME HOST APPNAME
			t := time.Now().UTC().Format("2006-01-02T15:04:05+00:00 ")
			msg = "<0>1 " + t + "1234 " + logplexToken + " " + procid + " - - " + msg
		}
		fmt.Fprintf(w, "%d %s", len(msg), msg)
	}
}

// Outlet takes batches of log lines and submits them to logplex via HTTP.
// Additionaly it can wrap each log line with a syslog header.
func outlet(batches <-chan []string, logplexToken, url, procid string, skipHeaders bool) {
	var b bytes.Buffer
	for batch := range batches {
		reqInFlight.Add(1)
		defer reqInFlight.Done()
		postLogs(b, batch, logplexToken, url, procid, skipHeaders)
	}
}

func postLogs(b bytes.Buffer, batch []string, logplexToken, url, procid string, skipHeaders bool) {
	prepare(&b, batch, logplexToken, procid, skipHeaders)
	req, _ := http.NewRequest("POST", url, &b)
	req.Header.Add("Content-Type", "application/logplex-1")
	req.Header.Add("Logplex-Msg-Count", strconv.Itoa(len(batch)))
	resp, err := http.DefaultClient.Do(req)
	b.Reset()
	if err != nil {
		fmt.Printf("error=%v\n", err)
	} else {
		fmt.Printf("at=logplex-post status=%v\n", resp.StatusCode)
		resp.Body.Close()
	}
}

// Handle facilitates the handoff between stdin/sockets & logplex http
// requests. If there is high volume traffic on the lines channel, we
// create batches based on the batchSize flag. For low volume traffic,
// we create batches based on a time interval.
func handle(lines <-chan string, batches chan<- []string, batchSize, wait int) {
	ticker := time.Tick(time.Millisecond * time.Duration(wait))
	batch := make([]string, 0, batchSize)
	for {
		select {
		case <-ticker:
			if len(batch) > 0 {
				batches <- batch
				batch = make([]string, 0, batchSize)
			}
		case l := <-lines:
			batch = append(batch, l)
			if len(batch) == cap(batch) {
				batches <- batch
				batch = make([]string, 0, batchSize)
			}
		}
	}
}

// Read will drop messages if the channel is buffered and the buffer is full.
// This is an alternitive to putting back pressure on the inputer of log-shuttle.
// If you want 0 chance of dropped messages, use an unbufferd channel and
// prepare the the process who is inputing data into log-shuttle to wait on
// log-shuttle while it pushes all of the data to logplex.
func read(r io.ReadCloser, lines chan<- string, drops, reads *uint64) {
	rdr := bufio.NewReader(r)
	for {
		line, err := rdr.ReadString('\n')
		if err == nil {
			// If we have an unbuffered chanel, we don't want to drop lines.
			// In this case we will apply back-pressure to callers of read.
			if cap(lines) == 0 {
				lines <- line
				atomic.AddUint64(reads, 1)
			} else {
				select {
				case lines <- line:
					atomic.AddUint64(reads, 1)
				default:
					atomic.AddUint64(drops, 1)
				}
			}
		} else {
			r.Close()
			return
		}
	}
}

func report(lines chan string, batches chan []string, drops, reads *uint64) {
	for _ = range time.Tick(time.Second) {
		d := atomic.LoadUint64(drops)
		r := atomic.LoadUint64(reads)
		atomic.AddUint64(drops, -d)
		atomic.AddUint64(reads, -r)
		fmt.Fprintf(os.Stdout, "reads=%d drops=%d lines=%d batches=%d\n", r, d, len(lines), len(batches))
	}
}

func main() {
	var drops uint64 = 0 //count the number of droped lines
	var reads uint64 = 0 //count the number of read lines
	batches := make(chan []string)
	lines := make(chan string, frontBuff)

	go report(lines, batches, &drops, &reads)
	go handle(lines, batches, batchSize, wait)
	for i := 0; i < workerCount; i++ {
		go outlet(batches, logplexToken, logplexUrl.String(), procid, skipHeaders)
	}

	if len(socket) == 0 {
		read(os.Stdin, lines, &drops, &reads)
		reqInFlight.Wait()
	} else {
		l, err := net.Listen("unix", socket)
		if err != nil {
			log.Fatal(err)
		}
		for {
			conn, err := l.Accept()
			if err != nil {
				fmt.Printf("Accept error. err=%v", err)
			}
			go read(conn, lines, &drops, &reads)
		}
	}
}
