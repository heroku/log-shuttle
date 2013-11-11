# Log Shuttle

Log-shuttle is an open source UNIX program that delivers messages from
applications and daemons to log routers and processors via HTTPs.

One of the motivations behind log-shuttle is to provide a simpler form of
encrypted & authenticated log delivery. Using HTTPs & Basic Authentication is
simpler than the techniques described in RFC5425. TLS transport mapping for
Syslog requires that you maintain both client & server certificates for
authentication. In multi-tenant environments, the maintenance of certificate
management can be quite burdensome.

When using log-shuttle with logplex it is recomended that you spawn 1
log-shuttle per logplex token. This will isolate data between customers and
ensure a good QoS. Log-shuttle accepts input from stdin in a newline (\n)
delimited format. Log-shuttle can also be configured to accept packets via a
`SOCK_DGRAM AF_UNIX` socket (up to 10kb). This can be used with syslog(3)
calls.  Run log-shuttle's help command for more options.

To block as little as possible, log-shuttle will drop outstanding batches if
there are too many that haven't been delivered.

## Hacking on log-shuttle

Fork the repo, hack, submit PRs.

### Local Setup

```bash
$ go version
go version go1.1.2 darwin/amd64
$ git clone https://github.com/heroku/log-shuttle.git
$ cd log-shuttle
$ go build
```

### Testing

```bash
$ go test
```

### Submitting Code

Before starting to work on a feature, drop a line to the [mailing list](https://groups.google.com/d/forum/log-shuttle) to get feedback and pro-tips.

* Keep changes in a feature branch
* Submit PR

### Building on Heroku

```bash
> heroku create -r build -b https://github.com/kr/heroku-buildpack-go.git log-shuttle-build
> git push build master
> heroku open -r build
```
Download deb

## License

Copyright (c) 2012 Ryan R. Smith
Copyright (c) 2013 Heroku Inc.

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
