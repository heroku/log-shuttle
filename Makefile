GO_LINKER_SYMBOL := "main.version"

all: test

test:
	go test ./...

install:
	$(eval GO_LINKER_VALUE := $(shell git describe --tags --always | sed s/^v//))
	go install -a -ldflags "-X ${GO_LINKER_SYMBOL} ${GO_LINKER_VALUE}" ./...

update-deps: godep
	godep save -r ./...

godep:
	go get -u github.com/tools/godep
