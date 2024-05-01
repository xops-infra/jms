#!/bin/bash
# build for linux&windows&mac

RELEASE="1.0.0_$(date +%Y%m%d)"

GOOS=darwin GOARCH=arm64 go build -o ./bin/jms-darwin-arm64 -ldflags "-X main.version=$RELEASE"

# GOOS=darwin GOARCH=amd64 go build -o ./bin/jms-darwin-amd64 -ldflags "-X main.version=$RELEASE"

# GOOS=linux GOARCH=amd64 go build -o ./bin/jms-linux-amd64 -ldflags "-X main.version=$RELEASE"

# GOOS=windows GOARCH=amd64 go build -o ./bin/jms-windows-amd64.exe -ldflags "-X main.version=$RELEASE"

if [ "$1" = "push" ]; then
    docker build -t zhoushoujian/jms:latest .
    if [ $? -eq 0 ]; then
        echo "build success"
        # 如果$1为 push 则推送，其他则不推送
        if [ "$1" = "push" ]; then
            docker push zhoushoujian/jms:latest
        fi
    else
        echo "build failed"
    fi
fi