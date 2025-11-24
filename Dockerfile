FROM golang:1.25 as builder

WORKDIR /app

ENV CGO_ENABLED=0

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . ./

RUN go build -o ./bin/auto-message-sender ./cmd/api

FROM ubuntu:22.04

RUN apt-get update && \
    apt-get install -y ca-certificates && \
    rm -rf /var/lib/apt/lists/*

COPY --from=builder /app/config.json /etc/auto-message-sender/config/config.json
COPY --from=builder /app/bin/auto-message-sender /opt/app/auto-message-sender

EXPOSE 6060

ENTRYPOINT ["/opt/app/auto-message-sender","--config","/etc/auto-message-sender/config/config.json"]