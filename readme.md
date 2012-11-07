![img](http://f.cl.ly/items/3o1i1M3i250F1j0Y3r2O/Space-shuttle-Endeavour-008.jpeg)

# Log Shuttle

Logplex suppoorts HTTP inputs. Each Dyno will pipe it's `stdout` to log-shuttle. Log-shuttle will POST the data to Logplex.

Problems that log-shuttle solves:

* Remove Syslog dependency between Dynos & Logplex.
* More control over backpressure.

## Prior Art

* [replacing logger proposal](https://github.com/heroku/runtime-docs/blob/master/replacing-logger-proposal.md)

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
$ ./log-shuttle -logplex-token="123" -soket="/tmp/log-shuttle"
$ echo 'hi world\n' | nc -U /tmp/log-shuttle
```

### Connect Via STDOUT

```bash
$ export LOGPLEX_URL=https://logplex.com WAIT=100 BUFF_SIZE=100
$ echo 'hi world\n' | ./log-shuttle -logplex-token="123"
```


