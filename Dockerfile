FROM golang:1.17.4 AS builder

COPY ./garbagecollect /app/garbagecollect
COPY ./handlers /app/handlers
COPY ./random /app/random
COPY ./static /app/static
COPY ./store /app/store
COPY ./templates /app/templates
COPY ./types /app/types
COPY ./go.* /app/
COPY ./main.go /app/

WORKDIR /app

RUN GOOS=linux GOARCH=amd64 \
    go build \
      -tags netgo \
      -ldflags '-w -extldflags "-static"' \
      -o /app/picoshare \
      main.go

FROM debian:stable-20211011-slim AS litestream_downloader

ARG litestream_version="v0.3.7"
ARG litestream_binary_tgz_filename="litestream-${litestream_version}-linux-amd64-static.tar.gz"

WORKDIR /litestream

RUN set -x && \
    apt-get update && \
    DEBIAN_FRONTEND=noninteractive apt-get install -y \
      ca-certificates \
      wget
RUN wget "https://github.com/benbjohnson/litestream/releases/download/${litestream_version}/${litestream_binary_tgz_filename}"
RUN tar -xvzf "${litestream_binary_tgz_filename}"

FROM alpine:3.15

RUN apk add --no-cache bash

COPY --from=builder /app/picoshare /app/picoshare
COPY --from=litestream_downloader /litestream/litestream /app/litestream
COPY ./docker-entrypoint /app/docker-entrypoint
COPY ./litestream.yml /etc/litestream.yml
COPY ./static /app/static
COPY ./templates /app/templates

WORKDIR /app

ENTRYPOINT ["/app/docker-entrypoint"]
