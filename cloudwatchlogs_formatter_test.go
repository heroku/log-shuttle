package shuttle

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
)

type mockCloudWatchLogsClient struct {
	describeLogStreamsFunc func(ctx context.Context, params *cloudwatchlogs.DescribeLogStreamsInput, optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.DescribeLogStreamsOutput, error)
	putLogEventsFunc       func(ctx context.Context, params *cloudwatchlogs.PutLogEventsInput, optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.PutLogEventsOutput, error)
}

func (m *mockCloudWatchLogsClient) DescribeLogStreams(ctx context.Context, params *cloudwatchlogs.DescribeLogStreamsInput, optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.DescribeLogStreamsOutput, error) {
	if m.describeLogStreamsFunc != nil {
		return m.describeLogStreamsFunc(ctx, params, optFns...)
	}
	return &cloudwatchlogs.DescribeLogStreamsOutput{
		LogStreams: []types.LogStream{
			{
				LogStreamName:       aws.String("test-stream"),
				UploadSequenceToken: aws.String("test-token"),
			},
		},
	}, nil
}

func (m *mockCloudWatchLogsClient) PutLogEvents(ctx context.Context, params *cloudwatchlogs.PutLogEventsInput, optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.PutLogEventsOutput, error) {
	if m.putLogEventsFunc != nil {
		return m.putLogEventsFunc(ctx, params, optFns...)
	}
	return &cloudwatchlogs.PutLogEventsOutput{
		NextSequenceToken: aws.String("next-token"),
	}, nil
}

func createTestFormatter(b Batch, eData []errData, client CloudWatchLogsClient) *CloudWatchLogsFormatter {
	body, err := newCloudWatchLogsBody(b, eData, "group", "stream", "test-token")
	if err != nil {
		panic(err)
	}

	f := &CloudWatchLogsFormatter{
		batch:         b,
		eData:         eData,
		logGroupName:  "group",
		logStreamName: "stream",
		region:        "us-east-1",
		url:           "https://logs.us-east-1.amazonaws.com",
		tokens:        make(chan string, 1),
		client:        client,
		ReadSeeker:    body,
	}
	f.tokens <- "test-token"
	return f
}

func TestCloudWatchLogsFormatter(t *testing.T) {
	config := newTestConfig()
	config.LogsURL = "https://key:secret@logs.us-east-1.amazonaws.com/group/stream"
	b := NewBatch(1)
	b.Add(LogLineOne)
	b.Add(LogLineTwo)

	// Create a mock client
	mockClient := &mockCloudWatchLogsClient{}

	// Test formatter creation
	formatter := createTestFormatter(b, noErrData, mockClient)

	// Test MsgCount
	if formatter.MsgCount() != 2 {
		t.Errorf("Expected MsgCount to be 2, got %d", formatter.MsgCount())
	}

	// Test Request creation
	req, err := formatter.Request()
	if err != nil {
		t.Fatalf("Error creating request: %v", err)
	}

	// Verify request headers
	if req.Header.Get("Content-Type") != xAmazonJSON11 {
		t.Errorf("Expected Content-Type to be %s, got %s", xAmazonJSON11, req.Header.Get("Content-Type"))
	}
	if req.Header.Get("X-Amz-Target") != xAmazonTarget {
		t.Errorf("Expected X-Amz-Target to be %s, got %s", xAmazonTarget, req.Header.Get("X-Amz-Target"))
	}

	// Read and verify request body
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		t.Fatalf("Error reading request body: %v", err)
	}
	t.Logf("Request body: %s", string(body))

	// Test response handling
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       ioutil.NopCloser(bytes.NewBufferString(`{"nextSequenceToken":"next-token"}`)),
	}
	if err := formatter.HandleResponse(resp); err != nil {
		t.Errorf("Error handling response: %v", err)
	}

	// Test error response handling
	errorResp := &http.Response{
		StatusCode: http.StatusBadRequest,
		Body:       ioutil.NopCloser(bytes.NewBufferString(`{"__type":"InvalidParameterException","message":"Invalid parameter"}`)),
	}
	if err := formatter.HandleResponse(errorResp); err == nil {
		t.Error("Expected error handling invalid response, got nil")
	}
}

func TestCloudWatchLogsFormatterErrorCases(t *testing.T) {
	config := newTestConfig()
	config.LogsURL = "https://key:secret@logs.us-east-1.amazonaws.com/group/stream"
	b := NewBatch(1)
	b.Add(LogLineOne)

	// Test with nil client
	formatter := createTestFormatter(b, noErrData, nil)

	_, err := formatter.Request()
	if err == nil {
		t.Error("Expected error with nil client, got nil")
	}

	// Test with invalid token
	formatter = createTestFormatter(b, noErrData, &mockCloudWatchLogsClient{})
	formatter.tokens = make(chan string, 1)
	formatter.tokens <- ""

	_, err = formatter.Request()
	if err == nil {
		t.Error("Expected error with empty token, got nil")
	}
}
