#!/usr/bin/env make -f

VERSION := 0.1.3

tempdir        := $(shell mktemp -d)
controldir     := $(tempdir)/DEBIAN
installpath    := $(tempdir)/usr/bin
buildpath      := .build
buildpackpath  := $(buildpath)/pack
buildpackcache := $(buildpath)/cache

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

deb: build
	mkdir -p -m 0755 $(controldir)
	echo "$$DEB_CONTROL" > $(controldir)/control
	mkdir -p $(installpath)
	install bin/log-shuttle $(installpath)/log-shuttle
	fakeroot dpkg-deb --build $(tempdir) .
	rm -rf $(tempdir)

clean:
	rm -rf $(buildpath)
	rm -f log-shuttle*.deb

build: $(buildpackpath)/bin
	$(buildpackpath)/bin/compile . $(buildpackcache)

$(buildpackcache):
	mkdir -p $(buildpath)
	mkdir -p $(buildpackcache)
	wget -P $(buildpath) http://codon-buildpacks.s3.amazonaws.com/buildpacks/fabiokung/go-git-only.tgz

$(buildpackpath)/bin: $(buildpackcache)
	mkdir -p $(buildpackpath)
	tar -C $(buildpackpath) -zxf $(buildpath)/go-git-only.tgz
