package main

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

func StartOutlets(count int, config ShuttleConfig, stats *Stats, inbox <-chan *Batch, batchReturn chan<- *Batch) *sync.WaitGroup {
	outletWaiter := new(sync.WaitGroup)

	for ; count > 0; count-- {
		outletWaiter.Add(1)
		go func() {
			defer outletWaiter.Done()
			outlet := NewOutlet(config, stats, inbox, batchReturn)
			outlet.Outlet()
		}()
	}

	return outletWaiter
}

type HttpOutlet struct {
	inbox       <-chan *Batch
	batchReturn chan<- *Batch
	stats       *Stats
	client      *http.Client
	config      ShuttleConfig
}

func NewOutlet(config ShuttleConfig, stats *Stats, inbox <-chan *Batch, batchReturn chan<- *Batch) *HttpOutlet {
	h := new(HttpOutlet)
	httpTransport := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: config.SkipVerify},
		Dial: func(network, address string) (net.Conn, error) {
			return net.DialTimeout(network, address, time.Duration(2*time.Second))
		},
	}

	httpTransport.ResponseHeaderTimeout = config.ResponseTimeout
	h.client = &http.Client{Transport: httpTransport}
	h.stats = stats
	h.inbox = inbox
	h.batchReturn = batchReturn
	h.config = config
	return h
}

// Outlet receives batches from the inbox and submits them to logplex via HTTP.
func (h *HttpOutlet) Outlet() {
	for batch := range h.inbox {

		if err := h.post(batch); err != nil {
			fmt.Fprintf(os.Stderr, "post-error=%s\n", err)
		}

		h.batchReturn <- batch
	}
}

func (h *HttpOutlet) post(b *Batch) error {
	req, err := http.NewRequest("POST", h.config.OutletURL(), b)
	if err != nil {
		return err
	}

	req.ContentLength = int64(b.Len())

	req.Header.Add("Content-Type", "application/logplex-1")
	req.Header.Add("Logplex-Msg-Count", strconv.Itoa(b.LineCount))
	req.Header.Add("Logshuttle-Drops", strconv.Itoa(int(h.stats.Drops.ReadAndReset())))
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
