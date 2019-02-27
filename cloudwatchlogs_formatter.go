package shuttle

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	v4 "github.com/aws/aws-sdk-go/aws/signer/v4"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
)

const (
	xAmazonTarget             = "Logs_20140328.PutLogEvents"
	xAmazonJSON11             = "application/x-amz-json-1.1"
	cloudWatchLogsServiceName = "logs"
)

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
	signer *v4.Signer
}

// NewCloudWatchLogsFormatterFunc that creates a HTTPFormatterFunc for formatting batched into Cloud Watch Logs requests
// tied to a specific region/host/log group/log stream.
func NewCloudWatchLogsFormatterFunc(region, host, logGroupName, logStreamName string) (NewHTTPFormatterFunc, error) {
	sess, err := session.NewSession(aws.NewConfig().WithRegion(region))
	if err != nil {
		return nil, err
	}

	c := cloudwatchlogs.New(sess)
	d, err := c.DescribeLogStreams(&cloudwatchlogs.DescribeLogStreamsInput{
		LogGroupName:        &logGroupName,
		LogStreamNamePrefix: &logStreamName,
	})
	if err != nil {
		return nil, err
	}

	var token string
	var found bool
	for _, ls := range d.LogStreams {
		if aws.StringValue(ls.LogStreamName) == logStreamName {
			found = true
			token = aws.StringValue(ls.UploadSequenceToken)
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("Unable to find log stream: %s", logStreamName)
	}

	si := v4.NewSigner(sess.Config.Credentials)
	tc := make(chan string, 1)
	tc <- token

	url := "https://" + host
	return func(b Batch, eData []errData, config *Config) HTTPFormatter {
		return &CloudWatchLogsFormatter{batch: b, eData: eData, url: url, signer: si, tokens: tc, region: region, logGroupName: logGroupName, logStreamName: logStreamName}
	}, nil
}

// MsgCount of the request. See HTTPSubFormatter for more info.
func (f *CloudWatchLogsFormatter) MsgCount() int {
	return f.batch.MsgCount() + len(f.eData)
}

// Request to submit for the batch. See HTTPFormatter for more info.
func (f *CloudWatchLogsFormatter) Request() (*http.Request, error) {
	body, err := newCloudWatchLogsBody(f.batch, f.eData, f.logGroupName, f.logStreamName, <-f.tokens)
	if err != nil {
		return nil, err
	}
	r, err := http.NewRequest(http.MethodPost, f.url, body)
	if err != nil {
		return r, err
	}
	r.Header.Add("Content-Type", xAmazonJSON11)
	r.Header.Add("X-Amz-Target", xAmazonTarget)

	_, err = f.signer.Sign(r, body, cloudWatchLogsServiceName, f.region, time.Now())
	f.ReadSeeker = body
	return r, err
}

// HandleResponse to the request that was generated. See ResponseHandler for more info.
func (f *CloudWatchLogsFormatter) HandleResponse(resp *http.Response) error {
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
