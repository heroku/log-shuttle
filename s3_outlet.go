package shuttle

import (
	"io"

	"github.com/heroku/log-shuttle/Godeps/_workspace/src/github.com/kr/s3/s3util"
)

type S3Outlet struct {
	config           Config
	inbox            <-chan Batch
	newFormatterFunc NewHTTPFormatterFunc
}

func NewS3Outlet(s *Shuttle) Outlet {
	return &S3Outlet{
		config:           s.config,
		inbox:            s.Batches,
		newFormatterFunc: s.config.FormatterFunc,
	}
}

func (s3o *S3Outlet) Outlet() {
	//TODO: apply format to url to make unique objects over time
	w, err := s3util.Create(s3o.config.LogsURL, nil, nil)
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
