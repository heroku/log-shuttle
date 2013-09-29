package main

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"sync"
)

type HttpOutlet struct {
	inbox            <-chan *Batch
	batchReturn      chan<- *Batch
	linesInFlight    *sync.WaitGroup
	requestsInFlight chan int
	client           *http.Client
	drops            *Counter
	config           ShuttleConfig
}

func NewOutlet(config ShuttleConfig, inflight *sync.WaitGroup, drops *Counter, inbox <-chan *Batch, batchReturn chan<- *Batch) *HttpOutlet {
	h := new(HttpOutlet)
	httpTransport := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: config.SkipVerify}}
	h.client = &http.Client{Transport: httpTransport}
	h.linesInFlight = inflight
	h.requestsInFlight = make(chan int, config.MaxRequests)
	h.drops = drops
	h.inbox = inbox
	h.batchReturn = batchReturn
	h.config = config
	return h
}

// Outlet receives batches from the inbox and submits them to logplex via HTTP.
func (h *HttpOutlet) Outlet() {
	for {
		// grab a batch to work
		batch := <-h.inbox

		// block if the channel is full to limit the number of concurrent requests
		h.requestsInFlight <- 1

		// deliver the batch async
		go h.deliverBatch(batch)
	}
}

func (h *HttpOutlet) deliverBatch(batch *Batch) {
	if err := h.post(batch); err != nil {
		fmt.Fprintf(os.Stderr, "post-error=%s\n", err)
	}

	// return the batch to the pool
	select {
	case h.batchReturn <- batch:
		// passed back, nothing else to do
	default:
		// channel is full, drop this batch on the floor
	}
}

func (h *HttpOutlet) post(b *Batch) error {
	// pull a request marker off the channel to free up space
	defer func() { <-h.requestsInFlight }()
	defer h.linesInFlight.Add(-b.LineCount())

	req, err := http.NewRequest("POST", h.config.OutletURL(), b)
	if err != nil {
		return err
	}

	req.ContentLength = int64(b.Len())

	req.Header.Add("Content-Type", "application/logplex-1")
	req.Header.Add("Logplex-Msg-Count", strconv.Itoa(b.LineCount()))
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
