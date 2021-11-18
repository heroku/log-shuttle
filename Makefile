GO_LINKER_SYMBOL := main.version
GO_BUILD_ENV := GOOS=linux GOARCH=amd64

all: test

test:
	go test -v ./...
	go test -v -race ./...

install: ldflags
	go install -v ${LDFLAGS} ./...

update-deps: govendor
	govendor add +ex

govendor:
	go get -u github.com/kardianos/govendor

glv:
	$(eval GO_LINKER_VALUE := $(shell git describe --tags --always))

ldflags: glv
	$(eval LDFLAGS := -ldflags "-X ${GO_LINKER_SYMBOL}=${GO_LINKER_VALUE}")

ver: glv
	$(eval VERSION := $(shell echo ${GO_LINKER_VALUE} | sed s/^v//))

docker: ldflags ver clean-docker-build
	${GO_BUILD_ENV} go build -v -o .docker_build/log-shuttle ${LDFLAGS} ./cmd/log-shuttle
	docker build -t heroku/log-shuttle:${VERSION} ./
	${MAKE} clean-docker-build

clean-docker-build:
	rm -rf .docker_build

docker-push: docker ver
	docker push heroku/log-shuttle:${VERSION}

tmp:
	$(eval TMP := $(shell mktemp -d -t log_shuttle.XXXXX))
