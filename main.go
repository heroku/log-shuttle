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
	"time"
)

var BuffSize, _ = strconv.Atoi(os.Getenv("BUFF_SIZE"))
var Wait, _ = strconv.Atoi(os.Getenv("WAIT"))
var LogplexURL = os.Getenv("LOGPLEX_URL")
var socket = flag.String("socket", "", "Location of UNIX domain socket.")
var logplexToken = flag.String("logplex-token", "abc123", "Secret logplex token.")

func prepare(w io.Writer, batch []string) {
	for _, msg := range batch {
		t := time.Now().UTC().Format(time.RFC3339 + " ")
		//http://tools.ietf.org/html/rfc5424
		//<prival>version time host procid msgid msg \n
		line := "<0>1 " + t + "1234 " + *logplexToken + " web.1 " + "- - " + msg + " \n"
		fmt.Fprintf(w, "%d %s", len(line), line)
	}
}

func outlet(batches <-chan []string) {
	for batch := range batches {
		u, err := url.Parse(LogplexURL)
		if err != nil {
			log.Fatal("can't parse LogplexURL")
		}
		u.User = url.UserPassword("", *logplexToken)
		var b bytes.Buffer
		prepare(&b, batch)
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		req, _ := http.NewRequest("POST", u.String(), &b)
		req.Header.Add("Content-Type", "application/logplex-1")
		client := &http.Client{Transport: tr}
		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("error=%v\n", err)
		} else {
			fmt.Printf("status=%v\n", resp.StatusCode)
			resp.Body.Close()
		}
	}
}

func handle(lines <-chan string, batches chan<- []string) {
	ticker := time.Tick(time.Millisecond * time.Duration(Wait))
	messages := make([]string, 0, BuffSize)
	for {
		select {
		case <-ticker:
			if len(messages) > 0 {
				batches <- messages
				messages = make([]string, 0, BuffSize)
			}
		case l := <-lines:
			messages = append(messages, l)
			if len(messages) == cap(messages) {
				batches <- messages
				messages = make([]string, 0, BuffSize)
			}
		}
	}
}

func read(r *bufio.Reader, lines chan<- string) {
	for {
		line, err := r.ReadString('\n')
		if err == nil {
			select {
			case lines <- line:
			default:
			}
		}
	}
}

func main() {
	flag.Parse()

	batches := make(chan []string)
	lines := make(chan string, BuffSize)

	go handle(lines, batches)
	go outlet(batches)

	if len(*socket) == 0 {
		rdr := bufio.NewReader(os.Stdin)
		read(rdr, lines)
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
			rdr := bufio.NewReader(conn)
			go read(rdr, lines)
		}
	}
}
