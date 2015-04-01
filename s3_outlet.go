package shuttle

import (
	"fmt"
	"io"
	"net/url"

	"github.com/heroku/log-shuttle/Godeps/_workspace/src/github.com/kr/s3"
	"github.com/heroku/log-shuttle/Godeps/_workspace/src/github.com/kr/s3/s3util"
)

// S3Outlet is an Outlet that sends logs to S3
// ATM This is nothing but a hack
type S3Outlet struct {
	config           Config
	s3config         *s3util.Config
	inbox            <-chan Batch
	newFormatterFunc NewHTTPFormatterFunc
}

// NewS3Outlet creates an Outlet that sends logs to an S3 bucket specified by a
// url
func NewS3Outlet(s *Shuttle) Outlet {
	fmt.Println("New S3")
	u, err := url.Parse(s.config.LogsURL)
	if err != nil {
		panic(err)
	}

	outlet := &S3Outlet{
		s3config: &s3util.Config{
			Service: s3.DefaultService,
			Keys:    new(s3.Keys),
		},
		config:           s.config,
		inbox:            s.Batches,
		newFormatterFunc: s.config.FormatterFunc,
	}

	outlet.s3config.Keys.AccessKey = u.User.Username()
	outlet.s3config.Keys.SecretKey, _ = u.User.Password()

	u.User = nil

	outlet.config.LogsURL = u.String()

	return outlet
}

// Outlet processes the batches and sends logs to s3.
func (s3o *S3Outlet) Outlet() {
	//TODO: apply format to url to make unique objects over time
	w, err := s3util.Create(s3o.config.LogsURL, nil, s3o.s3config)
	//TODO: do something better with errors
	if err != nil {
		panic(err)
	}
	for batch := range s3o.inbox {
		f := s3o.newFormatterFunc(batch, nil, &s3o.config)
		_, err = io.Copy(w, f)
		if err != nil && err != io.EOF {
			panic(err)
		}
	}

	w.Close()
}
