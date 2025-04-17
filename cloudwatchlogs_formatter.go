package shuttle

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
)

const (
	xAmazonTarget             = "Logs_20140328.PutLogEvents"
	xAmazonJSON11             = "application/x-amz-json-1.1"
	cloudWatchLogsServiceName = "logs"
)

// CloudWatchLogsClient defines the interface for CloudWatch Logs operations we need
type CloudWatchLogsClient interface {
	DescribeLogStreams(ctx context.Context, params *cloudwatchlogs.DescribeLogStreamsInput, optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.DescribeLogStreamsOutput, error)
	PutLogEvents(ctx context.Context, params *cloudwatchlogs.PutLogEventsInput, optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.PutLogEventsOutput, error)
}

// CloudWatchLogsFormatter formats a batch of logs for the Amazon Cloud Watch Logs service.
// NewCloudWatchLogsFormatterFunc should be used to create a HTTPFormatterFunc tied to a specific
// region/logGroupName/logStreamName.
type CloudWatchLogsFormatter struct {
	batch Batch
	eData []errData
	logGroupName,
	logStreamName string
	io.ReadSeeker
	region,
	url string
	tokens chan string
	client CloudWatchLogsClient
}

// NewCloudWatchLogsFormatterFunc that creates a HTTPFormatterFunc for formatting batched into Cloud Watch Logs requests
// tied to a specific region/host/log group/log stream.
func NewCloudWatchLogsFormatterFunc(region, host, logGroupName, logStreamName string) (NewHTTPFormatterFunc, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region))
	if err != nil {
		return nil, err
	}

	client := cloudwatchlogs.NewFromConfig(cfg)
	d, err := client.DescribeLogStreams(context.TODO(), &cloudwatchlogs.DescribeLogStreamsInput{
		LogGroupName:        aws.String(logGroupName),
		LogStreamNamePrefix: aws.String(logStreamName),
	})
	if err != nil {
		return nil, err
	}

	var token string
	var found bool
	for _, ls := range d.LogStreams {
		if aws.ToString(ls.LogStreamName) == logStreamName {
			found = true
			token = aws.ToString(ls.UploadSequenceToken)
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("Unable to find log stream: %s", logStreamName)
	}

	tc := make(chan string, 1)
	tc <- token

	url := "https://" + host
	return func(b Batch, eData []errData, config *Config) HTTPFormatter {
		return &CloudWatchLogsFormatter{
			batch:         b,
			eData:         eData,
			url:           url,
			tokens:        tc,
			region:        region,
			logGroupName:  logGroupName,
			logStreamName: logStreamName,
			client:        client,
		}
	}, nil
}

// MsgCount of the request. See HTTPSubFormatter for more info.
func (f *CloudWatchLogsFormatter) MsgCount() int {
	return f.batch.MsgCount() + len(f.eData)
}

// Request constructs a request for this formatter
func (f *CloudWatchLogsFormatter) Request() (*http.Request, error) {
	if f.client == nil {
		return nil, fmt.Errorf("CloudWatch Logs client is not initialized")
	}

	// Get the sequence token
	token := <-f.tokens
	if token == "" {
		return nil, fmt.Errorf("invalid sequence token")
	}

	// Create the request body
	body, err := newCloudWatchLogsBody(f.batch, f.eData, f.logGroupName, f.logStreamName, token)
	if err != nil {
		return nil, err
	}

	// Create a new request
	req, err := http.NewRequest(http.MethodPost, f.url, body)
	if err != nil {
		return nil, err
	}

	// Set headers
	req.Header.Add("Content-Type", xAmazonJSON11)
	req.Header.Add("X-Amz-Target", xAmazonTarget)

	// Use the AWS SDK v2 to put log events
	events := make([]types.InputLogEvent, 0, len(f.batch.logLines)+len(f.eData))

	// Add error events
	for _, e := range f.eData {
		var msg string
		switch e.eType {
		case errDrop:
			msg = fmt.Sprintf("log-shuttle dropped %d messages since %s", e.count, e.since.String())
		case errLost:
			msg = fmt.Sprintf("log-shuttle lost %d messages since %s", e.count, e.since.String())
		default:
			continue
		}
		events = append(events, types.InputLogEvent{
			Message:   aws.String(msg),
			Timestamp: aws.Int64(time.Now().Round(time.Millisecond).UnixNano() / int64(time.Millisecond)),
		})
	}

	// Add log line events
	for _, ll := range f.batch.logLines {
		events = append(events, types.InputLogEvent{
			Message:   aws.String(string(ll.line)),
			Timestamp: aws.Int64(ll.when.Round(time.Millisecond).UnixNano() / int64(time.Millisecond)),
		})
	}

	input := &cloudwatchlogs.PutLogEventsInput{
		LogGroupName:  aws.String(f.logGroupName),
		LogStreamName: aws.String(f.logStreamName),
		LogEvents:     events,
	}
	if token != "" { // Amazon balks if SequenceToken is set to ""
		input.SequenceToken = aws.String(token)
	}

	_, err = f.client.PutLogEvents(context.TODO(), input)
	if err != nil {
		return nil, err
	}

	f.ReadSeeker = body
	return req, nil
}

// HandleResponse to the request that was generated. See ResponseHandler for more info.
func (f *CloudWatchLogsFormatter) HandleResponse(resp *http.Response) error {
	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Type    string `json:"__type"`
			Message string `json:"message"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
			return fmt.Errorf("error response with status %d: %v", resp.StatusCode, err)
		}
		return fmt.Errorf("%s: %s", errResp.Type, errResp.Message)
	}

	d := json.NewDecoder(resp.Body)
	var p struct {
		NextSequenceToken string `json:"nextSequenceToken"`
	}
	if err := d.Decode(&p); err != nil {
		return err
	}

	// FIXME: It's a problem if there is no nextSequenceToken
	if p.NextSequenceToken != "" {
		f.tokens <- p.NextSequenceToken
	}

	return nil
}

type cwlEvent struct {
	Message   string `json:"message"`
	Timestamp int64  `json:"timestamp"`
}

type cwlPut struct {
	Events        []cwlEvent `json:"logEvents"`
	GroupName     string     `json:"logGroupName"`
	StreamName    string     `json:"logStreamName"`
	SequenceToken *string    `json:"sequenceToken"`
}

func newCloudWatchLogsBody(b Batch, eData []errData, logGroupName, logStreamName, st string) (io.ReadSeeker, error) {
	logs := cwlPut{
		Events:     make([]cwlEvent, 0),
		GroupName:  logGroupName,
		StreamName: logStreamName,
	}
	if st != "" { // Amazon balks if SequenceToken is set to ""
		logs.SequenceToken = &st
	}

	var msg string
	for _, e := range eData {
		switch e.eType {
		case errDrop:
			msg = fmt.Sprintf("log-shuttle dropped %d messages since %s", e.count, e.since.String())
		case errLost:
			msg = fmt.Sprintf("log-shuttle lost %d messages since %s", e.count, e.since.String())
		default:
			continue
		}
		logs.Events = append(
			logs.Events,
			cwlEvent{
				msg,
				time.Now().Round(time.Millisecond).UnixNano() / int64(time.Millisecond),
			},
		)
	}

	for _, ll := range b.logLines {
		logs.Events = append(
			logs.Events,
			cwlEvent{
				string(ll.line),
				ll.when.Round(time.Millisecond).UnixNano() / int64(time.Millisecond),
			},
		)
	}

	d, err := json.Marshal(&logs)
	return bytes.NewReader(d), err
}
