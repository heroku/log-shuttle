#!/usr/bin/env make -f

deb: tempdir := $(shell mktemp -d tmp.XXXXXXXXXX)
deb: controldir := $(tempdir)/DEBIAN
deb: installpath := $(tempdir)/usr/bin
deb: bin/log-shuttle
	mkdir -p -m 0755 $(controldir)
	echo "Package: log-shuttle" > $(controldir)/control
	echo "Version: $(shell bin/log-shuttle -version)" >> $(controldir)/control
	echo "Architecture: amd64" >> $(controldir)/control
	echo "Maintainer: \"Edward Muller\" <edward@heroku.com>" >> $(controldir)/control
	echo "Section: heroku" >> $(controldir)/control
	echo "Priority: optional" >> $(controldir)/control
	echo "Description: Move logs from a Dyno to Logplex/log-iss/etc." >> $(controldir)/control
	mkdir -p $(installpath)
	install bin/log-shuttle $(installpath)/log-shuttle
	fakeroot dpkg-deb --build $(tempdir) .
	rm -rf $(tempdir)

bin/log-shuttle:
	go get -u github.com/tools/godep
	godep go install -a -ldflags "-X github.com/heroku/log-shuttle.Version $(shell git describe --tags --always)" ./...
	mkdir bin
	cp $$GOPATH/bin/log-shuttle bin

clean:
	rm -rf ./bin/
	rm -f log-shuttle*.deb

build: bin/log-shuttle
