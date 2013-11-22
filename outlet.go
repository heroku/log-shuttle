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

		err := h.post(batch, stats.OutletPostTimingChan, int(stats.ReadAndResetDrops()), int(stats.ReadAndResetLost()))
		if err == nil {
			stats.OutletPostSuccess.Add(1)
		} else {
			fmt.Fprintf(os.Stderr, "post-error=%s\n", err)
			stats.OutletPostError.Add(1)
			stats.IncrementLost(uint64(batch.MsgCount))
		}

		h.batchReturn <- batch
	}
}

func (h *HttpOutlet) post(b *Batch, timingData chan<- float64, drops, lost int) error {
	req, err := http.NewRequest("POST", h.config.OutletURL(), b)
	if err != nil {
		return err
	}

	if lostAndDropped := lost + drops; lostAndDropped > 0 {
		b.WriteDrops(lostAndDropped)
	}

	req.ContentLength = int64(b.Len())
	req.Header.Add("Content-Type", "application/logplex-1")
	req.Header.Add("Logplex-Msg-Count", strconv.Itoa(b.MsgCount))
	req.Header.Add("Logshuttle-Drops", strconv.Itoa(drops))
	req.Header.Add("Logshuttle-Lost", strconv.Itoa(lost))
	resp, err := timePost(h.client, req, timingData)
	if err != nil {
		return err
	}

	if h.config.Verbose {
		fmt.Printf("at=post status=%d\n", resp.StatusCode)
	}

	resp.Body.Close()
	return nil
}

func timePost(client *http.Client, req *http.Request, timingData chan<- float64) (*http.Response, error) {
	defer func(t time.Time) { timingData <- time.Since(t).Seconds() }(time.Now())
	return client.Do(req)
}
