[![Travis](https://img.shields.io/travis/heroku/log-shuttle.svg)](https://travis-ci.org/heroku/log-shuttle)
[![Releases](https://img.shields.io/github/release/heroku/log-shuttle.svg)](https://github.com/heroku/log-shuttle/releases)
[![GoDoc](https://godoc.org/github.com/heroku/log-shuttle?status.svg)](http://godoc.org/github.com/heroku/log-shuttle)

# Log Shuttle

Log-shuttle is an open source UNIX program that delivers messages from
applications and daemons to log routers and processors via HTTPs.

One of the motivations behind log-shuttle is to provide a simpler form of
encrypted & authenticated log delivery. Using HTTPs & Basic Authentication is
simpler than the techniques described in RFC5425. TLS transport mapping for
Syslog requires that you maintain both client & server certificates for
authentication. In multi-tenant environments, the maintenance of certificate
management can be quite burdensome.

When using log-shuttle with logplex it is recommended that you spawn 1
log-shuttle per logplex token. This will isolate data between customers and
ensure a good QoS. Log-shuttle accepts input from stdin in a newline (\n)
delimited format.

When using log-shuttle with [Amazon's
Kinesis]("http://aws.amazon.com/kinesis/"), all the details for the region,
stream and access credentials are supplied in the -logs-url (or $LOGS_URL env
variable). See the Kinesis setion of this document.

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

See the [Amazon Endpoints
documentation](http://docs.aws.amazon.com/general/latest/gr/rande.html#ak_region)
for supported regions and hostnames.

#### Kinesis Caveats

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

## Install

```bash
$ go get -u github.com/heroku/log-shuttle/...
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
$ go test -v ./...
```

### Submitting Code

* Open an issue on [GitHub](https://github.com/heroku/log-shuttle/issues?state=open).
* Keep changes in a feature branch
* Submit PR

### Replacing local syslog

Libc uses a local AF_UNIX SOCK_DGRAM (or SOCK_STREAM) for syslog(3) calls. Most unix utils use the syslog(3) call to log to syslog. You can have log-shuttle transport those messages too with a little help from some other standard unix programs.

1. Stop your local syslog
2. rm -f /dev/log
3. us netcat, tr, stdbuf to read connections to /dev/log and convert the \0 terminator to \n

Like so...

```bash
sudo /etc/init.d/rsyslog stop
sudo rm -f /dev/log
(sudo nc -n -k -d -Ul /dev/log & until [ ! -e /dev/log ]; do sleep 0.01; done; sudo chmod a+rw /dev/log) | stdbuf -i0 -o0 tr \\0 \\n | ./log-shuttle -logs-url=... ... -input-format=rfc3164
```

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
