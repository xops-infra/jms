FROM golang:1.20 AS builder
ENV GOPROXY https://goproxy.cn,direct
ENV CGO_ENABLED 0
ENV GOOS linux
ENV GOARCH amd64
WORKDIR /opt
COPY . .
RUN go mod tidy && go build -o jms-go
RUN chmod +x /opt/jms-go

FROM centos:7
LABEL maintainer="zhoushoujianwork@163.com"
COPY --from=builder /opt/jms-go /usr/bin/jms-go
COPY --from=builder /opt/entrypoint.sh /root/entrypoint.sh
WORKDIR /root
RUN chmod +x /root/entrypoint.sh 
RUN ls -al /root
EXPOSE 2222
ENTRYPOINT /root/entrypoint.sh
