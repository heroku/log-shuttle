![img](http://f.cl.ly/items/3o1i1M3i250F1j0Y3r2O/Space-shuttle-Endeavour-008.jpeg)

# Log Shuttle

[Logplex](https://github.com/heroku/logplex) suppoorts HTTP inputs. Each user process will pipe it's `stdout` to log-shuttle. Log-shuttle will POST the data to [Logplex](https://github.com/heroku/logplex).

Problems that log-shuttle solves:

* Remove Syslog dependency between user process & Logplex.
* Improve cross-datacenter security model.
* More control over backpressure.

## Usage

### Install

```bash
# Assuming Go1.
$ go get github.com/heroku/log-shuttle
$ cd $GOPATH/src/github.com/heroku/log-shuttle
```

### Connect Via UNIX Socket

```bash
$ export LOGPLEX_URL=https://logplex.com WAIT=100 BUFF_SIZE=100
$ ./log-shuttle -logplex-token="123" -socket="/tmp/log-shuttle"
$ echo 'hi world\n' | nc -U /tmp/log-shuttle
```

### Connect Via STDOUT

```bash
$ export LOGPLEX_URL=https://logplex.com WAIT=100 BUFF_SIZE=100
$ echo 'hi world\n' | ./log-shuttle -logplex-token="123"
```

## License

```
Copyright (c) 2012 Ryan R. Smith

Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
```
