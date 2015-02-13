package shuttle

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/heroku/log-shuttle/Godeps/_workspace/src/github.com/bmizerany/aws4"
	"github.com/heroku/log-shuttle/kinesis"
)

// KinesisFormatter formats batches destined for AWS Kinesis HTTP endpoints
// Kinesis has a very small payload side, so recommend setting config.BatchSize in the 1-3 range so as to not loose logs because we go over the batch size.
// Kinesis formats the Data using the LogplexLineFormatter, which is additionally base64 encoded.
type KinesisFormatter struct {
	header     *bytes.Buffer
	records    []kinesis.Record
	formatters []io.Reader
	footer     *bytes.Reader
	rdr        io.Reader
	keys       *aws4.Keys
	url        *url.URL
}

// NewKinesisFormatter constructs a proper HTTPFormatter for Kinesis http targets
func NewKinesisFormatter(b Batch, eData []errData, config *Config) HTTPFormatter {
	u, err := url.Parse(config.LogsURL)
	if err != nil {
		panic(err)
	}
	awsKey := u.User.Username()
	awsSecret, _ := u.User.Password()
	streamName := strings.TrimPrefix(u.Path, "/")

	u.User = nil // Ensure there is no auth info
	u.Path = ""  // Ensure there is no path

	kf := &KinesisFormatter{
		header:  bytes.NewBuffer(make([]byte, 0, 500)),
		records: make([]kinesis.Record, 0, b.MsgCount()+len(eData)),
		footer:  bytes.NewReader([]byte("]}")),
		keys: &aws4.Keys{
			AccessKey: awsKey,
			SecretKey: awsSecret,
		},
		url: u,
	}
	kf.header.WriteString("{")
	kf.header.WriteString(fmt.Sprintf("\"StreamName\":\"%s\",", streamName))
	kf.header.WriteString("\"Records\":[")

	for _, edata := range eData {
		kf.records = append(kf.records, kinesis.Record{Data: NewLogplexErrorFormatter(edata, config), PartitionKey: config.Appname})
	}

	for _, l := range b.logLines {
		kf.records = append(kf.records, kinesis.Record{Data: NewLogplexLineFormatter(l, config), PartitionKey: config.Appname})
	}

	return kf
}

// Request constructs a request for this formatter
// See: http://docs.aws.amazon.com/kinesis/latest/APIReference/API_PutRecord.html
func (kf *KinesisFormatter) Request() (*http.Request, error) {
	req, err := http.NewRequest("POST", kf.url.String(), kf)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/x-amz-json-1.1")
	req.Header.Add("X-Amz-Target", "Kinesis_20131202.PutRecord")
	req.Host = kf.url.Host

	err = aws4.Sign(kf.keys, req)
	if err != nil {
		return nil, err
	}

	return req, nil

}

func (kf *KinesisFormatter) Read(p []byte) (n int, err error) {
	if kf.rdr == nil {
		recordsReader, recordsWriter := io.Pipe()
		kf.rdr = io.MultiReader(kf.header, recordsReader, kf.footer)
		go func() {
			encoder := json.NewEncoder(recordsWriter)
			for i := range kf.records {
				encoder.Encode(kf.records[i])
			}
			recordsWriter.Close()
		}()
	}

	return kf.rdr.Read(p)
}

//MsgCount doesn't matter for kinesis, just here to support the interface, so
//return 0
func (kf *KinesisFormatter) MsgCount() int {
	return 0
}
