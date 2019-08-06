package internal

import (
	"fmt"
	"log"
	"net/url"
	"regexp"
	"strings"

	shuttle "github.com/heroku/log-shuttle"
)

var (
	AWSEndpoint = regexp.MustCompile(`\A([^\.]+)\.([^\.]+)\.amazonaws\.com(?:\.cn)?\z`)
	AWSHost     = regexp.MustCompile(`\A(?:([^\.]+)\.)(?:([^\.]+)\.)(?:([^\.]+)\.)?amazonaws\.com(?:\.cn)?\z`)
)

const (
	kinesis = "kinesis"
	logs    = "logs"
)

func DetermineOutputFormatter(u *url.URL, errLogger *log.Logger) shuttle.NewHTTPFormatterFunc {
	service, err := DetermineAWSService(u.Host)
	if err != nil {
		return shuttle.NewLogplexBatchFormatter
	}
	region, err := DetermineAWSRegion(u.Host)
	if err != nil {
		return shuttle.NewLogplexBatchFormatter
	}

	switch service {
	case kinesis:
		return shuttle.NewKinesisFormatter
	case logs:
		logGroup, logStream, err := DetermineCloudWatchLogsGroupInfo(u.Path)
		if err != nil {
			errLogger.Fatalf("Error setting up Cloudwatch: %s", err)
		}
		ff, err := shuttle.NewCloudWatchLogsFormatterFunc(region, u.Host, logGroup, logStream)
		if err != nil {
			errLogger.Fatalf("Error setting up Cloudwatch: %s", err)
		}
		return ff
	}
	panic("Detected unsupported AWS Service from URL: " + u.Host)
}

func DetermineAWSRegion(host string) (string, error) {
	found := AWSHost.FindAllStringSubmatch(host, 2)
	if len(found) == 0 || len(found) > 0 && len(found[0]) < 3 {
		return "", fmt.Errorf("Can't determine AWS region from: %q", host)
	}
	return found[0][2], nil
}

func DetermineAWSService(host string) (string, error) {
	found := AWSEndpoint.FindAllStringSubmatch(host, 2)
	if len(found) == 0 || len(found) > 0 && len(found[0]) < 2 {
		return "", fmt.Errorf("Can't determine AWS service from: %q", host)
	}
	return found[0][1], nil
}

func DetermineCloudWatchLogsGroupInfo(path string) (string, string, error) {
	pp := strings.Split(path, "/")
	if len(pp) < 3 {
		return "", "", fmt.Errorf("Can't determine log group and stream name from: %q", path)
	}
	return pp[1], pp[2], nil
}
