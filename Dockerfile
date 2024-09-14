FROM golang:1.22.3 AS builder

RUN mkdir /app

COPY ./main.go /app
COPY ./scripts /app/scripts
COPY ./go.* /app/

WORKDIR /app

RUN ./scripts/build


FROM alpine:3.15

RUN apk add --no-cache bash

COPY --from=builder /app/bin/test-ncruces /app/test-ncruces

WORKDIR /app

ENTRYPOINT ["/app/test-ncruces"]
