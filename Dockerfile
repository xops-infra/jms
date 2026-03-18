# Build stage
FROM golang:1.25.8-bookworm AS builder
WORKDIR /build

# Use a CN-friendly Go module proxy to speed up/avoid blocked downloads.
ENV GOPROXY=https://goproxy.cn,direct \
    GOSUMDB=goproxy.cn

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN GOOS=linux GOARCH=amd64 go build -o jms-linux-amd64 -ldflags "-X main.version=$(date +%Y%m%d)"

# Final stage
FROM debian:bookworm-slim
LABEL maintainer="zhoushoujianwork@163.com"

RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /build/jms-linux-amd64 /usr/bin/jms-go
COPY ./entrypoint.sh /root/entrypoint.sh

RUN chmod +x /usr/bin/jms-go && \
    chmod +x /root/entrypoint.sh

WORKDIR /root
EXPOSE 22222 6060
ENTRYPOINT /root/entrypoint.sh
