# Log Shuttle

[![Travis](https://img.shields.io/travis/heroku/log-shuttle.svg)](https://travis-ci.org/heroku/log-shuttle)
[![Releases](https://img.shields.io/github/release/heroku/log-shuttle.svg)](https://github.com/heroku/log-shuttle/releases)
[![GoDoc](https://godoc.org/github.com/heroku/log-shuttle?status.svg)](http://godoc.org/github.com/heroku/log-shuttle)

Log-shuttle is an open source UNIX program that delivers messages from
applications and daemons to log routers and processors via HTTPs.

One of the motivations behind log-shuttle is to provide a simpler form of
encrypted & authenticated log delivery. Using HTTPs & Basic Authentication is
simpler than the techniques described in RFC5425. TLS transport mapping for
Syslog requires that you maintain both client & server certificates for
authentication. In multi-tenant environments, the maintenance of certificate
management can be quite burdensome.

Log-shuttle accepts input from stdin in a newline (\n)
delimited format.

When using log-shuttle with logplex it is recommended that you spawn 1
log-shuttle per logplex token. This will isolate data between tokens and
ensure a good QoS.

When using log-shuttle with either [Amazon's Kinesis]("http://aws.amazon.com/kinesis/") or [Amazon's Cloud Watch
Logs](https://aws.amazon.com/cloudwatch/features/) services all the details for the service are supplied in the
-logs-url (or $LOGS_URL env variable). See the [Amazon Endpoints
documentation](https://docs.aws.amazon.com/general/latest/gr/rande.html)
for supported regions and hostnames. See the Kinesis and Cloud Watch Logs sections of this document.

To block as little as possible, log-shuttle will drop outstanding batches if
it accumulates > -back-buff amount.

## Kinesis

log-shuttle sends data into Kinesis using the
[PutRecords](http://docs.aws.amazon.com/kinesis/latest/APIReference/API_PutRecords.html)
API call. Each Kinesis record is encoded as length prefixed rfc5424 encoded
logs as per [rfc6587](https://tools.ietf.org/html/rfc6587#section-3.4.1) (this
is the same format logplex accepts). One record per log line.

Log-shuttle expects the following encoding of -logs-url when using Amazon
Kinesis:

    ```
    https://<AWS_KEY>:<AWS_SECRET>@kinesis.<AMAZON_REGION>.amazonaws.com/<STREAM NAME>
    ```


### Kinesis Caveats

Things that should be handled better/things you should know:

1. `AWS_SECRET`, `AWS_KEY`, `AMAZON_REGION` & `STREAM NAME` need to be properly
   url encoded.
1. log-shuttle assumes a 200 response means everything is good. Kinesis can
   return a 200, meaning the http request was good, but include per record
   errors in the response body.
1. The maximum number of records in a PutRecords requests is 500, so set the
   batch size no higher than 498 (500 - 2 records for possible drops / lost).
1. Logplex max line length is 10k, Kinesis max record size is 50k of base64
   encoded data. A `-max-line-length` of somewhere less than 37500 should work
   for Kinesis w/o causing errors.
1. Kinesis does not support the -gzip option as that option compresses the body
   of the request.
1. Even with `-kinesis-shards`, no guarantees can be made about writing to unique
   shards.

## CloudWatch Logs

log-shuttle sends logs to CloudWatch Logs using the
[PutLogEvents](https://docs.aws.amazon.com/AmazonCloudWatchLogs/latest/APIReference/API_PutLogEvents.html) API call.
Each log line is a seperate event and delivered in the same order received by log-shuttle.

log-shuttle uses the [aws-sdk-go](https://aws.amazon.com/sdk-for-go/) library to determine the [AWS
credentials](https://docs.aws.amazon.com/sdk-for-go/v1/developer-guide/configuring-sdk.html#specifying-credentials) it
will use, starting CloudWatch Logs sequence token and to sign requests, but does not otherwise use the aws-sdk-go's clients.

log-shuttle expects the following encoding of -logs-url to use Amazon CloudWatch Logs:

    ```
    https://logs.<AMAZON_REGION>.amazonaws.com/<log group name>/<log stream name>
    ```

### Cloudwatch Caveats

Things that should be handled better/things you should know:

1. log-shuttle doesn't do any special handling around service limits atm:
   https://docs.aws.amazon.com/AmazonCloudWatch/latest/logs/cloudwatch_limits_cwl.html
1. log-shuttle needs more testing against CloudWatch to ensure it handles errors and limits better
1. PutLogEvents has a hard upper limit of 5 requests per second. If log-shuttle's input / settings causes > 5 batches
   per second to be created this limit could be exceeded. Modulating batch size and wait duration would be needed to fix
   this on a case by case basis.
1. Really long lines are not split like they are with logplex

## Install

```console
go get -u github.com/heroku/log-shuttle/...
```

After that `$GOPATH/bin/log-shuttle` should be available.

### Making Debs

Requires:

* dpkg (see also `brew install dpkg`)
* go 1.6+

```bash
make debs
```

## Docker

There is a Makefile target named `docker` that can be used to build a docker
image.

## Hacking on log-shuttle

Fork the repo, hack, submit PRs.

### Testing

```bash
go test -v ./...
```

### Submitting Code

* Open an issue on [GitHub](https://github.com/heroku/log-shuttle/issues?state=open).
* Keep changes in a feature branch
* Submit PR

## License

Copyright (c) 2013-15 Heroku Inc.

Permission is hereby granted, free of charge, to any person obtaining a copy of
this software and associated documentation files (the "Software"), to deal in
the Software without restriction, including without limitation the rights to
use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies
of the Software, and to permit persons to whom the Software is furnished to do
so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
