FROM golang:1.21 AS builder
ENV GOPROXY https://goproxy.cn,direct
ENV CGO_ENABLED 0
ENV GOOS linux
ENV GOARCH amd64
COPY ./bin/jms-linux-amd64 /tmp/
COPY ./entrypoint.sh /tmp/
RUN chmod +x /tmp/jms-linux-amd64
RUN chmod +x /tmp/entrypoint.sh 

FROM amd64/centos:7
LABEL maintainer="zhoushoujianwork@163.com"
COPY --from=builder /tmp/jms-linux-amd64 /usr/bin/jms-go
COPY --from=builder /tmp/entrypoint.sh /root/entrypoint.sh
WORKDIR /root
EXPOSE 2222
ENTRYPOINT /root/entrypoint.sh
