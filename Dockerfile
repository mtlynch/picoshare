FROM golang:1.18.4 AS builder

ARG TARGETPLATFORM

COPY ./cmd /app/cmd
COPY ./garbagecollect /app/garbagecollect
COPY ./handlers /app/handlers
COPY ./random /app/random
COPY ./static /app/static
COPY ./store /app/store
COPY ./templates /app/templates
COPY ./types /app/types
COPY ./go.* /app/

WORKDIR /app

RUN set -x && \
    if [[ "$TARGETPLATFORM" = "linux/arm/v7" ]]; then \
      GOARCH="arm"; \
    elif [[ "$TARGETPLATFORM" = "linux/arm64" ]]; then \
      GOARCH="arm64"; \
    else \
      GOARCH="amd64"; \
    fi && \
    set -u && \
    GOOS=linux \
    go build \
      -tags netgo \
      -ldflags '-w -extldflags "-static"' \
      -o /app/picoshare \
      cmd/picoshare/main.go

FROM debian:stable-20211011-slim AS litestream_downloader

ARG TARGETPLATFORM
ARG litestream_version="v0.3.9"

WORKDIR /litestream

RUN set -x && \
    apt-get update && \
    DEBIAN_FRONTEND=noninteractive apt-get install -y \
      ca-certificates \
      wget

RUN set -x && \
    if [[ "$TARGETPLATFORM" = "linux/arm/v7" ]]; then \
      ARCH="arm7" ; \
    elif [[ "$TARGETPLATFORM" = "linux/arm64" ]]; then \
      ARCH="arm64" ; \
    else \
      ARCH="amd64" ; \
    fi && \
    set -u && \
    litestream_binary_tgz_filename="litestream-${litestream_version}-linux-${ARCH}-static.tar.gz" && \
    wget "https://github.com/benbjohnson/litestream/releases/download/${litestream_version}/${litestream_binary_tgz_filename}" && \
    mv "${litestream_binary_tgz_filename}" litestream.tgz
RUN tar -xvzf litestream.tgz

FROM alpine:3.15

RUN apk add --no-cache bash

COPY --from=builder /app/picoshare /app/picoshare
COPY --from=litestream_downloader /litestream/litestream /app/litestream
COPY ./docker-entrypoint /app/docker-entrypoint
COPY ./litestream.yml /etc/litestream.yml
COPY ./static /app/static
COPY ./templates /app/templates
COPY ./LICENSE /app/LICENSE

WORKDIR /app

ENTRYPOINT ["/app/docker-entrypoint"]
CMD ["-db" "/data/store.db"]
