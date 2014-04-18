package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"runtime"
	"sync"
	"time"
)

const (
	RETRY_SLEEP = 100 // will be in ms
)

var (
	userAgent = fmt.Sprintf("log-shuttle/%s (%s; %s; %s; %s)", VERSION, runtime.Version(), runtime.GOOS, runtime.GOARCH, runtime.Compiler)
)

func StartOutlets(config ShuttleConfig, drops, lost *Counter, stats chan<- NamedValue, inbox <-chan Batch) *sync.WaitGroup {
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
	inbox  <-chan Batch
	stats  chan<- NamedValue
	drops  *Counter
	lost   *Counter
	client *http.Client
	config ShuttleConfig
}

func NewOutlet(config ShuttleConfig, drops, lost *Counter, stats chan<- NamedValue, inbox <-chan Batch) *HttpOutlet {
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

		h.retryPost(batch)
	}
}

// Retry io.EOF errors h.config.MaxAttempts times
func (h *HttpOutlet) retryPost(batch Batch) {
	var dropData, lostData errData

	edata := make([]errData, 0, 2)

	dropData.count, dropData.since = h.drops.ReadAndReset()
	if dropData.count > 0 {
		dropData.eType = errDrop
		edata = append(edata, dropData)
	}

	lostData.count, lostData.since = h.lost.ReadAndReset()
	if lostData.count > 0 {
		lostData.eType = errLost
		edata = append(edata, lostData)
	}

	uuid := batch.UUID.String()

	for attempts := 1; attempts <= h.config.MaxAttempts; attempts++ {
		formatter := NewLogplexBatchFormatter(batch, edata, &h.config)
		err := h.post(formatter, uuid)
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

func (h *HttpOutlet) post(formatter Formatter, uuid string) error {

	req, err := http.NewRequest("POST", h.config.OutletURL(), formatter)
	if err != nil {
		return err
	}

	if cl := formatter.ContentLength(); cl > 0 {
		req.ContentLength = cl
	}
	req.Header.Add("X-Request-Id", uuid)
	req.Header.Add("User-Agent", userAgent)

	for k, v := range formatter.Headers() {
		req.Header.Add(k, v)
	}

	resp, err := h.timeRequest(req)
	if err != nil {
		return err
	}

	switch status := resp.StatusCode; {
	case status >= 400:
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			ErrLogger.Printf("at=post request_id=%q status=%d error_reading_body=%q\n", uuid, status, err)
		} else {
			ErrLogger.Printf("at=post request_id=%q status=%d body=%q\n", uuid, status, body)
		}

	default:
		if h.config.Verbose {
			ErrLogger.Printf("at=post request_id=%q status=%d\n", uuid, status)
		}
	}

	resp.Body.Close()
	return nil
}

func (h *HttpOutlet) timeRequest(req *http.Request) (resp *http.Response, err error) {
	defer func(t time.Time) {
		name := "outlet.post.time"
		if err != nil {
			name += ".failure"
		} else {
			name += ".success"
		}
		h.stats <- NewNamedValue(name, time.Since(t).Seconds())
	}(time.Now())
	return h.client.Do(req)
}
