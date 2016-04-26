FROM gliderlabs/alpine:3.3

RUN apk-install ca-certificates

# assumes log-shuttle has already been built into .docker_build/log-shuttle,
# which the Makefile does.
COPY .docker_build/log-shuttle /bin/log-shuttle
ENTRYPOINT ["/bin/log-shuttle"]
