### 0.14.0 2016-03-22 Edward Muller (edward@heroku.com)

* Build with 1.6, move to vendor/
* Update dependencies
* Update .travis config to release docker images as well

### 0.13.1 2015-11-16 Andrew Gwozdziewycz (apg@heroku.com)

* Build with 1.5.1
* ReadBytes can return data with an io.EOF, don't drop that data
* Doc updates
* Kinesis Sharding support
* Dependency updates

### 0.13.0 2015-08-12 Edward Muller (edward@heroku.com)

* Fix -skip-headers to not bomb out it with -input-format=rfc5424
* Precisely ignore the top-level log-shuttle binary
* Doc updates
* LoadReader to add a reader for the shuttle to manage

### 0.12.0 2015-06-05 Edward Muller (edward@heroku.com)

* Count metrics emit the difference between intervals, not the counter so as to
    make down stream processing easier.
* line.count is now lines.read.count, which makes more sense
* Smaller docker image

### 0.11.0 2015-05-04 Edward Muller (edward@heroku.com)

* Raise all errors to the top and use a standard reporting format in
    log-shuttle.

### 0.10.3 2015-04-29 Edward Muller (edward@heroku.com)

* Add `msg.lost` to the emitted metrics. This is similar to v0.9.X
    `alltime.lost`

### 0.10.2 2015-04-29 Edward Muller (edward@heroku.com)

* Fix -version flag.
* Explicitly default gzip to false

### 0.10.1 2015-04-28 Edward Muller (edward@heroku.com)

* Fix up metrics and names a little bit.
* Fix typos in help text.

### 0.10.0 2015-04-28 Edward Muller (edward@heroku.com)

* -stats-addr re-added, but is a no-op. Will be removed in the future.
* 0.9.7 should have been 0.10.0 :-(

### 0.9.7 2015-04-27 Edward Muller (edward@heroku.com)

* Dockerfile (Thanks @ddollar)
* Preliminary AWS Kinesis Support
* Switch to go-metrics for metrics collection and reporting
* Namespaced most of the code into shuttle so it can be used as a library. The
    command line parts were moved to cmd/log-shuttle/. (Thanks @trestletech and
    others).
* go lint and vet cleanups
* Split out formatting of logs from delivery using the Formatter interfaces
* No more Stats\* config items.
* log-shuttle will no longer listen for requests to report stats on a socket.
* Some stats names have changed because of the switch to go-metrics.
* Line delay stat has been removed.
* log-shuttle will exit with an error when given a URL it considers invalid.
* gzip body support (Thanks @collinvandyck)
* Formalization of the input types.
* `$LOGPLEX_URL` deprecated in favor of `$LOGS_URL`
* `-skip-headers` deprecated in favor of `-input-format=rfc5424`
* Travis + GitHub release integration
* Releases built 100% static (vs. linked to libc)
* godep save -r ./... used to vendor and rewrite deps making log-shuttle
    go-gettable

## 0.9.6 2014-05-16 Edward Muller (edward@heroku.com)

* Restore lost.count and drops.count every metrics poll

### 0.9.5 2014-04-30 Edward Muller (edward@heroku.com)

* Retry more erorrs, backoff a little on retries, but fast path io.EOF

### 0.9.4 2014-04-18 Edward Muller (edward@heroku.com)

* Formalized the Formatter Interface, pretty much only an internal change
### 0.9.3 2014-04-17 Edward Muller (edward@heroku.com)

* -back-buff=50 default, 100 was probably too many
* Simplify LogplexBatchFormatter, adding a few MB/s performance wise.
* Other cleanups

### 0.9.2 2014-04-17 Edward Muller (edward@heroku.com)

* -back-buff=100 default to handle spikes in logs.

### 0.9.1 2014-04-16 Edward Muller (edward@heroku.com)

* Set the user agent to something sane: log-shuttle/0.9.0 (go1.2.1; darwin; amd64; gc)
### 0.9.0 2014-04-16 Edward Muller (edward@heroku.com)

* Large re-write of batching behaviour. Previously logs were written into the
  batch in logplex format. Now incoming data is stored in raw form and
  Formatters, supporting the io.Reader interface, are used to format the batch
  for http.Client's delivery to logplex. This change allowed me to get rid of the
  batch_manager as batches are now much lighter. In testing I've seen an overall
  decerase in RAM and an overall increase in performance as well. This also paves
  the way for other formatters. There is still work to do to abstract outlets a
  bit though, but I'm waiting until I implement delivery in another format to
  worry about that.
* Some changes to the metrics that are emitted. Previously they were all being
  written out as time durations, even when they're not actually.
* reset lastPoll when the socket is polled

### 0.8.1 2014-04-01 Edward Muller (edward@heroku.com)

* Reduce the amount of log-shuttle's in the stats output

### 0.8.0 2014-04-01 Edward Muller (edward@heroku.com)

* Log stats to syslog.info when configured with a delay
* Don't reset stats when they are polled via the socket
* Some refactorings

### 0.7.1 2014-03-10 Edward Muller (edward@heroku.com)

* Wrap the batch in a Reader so that we can properly retry delivery.
  Before this we weren't actually delivering retries. :-(

### 0.7.0 2014-02-25 Edward Muller (edward@heroku.com)

* retry EOF errors -max-attempts times (defaults to 3), with a 100ms sleep in between
* Log # of attempts at end

### 0.6.1 2014-02-20 Edward Muller (edward@heroku.com)

* Add Request Id to Logs

### 0.6.0 2014-02-19 Edward Muller (edward@heroku.com)

* Log using a logger. Default to stderr
* Optional Syslog logger via -log-to-syslog (using syslog.error)

### 0.5.5 2014-02-05 Edward Muller (edward@heroku.com)

* Bump default timeout to 5s

### 0.5.4 2014-01-29 Edward Muller (edward@heroku.com)

* Body logging of post responses in verbose mode (geoff@heroku.com)
* Missing deps

### 0.5.3 2014-01-29 Edward Muller (edward@heroku.com)

* Remove unused -report-every option
* Add X-Request-Id header to each outgoing batch

### 0.5.2 2014-01-10 Edward Muller (edward@heroku.com)

* Seperate L12 (drops) and L13 (lost) into seperate messages.

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
