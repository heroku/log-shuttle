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

Kinesis expects a single "record" per put, but log-shuttle can encode multiple
log lines into a single put request. If you want to maintain a 1 to 1 ratio of
log lines to records, set the -batch-size=1. Otherwise use a small batch size
somewhere in the 2-5 lines range, modulo your expected log line max length. ATM
Kinesis has a 50KB record size, so base64(max log line size * batch size) can't
be more than 50KB. log-shuttle does not (atm) enforce this, so you may see lost
batches if you start going over this limit.

Since Kinesis record sizes are pretty small, the -drop flag, which defaults to
true (existing behaviour), has been added. If -drop=false then log-shuttle does
not drop. Log-shuttle can still loose messages because of destination errors
thought.

Log-shuttle expects the following encoding of -logs-url when using Amazon
Kinesis:

    ```
    https://<AWS_KEY>:<AWS_SECRET>@kinesis.<AMAZON_REGION>.amazonaws.com/<STREAM NAME>
    ```

See the [Amazon Endpoints
documentation](http://docs.aws.amazon.com/general/latest/gr/rande.html#ak_region)
for supported regions and hostnames.

## Hacking on log-shuttle

Fork the repo, hack, submit PRs.

### Local Setup

```bash
$ go version
go version go1.4.1 darwin/amd64
# go get -u github.com/tools/godep
$ go get github.com/heroku/log-shuttle
$ cd $GOPATH/src/github.com/heroku/log-shuttle
$ godep go install ./...
```

After that `$GOPATH/bin/log-shuttle` should be available.

### Testing

```bash
$ godep go test ./...
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
(sudo nc -n -k -d -Ul /dev/log & until [ ! -e /dev/log ]; do sleep 0.01; done; sudo chmod a+rw /dev/log) | stdbuf -i0 -o0 tr \\0 \\n | ./log-shuttle -logs-url=... ... -input-format=1
```

## License

Copyright (c) 2013-14 Heroku Inc.

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
