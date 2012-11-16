![img](http://f.cl.ly/items/3o1i1M3i250F1j0Y3r2O/Space-shuttle-Endeavour-008.jpeg)

# Log Shuttle

[Logplex](https://github.com/heroku/logplex) supports HTTP inputs. Each user process will pipe it's `stdout` to log-shuttle. Log-shuttle will POST the data to [Logplex](https://github.com/heroku/logplex).

Problems that log-shuttle solves:

* Remove Syslog dependency between user process & Logplex.
* Improve cross-datacenter security model.
* More control over backpressure.

## Usage

### Install

```bash
# Assuming Go1.
$ go get github.com/heroku/log-shuttle
$ cd $GOPATH/src/github.com/ryandotsmith/log-shuttle
$ go build
```

### Connect Via UNIX Socket

```bash
$ export LOGPLEX_URL=https://logplex.com
$ ./log-shuttle -logplex-token="123" -socket="/tmp/log-shuttle"
$ echo 'hi world\n' | nc -U /tmp/log-shuttle
```

### Connect Via STDOUT

```bash
$ export LOGPLEX_URL=https://logplex.com
$ echo 'hi world\n' | ./log-shuttle -logplex-token="123"
```

### Flags

Run the following command for available flags: `$ log-shuttle -h`

#### -logplex-token

Logplex requries HTTP authentication. log-shuttle uses this flag's value to authenticate against the API in addition to inflating each log's headers with the token.

#### -skip-headers

There are certain cases in which you would not want log-shuttle to prepend log messages with the rfc5424 approved headers. By using the `skip-headers` flag, log-shuttle will not prepend headers before submitting the logs to logplex. If you are skipping headers, please ensure that you have the logplex token included in the headers.

#### -front-buff

The front buffer holds lines while the backend sends them to logplex. If log-shuttle receives large amounts of data with a small front-buff, log-shuttle will drop data. The number of dropped lines will be visible in log-shuttle's STDOUT.

#### -wait

The backend routine that delivers log lines to logplex will execute if the front-buff is full or on a timed schedule --whichever occurs first. The timer is configurable by the wait flag.

#### -batch-size

The batch-size determines how many rfc5424 formatted log-lines to pack into an HTTP request.

### l2met

[l2met](github.com/ryandotsmith/l2met) is a service that will convert log lines into librato metrics. You can point log-shuttle at an l2met service for maximum log leverage. Just set LOGPLEX_URL to your l2met drain URL.

## License

Copyright (c) 2012 Ryan R. Smith

Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
