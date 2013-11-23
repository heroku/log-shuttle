package main

// Total hacks to play with something

import (
	"github.com/kr/logfmt"
	dogstatsd "github.com/ooyala/go-dogstatsd"
	"log"
	"strconv"
)

type MetricOutputter struct {
	inbox   <-chan *LogLine
	appName string
}

func NewMetricOutputter(inbox <-chan *LogLine, config ShuttleConfig) *MetricOutputter {
	mh := new(MetricOutputter)
	mh.inbox = inbox
	return mh
}

func (mo *MetricOutputter) Start() {
	ddClient, err := dogstatsd.New("127.0.0.1:8125")
	if err != nil {
		log.Fatal("starting dogstatsd connection ", err)
	}
	ddClient.Namespace = "reservoir."
	for line := range mo.inbox {
		mh := new(MetricHandler)
		err := logfmt.Unmarshal(line.line, mh)
		if err != nil {
			log.Println("Error unmarhsaling log line for metrics: ", err)
			continue
		}
		if mh.data["ns"] == "writer" && mh.data["at"] == "finish" {
			err = ddClient.Count("files.uploaded", 1, nil, 1)
			if err != nil {
				log.Println("Error sending data (files.uploaded): ", err)
				continue
			}
			ev, err := strconv.ParseFloat(mh.data["elapsed"], 64)
			if err != nil {
				log.Println("Error parsing float ("+mh.data["elapsed"]+"): ", err)
				continue
			}
			err = ddClient.Histogram("s3.write", ev, nil, 1)
			if err != nil {
				log.Println("Error sending data (s3.write): ", err)
				continue
			}

			cv, err := strconv.ParseInt(mh.data["count"], 10, 64)
			if err != nil {
				log.Println("Error parsing int("+mh.data["count"]+"): ", err)
				continue
			}
			err = ddClient.Count("lines.uploaded", cv, nil, 1)
			if err != nil {
				log.Println("Error sending data (lines.uploaded): ", err)
				continue
			}

			csv, err := strconv.ParseInt(mh.data["compressed_size"], 10, 64)
			if err != nil {
				log.Println("Error parsing int("+mh.data["compressed_size"]+"): ", err)
				continue
			}
			err = ddClient.Count("bytes.uploaded", csv, nil, 1)
			if err != nil {
				log.Println("Error sending data (bytes.uploaded): ", err)
				continue
			}

		}

	}
}

type MetricHandler struct {
	logfmt.Handler
	data map[string]string
}

func (mh *MetricHandler) HandleLogfmt(key, val []byte) error {
	mh.data[string(key)] = string(val)
	return nil
}
