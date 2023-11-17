#!/bin/bash
# build for linux&windows&mac

go build -o /usr/bin/jms-go main.go

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