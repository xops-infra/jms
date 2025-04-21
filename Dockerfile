# Build stage
FROM golang:1.21 AS builder
WORKDIR /data/github/jms
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o jms-linux-amd64 -ldflags "-X main.version=v2.0-$(date +%Y%m%d)"

# Final stage
FROM ubuntu:22.04
LABEL maintainer="zhoushoujianwork@163.com"

COPY --from=builder /data/github/jms/jms-linux-amd64 /usr/bin/jms-go
COPY ./entrypoint.sh /root/entrypoint.sh

RUN chmod +x /usr/bin/jms-go && \
    chmod +x /root/entrypoint.sh

WORKDIR /root
EXPOSE 22222 8013
ENTRYPOINT /root/entrypoint.sh
