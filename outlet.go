package main

import (
	"bytes"
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

type Destination struct {
	url  string
	lost *Counter
}

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

	destinations := [...]Destination{Destination{"http://foo.com", Counter{}}, Destination{"http://bar.com", Counter{}}}

	for batch := range h.inbox {
		// for destinations {
		h.stats <- NewNamedValue("outlet.inbox.length", float64(len(h.inbox)))

		drops, dropsSince := h.drops.ReadAndReset()
		if drops > 0 {
			batch.WriteDrops(drops, dropsSince)
		}

		lost, lostSince := h.lost.ReadAndReset()
		if lost > 0 {
			batch.WriteLost(lost, lostSince)
		}

		//for dest := range destinations {
		//	wg.Add(1)
		//	go func() {
		//V		h.retryPost(batch, dest)
		//		wg.Done()
		//	}()
		//}
		//wg.Wait()
		h.retryPost(batch)

		h.batchReturn <- batch
		// }
	}
}

// Retry io.EOF errors h.config.MaxAttempts times
func (h *HttpOutlet) retryPost(batch *Batch, dest Destination) {
	for attempts := 1; attempts <= h.config.MaxAttempts; attempts++ {
		err := h.post(batch, dest)
		if err != nil {
			err, eok := err.(*url.Error)
			if eok && err.Err == io.EOF && attempts < h.config.MaxAttempts {
				time.Sleep(RETRY_SLEEP * time.Millisecond)
				continue
			} else {
				ErrLogger.Printf("at=post request_id=%q attempts=%d error=%q\n", batch.UUID.String(), attempts, err)
				h.lost.Add(batch.MsgCount)
				return
			}
		} else {
			return
		}
	}
	return
}

func (h *HttpOutlet) post(batch *Batch, dest Destination) error {
	// extract the destination's lost count and timestamp and append it to the logs we're posting
	lost, lostSince := dest.lost.ReadAndReset()
	lostMsg := fmt.Sprintf("Lost %d messages since %s", lost, lostSince.UTC().Format(BATCH_TIME_FORMAT))
	postBytes := append(batch.Bytes(), []byte(lostMsg))

	req, err := http.NewRequest("POST", dest.url, bytes.NewReader(postBytes))
	if err != nil {
		return err
	}

	req.ContentLength = int64(batch.Len())
	req.Header.Add("Content-Type", "application/logplex-1")
	req.Header.Add("Logplex-Msg-Count", strconv.Itoa(batch.MsgCount))
	req.Header.Add("Logshuttle-Drops", strconv.Itoa(batch.Drops))
	req.Header.Add("Logshuttle-Lost", strconv.Itoa(batch.Lost))
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
