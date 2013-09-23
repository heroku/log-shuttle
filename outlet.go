package main

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

type HttpOutlet struct {
	Inbox    chan string
	Outbox   chan []string
	InFlight *sync.WaitGroup
	Config   *ShuttleConfig
	client   *http.Client
}

func NewOutlet(conf *ShuttleConfig, inbox chan string, inflight *sync.WaitGroup) *HttpOutlet {
	h := new(HttpOutlet)
	h.Config = conf
	h.Inbox = inbox
	h.InFlight = inflight
	h.Outbox = make(chan []string, h.Config.BatchSize)
	httpTransport := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: conf.SkipVerify}}
	h.client = &http.Client{Transport: httpTransport}
	return h
}

// Transfer facilitates the handoff between stdin/sockets & logplex http
// requests. If there is high volume traffic on the lines channel, we
// create batches based on the batchSize flag. For low volume traffic,
// we create batches based on a time interval.
func (h *HttpOutlet) Transfer() error {
	ticker := time.Tick(h.Config.WaitDuration())
	batch := make([]string, 0, h.Config.BatchSize)
	for {
		select {
		case <-ticker:
			if len(batch) > 0 {
				h.Outbox <- batch
				batch = make([]string, 0, h.Config.BatchSize)
			}
		case l := <-h.Inbox:
			batch = append(batch, l)
			if len(batch) == cap(batch) {
				h.Outbox <- batch
				batch = make([]string, 0, h.Config.BatchSize)
			}
		}
	}
}

// Outlet takes batches of log lines and submits them to logplex via HTTP.
// Additionaly it can wrap each log line with a syslog header.
func (h *HttpOutlet) Outlet() {
	//Use only 1 buffer and reset it when we are done.
	//This is to make fewer memory allocations.
	var b bytes.Buffer
	for logs := range h.Outbox {
		if err := h.post(&b, logs); err != nil {
			fmt.Fprintf(os.Stderr, "post-error=%s\n", err)
		}
		b.Reset()
	}
}

func (h *HttpOutlet) post(b *bytes.Buffer, logs []string) error {
	//Track the number of http requests we have in flight.
	defer h.InFlight.Add(-len(logs))

	for _, line := range logs {
		fmt.Fprintf(b, "%d %s", len(line), line)
	}

	req, err := http.NewRequest("POST", h.Config.OutletURL(), b)
	if err != nil {
		return err
	}

	req.Header.Add("Content-Type", "application/logplex-1")
	req.Header.Add("Logplex-Msg-Count", strconv.Itoa(len(logs)))
	resp, err := h.client.Do(req)
	if err != nil {
		return err
	}
	if h.Config.Verbose {
		fmt.Printf("at=post status=%d\n", resp.StatusCode)
	}
	resp.Body.Close()
	return nil
}
