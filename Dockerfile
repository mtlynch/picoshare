FROM golang:1.17.4 AS builder

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

FROM alpine:3.15

RUN apk add --no-cache bash

COPY --from=builder /app/picoshare /app/picoshare
COPY ./static /app/static
COPY ./templates /app/templates

WORKDIR /app

ENTRYPOINT ["/app/picoshare"]
