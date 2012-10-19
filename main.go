package main

import (
	"bufio"
	"bytes"
	"fmt"
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
		//http://tools.ietf.org/html/rfc5424
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
		fmt.Printf("status=%v\n", resp.Status)
		resp.Body.Close()
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

func main() {
	batches := make(chan []string)
	lines := make(chan string, BuffSize)

	go handle(lines, batches)
	go outlet(batches)

	rdr := bufio.NewReader(os.Stdin)
	for {
		line, err := rdr.ReadString('\n')
		if err == nil {
			select {
			case lines <- line:
			default:
				fmt.Printf("drop\n")
			}
		}
	}
	return
}
