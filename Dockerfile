FROM gliderlabs/alpine:3.3

RUN apk-install ca-certificates

ENV GOVERSION go1.6
ENV LOG_SHUTTLE_VERSION 0.13.1
ENV GOROOT /usr/local/go
ENV GOPATH $HOME/go
ENV PATH $GOROOT/bin:$GOPATH/bin:$PATH
ENV HD=$GOPATH/src/github.com/heroku
ENV LSD=$HD/log-shuttle

# Use wget so we don't have to install curl
# Do it as a single RUN so we can cleanup w/o creating a layer
# first line is for glibc compat with musl
RUN mkdir /lib64 && ln -s /lib/libc.musl-x86_64.so.1 /lib64/ld-linux-x86-64.so.2 && \
    wget "https://storage.googleapis.com/golang/$GOVERSION.linux-amd64.tar.gz" && \
    tar xzf $GOVERSION.linux-amd64.tar.gz -C /usr/local && rm -f $GOVERSION.linux-amd64.tar.gz && \
    wget "https://codeload.github.com/heroku/log-shuttle/tar.gz/v$LOG_SHUTTLE_VERSION" && \
    mkdir -p $HD && \
    tar xzf v$LOG_SHUTTLE_VERSION -C $HD && rm -f v$LOG_SHUTTLE_VERSION && \
    cd $HD && \
    mv log-shuttle-$LOG_SHUTTLE_VERSION log-shuttle && \
    cd log-shuttle && \
    GOBIN=/bin go install -v ./... && \
    cd $HOME && \
    rm -rf $GOROOT $GOPATH

ENTRYPOINT ["/bin/log-shuttle"]
