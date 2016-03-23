FROM gliderlabs/alpine:3.3

RUN apk-install ca-certificates

# assumes gox has already installed the files here
COPY .docker_build/log-shuttle /bin/log-shuttle
ENTRYPOINT ["/bin/log-shuttle"]
