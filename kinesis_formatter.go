package shuttle

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/bmizerany/aws4"
)

// KinesisFormatter formats batches destined for AWS Kinesis HTTP endpoints
// Kinesis has a very small payload side, so recommend setting config.BatchSize in the 1-3 range so as to not loose logs because we go over the batch size.
// Kinesis formats the Data using the LogplexLineFormatter, which is additionally base64 encoded.
type KinesisFormatter struct {
	header     *bytes.Buffer
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
	u.User = nil // Ensure there is no auth info
	kf := &KinesisFormatter{
		header:     bytes.NewBuffer(make([]byte, 0, 500)),
		formatters: make([]io.Reader, 0, b.MsgCount()+len(eData)),
		footer:     bytes.NewReader([]byte{'"', '}'}),
		keys: &aws4.Keys{
			AccessKey: config.AwsAccessKey,
			SecretKey: config.AwsSecretKey,
		},
		url: u,
	}
	kf.header.WriteString("{")
	kf.header.WriteString(fmt.Sprintf("\"StreamName\":\"%s\",", config.KinesisStreamName))
	kf.header.WriteString(fmt.Sprintf("\"PartitionKey\":\"%s\",", config.Appname))
	kf.header.WriteString("\"Data\":\"")

	for _, edata := range eData {
		kf.formatters = append(kf.formatters, NewLogplexErrorFormatter(edata, *config))
	}

	for _, l := range b.logLines {
		kf.formatters = append(kf.formatters, NewLogplexLineFormatter(l, config))
	}

	return kf
}

//ContentLength doesn't matter for Kinesis, just here to support the interface,
//so return 0
func (kf *KinesisFormatter) ContentLength() int64 {
	return 0
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
		dataReader, dataWriter := io.Pipe()
		kf.rdr = io.MultiReader(kf.header, dataReader, kf.footer)
		go func() {
			encoder := base64.NewEncoder(base64.StdEncoding, dataWriter)
			//TODO: Handle errors somehow?
			io.Copy(encoder, io.MultiReader(kf.formatters...))
			encoder.Close()
			dataWriter.Close()
		}()
	}

	// header get's read completely
	// io.Pipe(

	return kf.rdr.Read(p)
}

//MsgCount doesn't matter for kinesis, just here to support the interface, so
//return 0
func (kf *KinesisFormatter) MsgCount() int {
	return 0
}
