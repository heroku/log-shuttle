package main

import (
	"crypto/tls"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"
)

const (
	RETRY_SLEEP = 100 // will be in ms
)

func StartOutlets(config ShuttleConfig, drops, lost *Counter, stats chan<- NamedValue, inbox <-chan *Batch) *sync.WaitGroup {
	outletWaiter := new(sync.WaitGroup)

	for i := 0; i < config.NumOutlets; i++ {
		outletWaiter.Add(1)
		go func() {
			defer outletWaiter.Done()
			outlet := NewOutlet(config, drops, lost, stats, inbox)
			outlet.Outlet()
		}()
	}

	return outletWaiter
}

type HttpOutlet struct {
	inbox  <-chan *Batch
	stats  chan<- NamedValue
	drops  *Counter
	lost   *Counter
	client *http.Client
	config ShuttleConfig
}

func NewOutlet(config ShuttleConfig, drops, lost *Counter, stats chan<- NamedValue, inbox <-chan *Batch) *HttpOutlet {
	return &HttpOutlet{
		drops:  drops,
		lost:   lost,
		stats:  stats,
		inbox:  inbox,
		config: config,
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

		// drops, dropsSince := h.drops.ReadAndReset()
		//if drops > 0 {
		//		batch.WriteDrops(drops, dropsSince)
		//}

		// lost, lostSince := h.lost.ReadAndReset()
		//if lost > 0 {
		//	batch.WriteLost(lost, lostSince)
		//}

		h.retryPost(batch)
	}
}

// Retry io.EOF errors h.config.MaxAttempts times
func (h *HttpOutlet) retryPost(batch *Batch) {
	for attempts := 1; attempts <= h.config.MaxAttempts; attempts++ {
		err := h.post(batch)
		if err != nil {
			err, eok := err.(*url.Error)
			if eok && err.Err == io.EOF && attempts < h.config.MaxAttempts {
				time.Sleep(RETRY_SLEEP * time.Millisecond)
				continue
			} else {
				ErrLogger.Printf("at=post request_id=%q attempts=%d error=%q\n", batch.UUID.String(), attempts, err)
				h.lost.Add(batch.MsgCount())
				return
			}
		} else {
			return
		}
	}
	return
}

func (h *HttpOutlet) post(batch *Batch) error {
	reader := NewLogplexBatchFormatter(batch, &h.config)

	req, err := http.NewRequest("POST", h.config.OutletURL(), reader)
	if err != nil {
		return err
	}

	req.Header.Add("Content-Type", "application/logplex-1")
	req.Header.Add("Logplex-Msg-Count", strconv.Itoa(reader.MsgCount()))
	//req.Header.Add("Logshuttle-Drops", strconv.Itoa(batch.Drops))
	//req.Header.Add("Logshuttle-Lost", strconv.Itoa(batch.Lost))
	req.Header.Add("X-Request-Id", batch.UUID.String())

	resp, err := h.timeRequest(req)
	if err != nil {
		return err
	}

	switch status := resp.StatusCode; {
	case status >= 400:
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			ErrLogger.Printf("at=post request_id=%q status=%d error_reading_body=%q\n", batch.UUID, status, err)
		} else {
			ErrLogger.Printf("at=post request_id=%q status=%d body=%q\n", batch.UUID, status, body)
		}

	default:
		if h.config.Verbose {
			ErrLogger.Printf("at=post request_id=%q status=%d\n", batch.UUID, status)
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
