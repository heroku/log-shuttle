package shuttle

import (
	"compress/gzip"
	"context"
	"io/ioutil"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kinesis"
	"github.com/aws/aws-sdk-go-v2/service/kinesis/types"
)

type mockKinesisClient struct {
	putRecordsFunc func(ctx context.Context, params *kinesis.PutRecordsInput, optFns ...func(*kinesis.Options)) (*kinesis.PutRecordsOutput, error)
}

func (m mockKinesisClient) PutRecords(ctx context.Context, params *kinesis.PutRecordsInput, optFns ...func(*kinesis.Options)) (*kinesis.PutRecordsOutput, error) {
	if m.putRecordsFunc != nil {
		return m.putRecordsFunc(ctx, params, optFns...)
	}
	return &kinesis.PutRecordsOutput{
		FailedRecordCount: aws.Int32(0),
		Records: []types.PutRecordsResultEntry{
			{
				SequenceNumber: aws.String("1"),
				ShardId:        aws.String("shard-1"),
			},
		},
	}, nil
}

func TestKinesisFormatter(t *testing.T) {
	config := newTestConfig()
	config.LogsURL = "https://key:secret@foo/Stream"
	b := NewBatch(1)
	b.Add(LogLineOne)
	b.Add(LogLineTwo)
	br := NewKinesisFormatter(b, noErrData, &config)
	d, err := ioutil.ReadAll(br)
	if err != nil {
		t.Fatalf("Error reading everything from batch: %q", err)
	}

	t.Logf("%q", string(d))
}

func TestKinesisFormatterRequest(t *testing.T) {
	config := newTestConfig()
	config.LogsURL = "https://key:secret@kinesis.us-east-1.amazonaws.com/Stream"
	b := NewBatch(1)
	b.Add(LogLineOne)
	b.Add(LogLineTwo)
	kf := NewKinesisFormatter(b, noErrData, &config).(*KinesisFormatter)

	// Create a new mock client
	mockClient := mockKinesisClient{}

	// Replace the real client with a mock
	kf.client = mockClient

	r, err := kf.Request()
	if err != nil {
		t.Fatal("Unexpected error calling Request: ", err)
	}

	// Read the body of the request
	d, err := ioutil.ReadAll(r.Body)
	if err != nil {
		t.Fatalf("Error reading everything from the request: %q", err)
	}

	t.Logf("%q", string(d))
}

func TestKinesisGzip(t *testing.T) {
	config := newTestConfig()
	config.LogsURL = "https://key:secret@foo/Stream"
	b := NewBatch(1)
	b.Add(LogLineOne)
	kf := NewKinesisFormatter(b, noErrData, &config)

	gf := NewGzipFormatter(kf)

	// decompress the bytes and verify the message
	gunzipper, err := gzip.NewReader(gf)
	if err != nil {
		t.Fatal("Error making a reader: ", err)
	}

	// read the uncompressed bytes
	uncompressed, err := ioutil.ReadAll(gunzipper)
	if err != nil {
		t.Fatal("Errors reading the compressed bytes: ", err)
	}

	t.Log("Data: ", string(uncompressed))
}
