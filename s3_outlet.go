package shuttle

import (
	"compress/gzip"
	"fmt"
	"io"
	"net/url"
	"time"

	"github.com/heroku/log-shuttle/Godeps/_workspace/src/github.com/kr/s3"
	"github.com/heroku/log-shuttle/Godeps/_workspace/src/github.com/kr/s3/s3util"
)

// S3Outlet is an Outlet that sends logs to S3
// ATM This is nothing but a hack
type S3Outlet struct {
	config           Config
	idx              int
	sliceDuration    time.Duration
	fileExtension    string
	compress         bool
	s3config         *s3util.Config
	inbox            <-chan Batch
	newFormatterFunc NewHTTPFormatterFunc
}

// NewS3Outlet creates an Outlet that sends logs to an S3 bucket specified by a
// url
func NewS3Outlet(s *Shuttle) (Outlet, error) {
	u, err := url.Parse(s.config.LogsURL)
	if err != nil {
		return nil, err
	}

	outlet := &S3Outlet{
		s3config: &s3util.Config{
			Service: s3.DefaultService,
			Keys:    new(s3.Keys),
		},
		sliceDuration:    2 * time.Hour,
		fileExtension:    "gz",
		config:           s.config,
		inbox:            s.Batches,
		newFormatterFunc: s.config.FormatterFunc,
	}

	outlet.s3config.Keys.AccessKey = u.User.Username()
	outlet.s3config.Keys.SecretKey, _ = u.User.Password()

	v := u.Query()
	if v, ok := v["slice_duration"]; ok {
		if outlet.sliceDuration, err = time.ParseDuration(v[0]); err != nil {
			return nil, err
		}
	}

	if v, ok := v["ext"]; ok {
		outlet.fileExtension = v[0]
	}

	if outlet.fileExtension[len(outlet.fileExtension)-2:] == "gz" {
		outlet.compress = true
	}

	u.RawQuery = ""
	u.User = nil

	outlet.config.LogsURL = u.String()

	return outlet, nil
}

// writes data to an s3 url until the duration happens. If s3o.compress is set,
// data will be written through a gzip.Writer
// returns true if there is no more data to process
func (s3o *S3Outlet) writeSlice(url string, until time.Duration) bool {
	var s3w io.Writer

	w, err := s3util.Create(url, nil, s3o.s3config)
	if err != nil {
		panic(err)
	}
	defer w.Close()
	s3w = w

	if s3o.compress {
		gw := gzip.NewWriter(w)
		defer gw.Close()
		s3w = gw
	}

	endSlice := time.NewTimer(until)
	defer endSlice.Stop()

	for {
		select {
		case batch, ok := <-s3o.inbox:
			if !ok {
				return true
			}
			f := s3o.newFormatterFunc(batch, nil, &s3o.config)
			if _, err = io.Copy(s3w, f); err != nil {
				panic(err)
			}
		case <-endSlice.C:
			return false
		}
	}
}

// Outlet processes the batches and sends logs to s3.
func (s3o *S3Outlet) Outlet() {
	for startTime := time.Now().UTC().Truncate(s3o.sliceDuration); ; startTime = startTime.Add(s3o.sliceDuration) {
		url := s3o.config.LogsURL

		dt := startTime.Format("2006-01-02")
		url += "/" + dt + "/" + dt

		if s3o.sliceDuration < (24 * time.Hour) {
			url += fmt.Sprintf("-%02d", startTime.Hour())
		}
		if s3o.sliceDuration < time.Hour {
			url += fmt.Sprintf("-%02d", startTime.Minute())
		}
		if s3o.sliceDuration < time.Minute {
			url += fmt.Sprintf("-%02d", startTime.Second())
		}

		if s3o.fileExtension != "" {
			url += "." + s3o.fileExtension
		}

		until := startTime.Add(s3o.sliceDuration).Sub(time.Now().UTC())
		if s3o.writeSlice(url, until) {
			return
		}
	}
}
