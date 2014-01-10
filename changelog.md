### 0.5.1 2014-01-09 Edward Muller (edward@heroku.com)

* Remove socket code. There are ways to do this outside of the program (nc/socat, see commit messages)
* Handle lines longer than logplex can accept (10K), by splitting them up and writing them to the batch as seperate messages.

### 0.5.0 2013-12-18 Edward Muller (edward@heroku.com)

* Log shuttle metrics via a socket (optional)

### 0.3.1 2013-09-30 Edward Muller (edward@heroku.com)

* max requests = 5: Limits in flight requests

### 0.3.0 2013-09-27 Edward Muller (edward@heroku.com)

* use a capped leaky bucket to hold the buffers
* handle deliveries in an async manner
* increate batch size to something sane
* reduce timeout

### 0.2.2 2013-09-24 Dan Peterson (dan@heroku.com)

* Restore skip-headers

### 0.2.1 2013-09-24 Edward Muller <edward@heroku.com>

* Blocking until the queues are drained or the deliveries error.
* Logshuttle-Drop header, which includes the drops since last post.
* Timestamp each incoming log line as quickly as possible.

### 0.2 2013-05-23 Ryan Smith <r@32k.io>

* [linux](https://s3-us-west-2.amazonaws.com/log-shuttle.io/v0.2/linux/amd64/log-shuttle.tar.gz), [osx](https://s3-us-west-2.amazonaws.com/log-shuttle.io/v0.2/osx/log-shuttle.tar.gz), [deb](https://s3-us-west-2.amazonaws.com/log-shuttle.io/v0.2/linux/amd64/log-shuttle_0.2_amd64.deb)
* Remove read/drop reports from stdout.
* Refactor & adding tests.

### 0.1.3 2013-04-16 Ryan Smith <r@32k.io>

* [deb](https://s3-us-west-2.amazonaws.com/log-shuttle/debs/log-shuttle_0.1.3_amd64.deb)
* BUGFIX: the skip-headers feature (bug?) would block all outgoing data

### 0.0.1 2013-04-16 Ryan Smith <r@32k.io>

* The start of versioning
