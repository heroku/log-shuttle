module github.com/heroku/log-shuttle

require (
	github.com/aws/aws-sdk-go-v2 v1.26.1
	github.com/aws/aws-sdk-go-v2/config v1.27.7
	github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs v1.35.1
	github.com/aws/aws-sdk-go-v2/service/kinesis v1.27.1
	github.com/heroku/slog v0.0.0-20150110001655-7746152d9340
	github.com/pborman/uuid v0.0.0-20150824212802-cccd189d45f7
	github.com/pebbe/util v0.0.0-20140716220158-e0e04dfe647c
	github.com/rcrowley/go-metrics v0.0.0-20141108142129-dee209f2455f
)

go 1.13
