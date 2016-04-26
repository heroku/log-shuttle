package shuttle

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/bmizerany/aws4"
)

// KinesisFormatter formats batches destined for AWS Kinesis HTTP endpoints
// Kinesis has a very small payload side, so recommend setting config.BatchSize in the 1-3 range so as to not loose logs because we go over the batch size.
// Kinesis formats the Data using the LogplexLineFormatter, which is additionally base64 encoded.
type KinesisFormatter struct {
	records []KinesisRecord
	keys    *aws4.Keys
	url     *url.URL
	io.Reader
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
		records: make([]KinesisRecord, 0, b.MsgCount()+len(eData)),
		keys:    &aws4.Keys{AccessKey: awsKey, SecretKey: awsSecret},
		url:     u,
	}

	for _, edata := range eData {
		kf.records = append(kf.records, KinesisRecord{llf: NewLogplexErrorFormatter(edata, config)})
	}

	for _, l := range b.logLines {
		kf.records = append(kf.records, KinesisRecord{llf: NewLogplexLineFormatter(l, config)})
	}

	recordsReader, recordsWriter := io.Pipe()
	kf.Reader = io.MultiReader(
		bytes.NewReader([]byte(`{"StreamName":"`+streamName+`","Records":[`)),
		recordsReader,
		bytes.NewReader([]byte("]}")),
	)

	go func() {
		var cs int
		for i, record := range kf.records {
			cs = determineShard(cs, config.KinesisShards)
			record.shard = cs
			if _, err := record.WriteTo(recordsWriter); err != nil {
				recordsWriter.CloseWithError(err)
				return
			}
			if i < len(kf.records)-1 {
				if _, err := recordsWriter.Write([]byte(`,`)); err != nil {
					recordsWriter.CloseWithError(err)
					return
				}
			}
		}
		recordsWriter.Close()
	}()

	return kf
}

// Given a current shard number and the number number of shards return the next shard number
// In the case of 0 KinesisRecord does not add the shard number to the PartitionKey
func determineShard(c, max int) int {
	if max == 1 {
		return 0
	}
	if c == max {
		c = 0
	}
	return c + 1
}

// Request constructs a request for this formatter
// See: http://docs.aws.amazon.com/kinesis/latest/APIReference/API_PutRecord.html
func (kf *KinesisFormatter) Request() (*http.Request, error) {
	req, err := http.NewRequest("POST", kf.url.String(), kf)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/x-amz-json-1.1")
	req.Header.Add("X-Amz-Target", "Kinesis_20131202.PutRecords")
	req.Host = kf.url.Host

	err = aws4.Sign(kf.keys, req)
	if err != nil {
		return nil, err
	}

	return req, nil
}

//MsgCount returns the number of records that the formatter is formatting
func (kf *KinesisFormatter) MsgCount() int {
	return len(kf.records)
}
