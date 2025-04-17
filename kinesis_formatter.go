package shuttle

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/kinesis"
	"github.com/aws/aws-sdk-go-v2/service/kinesis/types"
)

// KinesisClient defines the interface for Kinesis operations we need
type KinesisClient interface {
	PutRecords(ctx context.Context, params *kinesis.PutRecordsInput, optFns ...func(*kinesis.Options)) (*kinesis.PutRecordsOutput, error)
}

// KinesisFormatter formats batches destined for AWS Kinesis HTTP endpoints
// Kinesis has a very small payload side, so recommend setting config.BatchSize in the 1-3 range so as to not loose logs because we go over the batch size.
// Kinesis formats the Data using the LogplexLineFormatter, which is additionally base64 encoded.
type KinesisFormatter struct {
	records []KinesisRecord
	client  KinesisClient
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

	// Create AWS config with credentials
	cfg, err := awsconfig.LoadDefaultConfig(context.TODO(),
		awsconfig.WithRegion("us-east-1"), // Kinesis region
		awsconfig.WithCredentialsProvider(aws.CredentialsProviderFunc(func(ctx context.Context) (aws.Credentials, error) {
			return aws.Credentials{
				AccessKeyID:     awsKey,
				SecretAccessKey: awsSecret,
			}, nil
		})),
	)
	if err != nil {
		panic(err)
	}

	client := kinesis.NewFromConfig(cfg)
	kf := &KinesisFormatter{
		records: make([]KinesisRecord, 0, b.MsgCount()+len(eData)),
		client:  client,
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

	// Use the AWS SDK v2 to put records
	records := make([]types.PutRecordsRequestEntry, 0, len(kf.records))
	for _, record := range kf.records {
		// Create a buffer to hold the record data
		var buf bytes.Buffer
		if _, err := record.WriteTo(&buf); err != nil {
			return nil, err
		}

		// Parse the JSON to get the data and partition key
		var r struct {
			Data         string `json:"Data"`
			PartitionKey string `json:"PartitionKey"`
		}
		if err := json.Unmarshal(buf.Bytes(), &r); err != nil {
			return nil, err
		}

		// Decode the base64 data
		data, err := base64.StdEncoding.DecodeString(r.Data)
		if err != nil {
			return nil, err
		}

		entry := types.PutRecordsRequestEntry{
			Data:         data,
			PartitionKey: aws.String(r.PartitionKey),
		}
		records = append(records, entry)
	}

	input := &kinesis.PutRecordsInput{
		Records:    records,
		StreamName: aws.String(strings.TrimPrefix(kf.url.Path, "/")),
	}

	_, err = kf.client.PutRecords(context.TODO(), input)
	if err != nil {
		return nil, err
	}

	return req, nil
}

// MsgCount returns the number of records that the formatter is formatting
func (kf *KinesisFormatter) MsgCount() int {
	return len(kf.records)
}
