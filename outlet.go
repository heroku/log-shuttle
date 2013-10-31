package main

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"sync"
)

func StartOutlets(config ShuttleConfig, stats *ProgramStats, inbox <-chan *Batch, batchReturn chan<- *Batch) *sync.WaitGroup {
	outletWaiter := new(sync.WaitGroup)

	for i := 0; i < config.NumOutlets; i++ {
		outletWaiter.Add(1)
		go func() {
			defer outletWaiter.Done()
			outlet := NewOutlet(config, inbox, batchReturn)
			outlet.Outlet(stats)
		}()
	}

	return outletWaiter
}

type HttpOutlet struct {
	inbox       <-chan *Batch
	batchReturn chan<- *Batch
	client      *http.Client
	config      ShuttleConfig
}

func NewOutlet(config ShuttleConfig, inbox <-chan *Batch, batchReturn chan<- *Batch) *HttpOutlet {
	h := new(HttpOutlet)
	httpTransport := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: config.SkipVerify},
		Dial: func(network, address string) (net.Conn, error) {
			return net.DialTimeout(network, address, config.Timeout)
		},
	}

	httpTransport.ResponseHeaderTimeout = config.Timeout
	h.client = &http.Client{Transport: httpTransport}
	h.inbox = inbox
	h.batchReturn = batchReturn
	h.config = config
	return h
}

// Outlet receives batches from the inbox and submits them to logplex via HTTP.
func (h *HttpOutlet) Outlet(stats *ProgramStats) {
	for batch := range h.inbox {

		if err := h.post(batch, stats); err != nil {
			fmt.Fprintf(os.Stderr, "post-error=%s\n", err)
		}

		h.batchReturn <- batch
	}
}

func (h *HttpOutlet) post(b *Batch, stats *ProgramStats) error {
	req, err := http.NewRequest("POST", h.config.OutletURL(), b)
	if err != nil {
		return err
	}

	drops := int(stats.Drops.ReadAndReset())
	if drops > 0 {
		b.WriteDrops(drops)
	}

	req.ContentLength = int64(b.Len())
	req.Header.Add("Content-Type", "application/logplex-1")
	req.Header.Add("Logplex-Msg-Count", strconv.Itoa(b.LineCount))
	req.Header.Add("Logshuttle-Drops", strconv.Itoa(drops))
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
