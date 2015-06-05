FROM gliderlabs/alpine:3.1

RUN apk-install ca-certificates

# assumes gox has already installed the files here
COPY .docker_build/log-shuttle_linux_amd64 /bin/log-shuttle
ENTRYPOINT ["/bin/log-shuttle"]
