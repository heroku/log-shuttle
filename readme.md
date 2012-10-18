# Log Shuttle

![img](http://f.cl.ly/items/162g1W2b1b3Z0V3e3O3J/n119642.jpeg)

Logplex suppoort HTTP inputs. Each Dyno will pipe it's $STDOUT to log-shuttle. Problems that log-shuttle solves:

* Remove TCP dependency between Dynos & Logplex.
* More control over backpressure.


## Usage

```bash
$ go get github.com/heroku/log-shuttle
$ cd $GOPATH/src/github.com/heroku/log-shuttle
$ echo 'hello world\n' | WAIT=100 BUFF_SIZE=100 go run main.go
```
