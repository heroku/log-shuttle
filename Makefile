#!/usr/bin/env make -f

VERSION := 0.1.5

tempdir        := $(shell mktemp -d)
controldir     := $(tempdir)/DEBIAN
installpath    := $(tempdir)/usr/bin

define DEB_CONTROL
Package: log-shuttle
Version: $(VERSION)
Architecture: amd64
Maintainer: "Ryan R. Smith" <ryan@heroku.com>
Section: heroku
Priority: optional
Description: Move logs from the Dyno to the Logplex.
endef
export DEB_CONTROL

deb: bin/log-shuttle
	mkdir -p -m 0755 $(controldir)
	echo "$$DEB_CONTROL" > $(controldir)/control
	mkdir -p $(installpath)
	install bin/log-shuttle $(installpath)/log-shuttle
	fakeroot dpkg-deb --build $(tempdir) .
	rm -rf $(tempdir)

bin/log-shuttle:
	git clone git://github.com/kr/heroku-buildpack-go.git .build
	.build/bin/compile . .build/cache/

clean:
	rm -rf ./bin/
	rm -rf .build/
	rm -rf ./.profile.d/
	rm -f log-shuttle*.deb

build: bin/log-shuttle
