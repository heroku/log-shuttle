package main

import (
	"bytes"
	"bufio"
	"fmt"
	"os"
	"time"
	"strconv"
	"net/http"
	"encoding/json"
)

func outlet(batches <-chan []string) {
	for batch := range batches {
		url := "http://httpbin.org/post"
		b, _ := json.Marshal(batch)
		resp, err := http.Post(url, "application/json", bytes.NewBuffer(b))
		defer resp.Body.Close()
		if err != nil {
			fmt.Printf("error=%v\n", err)
		}
		fmt.Printf("status=%v\n", resp.Status)
	}
}

func handle(lines <-chan string, batches chan<- []string) {
	buffsize, _ := strconv.Atoi(os.Getenv("BUFF_SIZE"))
	wait, _ := strconv.Atoi(os.Getenv("WAIT"))
	ticker := time.Tick(time.Millisecond * time.Duration(wait))
	messages := make([]string, 0, buffsize)
	for {
		select {
		case <-ticker:
			batches <- messages
			messages = make([]string, 0, buffsize)
		case l := <-lines:
			messages = append(messages, l)
			if len(messages) == cap(messages) {
				batches <- messages
				messages = make([]string, 0, buffsize)
			}
		}
	}
}

func main() {
	batches := make(chan []string)
	lines := make(chan string)

	go handle(lines, batches)
	go outlet(batches)

	rdr := bufio.NewReader(os.Stdin)
	for {
		line, err := rdr.ReadString('\n')
		if err == nil {
			lines <- line
		}
	}
	return
}
