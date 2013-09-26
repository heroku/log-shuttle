package main

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"sync"
)

type HttpOutlet struct {
	inbox    <-chan []string
	inFlight *sync.WaitGroup
	client   *http.Client
	drops    *Counter
	config   ShuttleConfig
}

func NewOutlet(config ShuttleConfig, inflight *sync.WaitGroup, drops *Counter, inbox <-chan []string) *HttpOutlet {
	h := new(HttpOutlet)
	httpTransport := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: config.SkipVerify}}
	h.client = &http.Client{Transport: httpTransport}
	h.inFlight = inflight
	h.drops = drops
	h.inbox = inbox
	h.config = config
	return h
}

// Outlet received batches of log lines from inbox and submits them to logplex
// via HTTP.
func (h *HttpOutlet) Outlet() {
	// Use only 1 buffer and reset it when we are done.  This is to make fewer
	// memory allocations.
	var b bytes.Buffer
	for logs := range h.inbox {
		if err := h.post(&b, logs); err != nil {
			fmt.Fprintf(os.Stderr, "post-error=%s\n", err)
		}
		b.Reset()
	}
}

func (h *HttpOutlet) post(b *bytes.Buffer, logs []string) error {
	//Decrement the number of log line we post (or fail to post)
	defer h.inFlight.Add(-len(logs))

	for _, line := range logs {
		fmt.Fprintf(b, "%d %s", len(line), line)
	}

	req, err := http.NewRequest("POST", h.config.OutletURL(), b)
	if err != nil {
		return err
	}

	req.Header.Add("Content-Type", "application/logplex-1")
	req.Header.Add("Logplex-Msg-Count", strconv.Itoa(len(logs)))
	req.Header.Add("Logshuttle-Drops", strconv.Itoa(int(h.drops.ReadAndReset())))
	resp, err := h.client.Do(req)
	if err != nil {
		return err
	}

	if h.config.Verbose {
		fmt.Printf("at=post status=%d\n", resp.StatusCode)
	}

	resp.Body.Close()
	return nil
}
