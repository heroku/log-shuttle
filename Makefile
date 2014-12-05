#!/usr/bin/env make -f

deb: tempdir := $(shell mktemp -d tmp.XXXXXXXXXX)
deb: controldir := $(tempdir)/DEBIAN
deb: controlfile := $(controldir)/control
deb: installpath := $(tempdir)/usr/bin
deb: bin/log-shuttle
	mkdir -p -m 0755 $(controldir)
	echo "Package: log-shuttle" > $(controlfile)
	echo "Version: $(shell bin/log-shuttle -version)" >> $(controlfile)
	echo "Architecture: amd64" >> $(controlfile)
	echo "Maintainer: \"Edward Muller\" <edward@heroku.com>" >> $(controlfile)
	echo "Section: heroku" >> $(controlfile)
	echo "Priority: optional" >> $(controlfile)
	echo "Description: Move logs from a Dyno to Logplex/log-iss/etc." >> $(controlfile)
	mkdir -p $(installpath)
	install bin/log-shuttle $(installpath)/log-shuttle
	fakeroot dpkg-deb --build $(tempdir) .
	rm -rf $(tempdir)

# This is largely here so you can do `make build` outside of a heroku build slug and basically get the same thing
bin/log-shuttle:
	go get -u github.com/tools/godep
	godep go install -a -ldflags "-X github.com/heroku/log-shuttle.Version $(shell git describe --tags --always)" ./...
	mkdir bin
	cp $$GOPATH/bin/log-shuttle bin

clean:
	rm -rf ./bin/
	rm -f log-shuttle*.deb

build: bin/log-shuttle
