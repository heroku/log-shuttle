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
	"strconv"
	"sync"
	"time"
)

const (
	RETRY_SLEEP = 100 // will be in ms
)

var (
	userAgent = fmt.Sprintf("log-shuttle/%s (%s; %s; %s; %s)", VERSION, runtime.Version(), runtime.GOOS, runtime.GOARCH, runtime.Compiler)
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

		h.retryPost(batch)
	}
}

// Retry io.EOF errors h.config.MaxAttempts times
func (h *HttpOutlet) retryPost(batch *Batch) {
	var dropData, lostData errData

	dropData.count, dropData.since = h.drops.ReadAndReset()
	dropData.eType = errDrop

	lostData.count, lostData.since = h.lost.ReadAndReset()
	lostData.eType = errLost

	for attempts := 1; attempts <= h.config.MaxAttempts; attempts++ {
		err := h.post(batch, dropData, lostData)
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

func (h *HttpOutlet) post(batch *Batch, dropData, lostData errData) error {
	var contentLength int

	readers := make([]io.Reader, 0, 3)

	if dropData.count > 0 {
		rdr := NewLogplexErrorFormatter(dropData, h.config)
		readers = append(readers, rdr)
		contentLength += rdr.Length()
	}

	if lostData.count > 0 {
		rdr := NewLogplexErrorFormatter(lostData, h.config)
		readers = append(readers, rdr)
		contentLength += rdr.Length()
	}

	msgReader := NewLogplexBatchFormatter(batch, &h.config)
	readers = append(readers, msgReader)
	contentLength += msgReader.Length()

	req, err := http.NewRequest("POST", h.config.OutletURL(), io.MultiReader(readers...))
	if err != nil {
		return err
	}

	req.Header.Add("Content-Type", "application/logplex-1")
	if contentLength > 0 {
		req.ContentLength = int64(contentLength)
	}
	req.Header.Add("Logplex-Msg-Count", strconv.Itoa(msgReader.MsgCount()))
	if dropData.count > 0 {
		req.Header.Add("Logshuttle-Drops", strconv.Itoa(dropData.count))
	}
	if lostData.count > 0 {
		req.Header.Add("Logshuttle-Lost", strconv.Itoa(lostData.count))
	}
	req.Header.Add("X-Request-Id", batch.UUID.String())
	req.Header.Add("User-Agent", userAgent)

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
