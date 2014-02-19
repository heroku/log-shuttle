package main

import (
	"crypto/tls"
	"github.com/nu7hatch/gouuid"
	"io/ioutil"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"
)

func StartOutlets(config ShuttleConfig, drops, lost *Counter, stats chan<- NamedValue, inbox <-chan *Batch, batchReturn chan<- *Batch) *sync.WaitGroup {
	outletWaiter := new(sync.WaitGroup)

	for i := 0; i < config.NumOutlets; i++ {
		outletWaiter.Add(1)
		go func() {
			defer outletWaiter.Done()
			outlet := NewOutlet(config, drops, lost, stats, inbox, batchReturn)
			outlet.Outlet()
		}()
	}

	return outletWaiter
}

type HttpOutlet struct {
	inbox       <-chan *Batch
	batchReturn chan<- *Batch
	stats       chan<- NamedValue
	drops       *Counter
	lost        *Counter
	client      *http.Client
	config      ShuttleConfig
}

func NewOutlet(config ShuttleConfig, drops, lost *Counter, stats chan<- NamedValue, inbox <-chan *Batch, batchReturn chan<- *Batch) *HttpOutlet {
	return &HttpOutlet{
		drops:       drops,
		lost:        lost,
		stats:       stats,
		inbox:       inbox,
		batchReturn: batchReturn,
		config:      config,
		client: &http.Client{
			Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: config.SkipVerify},
				ResponseHeaderTimeout: config.Timeout,
				Dial: func(network, address string) (net.Conn, error) {
					return net.DialTimeout(network, address, config.Timeout)
				},
			},
		},
	}
}

// Outlet receives batches from the inbox and submits them to logplex via HTTP.
func (h *HttpOutlet) Outlet() {

	for batch := range h.inbox {
		h.stats <- NewNamedValue("outlet.inbox.length", float64(len(h.inbox)))

		err := h.post(batch)
		if err != nil {
			ErrLogger.Printf("post-error=%q\n", err)
			h.lost.Add(batch.MsgCount)
		}

		h.batchReturn <- batch
	}
}

func (h *HttpOutlet) post(b *Batch) error {
	req, err := http.NewRequest("POST", h.config.OutletURL(), b)
	if err != nil {
		return err
	}

	drops, dropsSince := h.drops.ReadAndReset()
	if drops > 0 {
		b.WriteDrops(drops, dropsSince)
	}

	lost, lostSince := h.lost.ReadAndReset()
	if lost > 0 {
		b.WriteLost(lost, lostSince)
	}

	req.ContentLength = int64(b.Len())
	req.Header.Add("Content-Type", "application/logplex-1")
	req.Header.Add("Logplex-Msg-Count", strconv.Itoa(b.MsgCount))
	req.Header.Add("Logshuttle-Drops", strconv.Itoa(drops))
	req.Header.Add("Logshuttle-Lost", strconv.Itoa(lost))
	uuid, err := uuid.NewV4()
	if err != nil {
		ErrLogger.Printf("at=generate_uuid err=%q\n", err)
	} else {
		req.Header.Add("X-Request-Id", uuid.String())
	}

	resp, err := h.timeRequest(req)
	if err != nil {
		return err
	}

	switch status := resp.StatusCode; {
	case status >= 400:
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			ErrLogger.Printf("at=post status=%d error_reading_body=%q\n", status, err)
		} else {
			ErrLogger.Printf("at=post status=%d body=%q\n", status, body)
		}

	default:
		if h.config.Verbose {
			ErrLogger.Printf("at=post status=%d\n", status)
		}
	}

	resp.Body.Close()
	return nil
}

func (h *HttpOutlet) timeRequest(req *http.Request) (resp *http.Response, err error) {
	defer func(t time.Time) {
		name := "outlet.post"
		if err != nil {
			name += ".failure"
		} else {
			name += ".success"
		}
		h.stats <- NewNamedValue(name, time.Since(t).Seconds())
	}(time.Now())
	return h.client.Do(req)
}
