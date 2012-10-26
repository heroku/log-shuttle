package main

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"fmt"
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

func prepare(batch []string, token string) string {
	result := ""
	length := 0
	for _, msg := range batch {
		t := time.Now().UTC().Format(time.RFC3339 + " ")
		//http://tools.ietf.org/html/rfc5424
		//<prival>version time host procid msgid msg \n
		line := "<0>1 " + t + "1234 " + token + " web.1 " + "- - " + msg + " \n"
		result += line
		length += len(line)
	}
	return fmt.Sprintf("%d", length) + " " + result
}

func outlet(batches <-chan []string, token string) {
	for batch := range batches {
		u, err := url.Parse(LogplexURL)
		if err != nil {
			log.Fatal("can't parse LogplexURL")
		}
		u.User = url.UserPassword("", token)
		b := bytes.NewBufferString(prepare(batch, token))
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		req, _ := http.NewRequest("POST", u.String(), b)
		req.Header.Add("Content-Type", "application/logplex-1")
		client := &http.Client{Transport: tr}
		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("error=%v\n", err)
		}
		resp.Body.Close()
		fmt.Printf("status=%v\n", resp.StatusCode)
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

func read(c net.Conn, lines chan<- string) {
	rdr := bufio.NewReader(c)
	for {
		line, err := rdr.ReadString('\n')
		if err == nil {
			select {
			case lines <- line:
			default:
			}
		}
	}
}

func main() {
	token := os.Args[1]
	batches := make(chan []string)
	lines := make(chan string, BuffSize)

	go handle(lines, batches)
	go outlet(batches, token)

	l, err := net.Listen("unix", "/tmp/log-shuttle.tmp")
	if err != nil {
		log.Fatal(err)
	}
	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Printf("Accept error. err=%v", err)
		}
		go read(conn, lines)
	}
}
