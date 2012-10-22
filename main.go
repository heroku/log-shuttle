package main

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"
)

var BuffSize, _ = strconv.Atoi(os.Getenv("BUFF_SIZE"))
var Wait, _ = strconv.Atoi(os.Getenv("WAIT"))

func prepare(batch []string) string {
	result := ""
	length := 0
	for _, msg := range batch {
		t := time.Now().UTC().Format(time.RFC3339 + " ")
		//<prival>version time app-name procid msgid msg \n
		line := "<0>1 " + t + "1234 " + "5678 " + "- " + msg + " \n"
		result += line
		length += len(line)
	}
	return fmt.Sprintf("%d", length) + " " + result
}

func outlet(batches <-chan []string) {
	for batch := range batches {
		url := "http://httpbin.org/post"
		b := bytes.NewBufferString(prepare(batch))
		resp, err := http.Post(url, "application/text", b)
		if err != nil {
			fmt.Printf("error=%v\n", err)
		}
		resp.Body.Close()
		fmt.Printf("status=%v\n", resp.Status)
	}
}

func handle(lines <-chan string, batches chan<- []string) {
	ticker := time.Tick(time.Millisecond * time.Duration(Wait))
	messages := make([]string, 0, BuffSize)
	for {
		select {
		case <-ticker:
			batches <- messages
			messages = make([]string, 0, BuffSize)
		case l := <-lines:
			messages = append(messages, l)
			if len(messages) == cap(messages) {
				batches <- messages
				messages = make([]string, 0, BuffSize)
			}
		}
	}
}

func newConns(l net.Listener) chan net.Conn {
	ch := make(chan net.Conn)
	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				log.Fatal(err)
			}
			ch <- conn
		}
	}()
	return ch
}

func read(c net.Conn, lc chan<- string) {
	rdr := bufio.NewReader(c)
	for {
		line, err := rdr.ReadString('\n')
		if err == nil {
			select {
			case lc<-line:
			default:
			}
		}
	}
}

func main() {
	batches := make(chan []string)
	lines := make(chan string, BuffSize)

	go handle(lines, batches)
	go outlet(batches)

	l, err := net.Listen("unix", "/tmp/shuttle.tmp")
	if err != nil {
		log.Fatal(err)
	}

	c := newConns(l)
	for {
		go read(<-c, lines)
	}
	return
}
