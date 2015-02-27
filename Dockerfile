FROM gliderlabs/alpine:3.1

RUN apk-install go
ENV GOPATH=/go

COPY . /go/src/github.com/heroku/log-shuttle
RUN go get github.com/heroku/log-shuttle/cmd/log-shuttle

RUN apk del go

ENTRYPOINT ["/go/bin/log-shuttle"]
