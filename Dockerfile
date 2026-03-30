# syntax=docker/dockerfile:1.7
# Build Go binary
FROM golang:1.25.8-bookworm AS go-builder
WORKDIR /build

# Use a CN-friendly Go module proxy to speed up/avoid blocked downloads.
ENV GOPROXY=https://goproxy.cn,direct \
    GOSUMDB=goproxy.cn

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod,sharing=locked \
    --mount=type=cache,target=/root/.cache/go-build,sharing=locked \
    go mod download

COPY . .
RUN --mount=type=cache,target=/go/pkg/mod,sharing=locked \
    --mount=type=cache,target=/root/.cache/go-build,sharing=locked \
    GOOS=linux GOARCH=amd64 go build -o jms-linux-amd64 -ldflags "-X main.version=$(date +%Y%m%d)"

# Build web assets
FROM node:20-bookworm AS web-builder
WORKDIR /web

COPY web/package.json web/package-lock.json ./
RUN npm ci

COPY web/ ./
RUN npm run build

# Final stage
FROM debian:bookworm-slim
LABEL maintainer="zhoushoujianwork@163.com"

RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates nginx \
    && rm -f /etc/nginx/sites-enabled/default /etc/nginx/conf.d/default.conf \
    && rm -rf /var/lib/apt/lists/*

COPY --from=go-builder /build/jms-linux-amd64 /usr/bin/jms-go
COPY --from=web-builder /web/dist /usr/share/nginx/html
COPY ./web/nginx.conf /etc/nginx/conf.d/default.conf
COPY ./entrypoint.sh /root/entrypoint.sh

RUN chmod +x /usr/bin/jms-go && \
    chmod +x /root/entrypoint.sh

WORKDIR /root
EXPOSE 80 22222 6060 8013
ENTRYPOINT /root/entrypoint.sh
