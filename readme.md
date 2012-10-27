![img](http://f.cl.ly/items/3o1i1M3i250F1j0Y3r2O/Space-shuttle-Endeavour-008.jpeg)

# Log Shuttle

[Logplex](https://github.com/heroku/logplex) suppoorts HTTP inputs. Each user process will pipe it's `stdout` to log-shuttle. Log-shuttle will POST the data to [Logplex](https://github.com/heroku/logplex).

Problems that log-shuttle solves:

* Remove Syslog dependency between user process & Logplex.
* Improve cross-datacenter security model.
* More control over backpressure.

## Prior Art

* [replacing logger proposal](https://github.com/heroku/runtime-docs/blob/master/replacing-logger-proposal.md)

## Usage

```bash
$ go get github.com/heroku/log-shuttle
$ cd $GOPATH/src/github.com/heroku/log-shuttle
$ LOGPLEX_URL=https://logplex.com WAIT=100 BUFF_SIZE=100
$ go run main.go your-logplex-token
$ echo 'hi world' | nc -U /tmp/log-shuttle.tmp
```
