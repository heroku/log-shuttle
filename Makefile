GO_LINKER_SYMBOL=main.version
GOOS=linux
GOARCH=amd64
GO_BUILD_ENV="GOOS=$(GOOS) GOARCH=$(GOARCH)"

all: test

test:
	go test -v ./...
	go test -v -race ./...

install: ldflags
	go install -v ${LDFLAGS} ./...

debs: tmp ldflags ver
	$(eval DEB_ROOT := "${TMP}/DEBIAN")
	${GO_BUILD_ENV} go build -v -o ${TMP}/usr/bin/log-shuttle ${LDFLAGS} ./cmd/log-shuttle
	mkdir -p ${DEB_ROOT}
	cat misc/DEBIAN.control | sed s/{{VERSION}}/${VERSION}/ > ${DEB_ROOT}/control
	dpkg-deb -Zgzip -b ${TMP} log-shuttle_${VERSION}_amd64.deb
	rm -rf ${TMP}

glv:
	$(eval GO_LINKER_VALUE := $(shell git describe --tags --always))

ldflags: glv
	$(eval LDFLAGS := "-X ${GO_LINKER_SYMBOL}=${GO_LINKER_VALUE}")

ver: glv
	$(eval VERSION := $(shell echo ${GO_LINKER_VALUE} | sed s/^v//))

docker: glv ver
	docker build --build-arg GO_LINKER_SYMBOL=$(GO_LINKER_SYMBOL) \
		--build-arg GO_LINKER_VALUE=$(GO_LINKER_VALUE) \
		--build-arg GOOS=$(GOOS) --build-arg GARCH=$(GOARCH) \
		-t heroku/log-shuttle:${VERSION} ./

docker-push: docker ver
	docker push heroku/log-shuttle:${VERSION}

tmp:
	$(eval TMP := $(shell mktemp -d -t log_shuttle.XXXXX))
