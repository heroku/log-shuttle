# Log Shuttle

See the project's [website](http://log-shuttle.io) for synopsis, setup, and usage instructions.

## Hacking on log-shuttle

[![Build Status](https://drone.io/github.com/heroku/log-shuttle/status.png)](https://drone.io/github.com/heroku/log-shuttle/latest)

### Local Setup

```bash
$ go version
go version go1.1 darwin/amd64
$ git clone https://github.com/heroku/log-shuttle.git
$ cd log-shuttle
$ go build
```

### Testing

```bash
$ go test
```

### Submitting Code

Before starting to work an a feature, drop a line to the [mailing list](https://groups.google.com/d/forum/log-shuttle) to get feedback and pro-tips.

* Keep changes in a feature branch
* Submit PR
* Update `logShuttleVersion` in main.go
* Update `VERSION` in Makefile
* Add entry in CHANGELOG
* git tag -a vX.Y -m 'vX.Y had this change' HEAD

### Building on Heroku

```bash
> heroku create -r build -b https://github.com/kr/heroku-buildpack-go.git log-shuttle-build
> git push build master
> heroku open -r build
```
Download deb

## License

Copyright (c) 2012 Ryan R. Smith

Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
