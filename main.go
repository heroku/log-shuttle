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
	"sync/atomic"
	"time"
)

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

func outlet(batches <-chan []string, logplexToken, url, procid string, skipHeaders bool) {
	var b bytes.Buffer
	for batch := range batches {
		prepare(&b, batch, logplexToken, procid, skipHeaders)
		req, _ := http.NewRequest("POST", url, &b)
		req.Header.Add("Content-Type", "application/logplex-1")
		resp, err := http.DefaultClient.Do(req)
		b.Reset()
		if err != nil {
			fmt.Printf("error=%v\n", err)
		} else {
			fmt.Printf("at=logplex-post status=%v\n", resp.StatusCode)
			resp.Body.Close()
		}
	}
}

// Handle facilitates the handoff between stdin/sockets & logplex http
// requests. If there is high volume traffic on the lines channel, we
// create batchces based on the batcheSize flag. For low volume traffic,
// we create batches based on a time interval.
func handle(lines <-chan string, batches chan<- []string, batcheSize, wait int) {
	ticker := time.Tick(time.Millisecond * time.Duration(wait))
	batch := make([]string, 0, batcheSize)
	for {
		select {
		case <-ticker:
			if len(batch) > 0 {
				batches <- batch
				batch = make([]string, 0, batcheSize)
			}
		case l := <-lines:
			batch = append(batch, l)
			if len(batch) == cap(batch) {
				batches <- batch
				batch = make([]string, 0, batcheSize)
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
	frontBuff := flag.Int("front-buff", 0, "Number of messages to buffer in log-shuttle's input chanel.")
	batcheSize := flag.Int("batch-size", 50, "Number of messages to pack into a logplex http request.")
	wait := flag.Int("wait", 500, "Number of ms to flush messages to logplex")
	socket := flag.String("socket", "", "Location of UNIX domain socket.")
	logplexToken := flag.String("logplex-token", "abc123", "Secret logplex token.")
	procid := flag.String("procid", "", "The procid for the syslog payload")
	skipHeaders := flag.Bool("skip-headers", false, "Skip the prepending of rfc5424 headers.")
	flag.Parse()

	logplexUrl, err := url.Parse(os.Getenv("LOGPLEX_URL"))
	if err != nil {
		log.Fatal("Can't parse LOGPLEX_URL: ", err)
	}

	// If the username and password weren't part of the URL, use the
	// logplex-token as the password
	if logplexUrl.User == nil {
		logplexUrl.User = url.UserPassword("token", *logplexToken)
	}

	if logplexUrl.Scheme == "https" {
		//TODO Require a good cert from Logplex.
		tr := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
		http.DefaultTransport = tr
	}

	var drops uint64 = 0 //count the number of droped lines
	var reads uint64 = 0 //count the number of read lines
	batches := make(chan []string)
	lines := make(chan string, *frontBuff)

	go report(lines, batches, &drops, &reads)
	go handle(lines, batches, *batcheSize, *wait)
	go outlet(batches, *logplexToken, logplexUrl.String(), *procid, *skipHeaders)

	if len(*socket) == 0 {
		read(os.Stdin, lines, &drops, &reads)
	} else {
		l, err := net.Listen("unix", *socket)
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
