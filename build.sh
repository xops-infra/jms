#!/bin/bash
# build for linux&windows&mac
# v1 旨在 allinone 运行
# v2 拆分api和ssh完全拆分，目标在于支持分布式多 ssh节点，提高容错
RELEASE="v2.0.0-beta.$(date +%Y%m%d)"

# GOOS=darwin GOARCH=arm64 go build -o ./bin/jms-darwin-arm64 -ldflags "-X main.version=$RELEASE"
# GOOS=darwin GOARCH=amd64 go build -o ./bin/jms-darwin-amd64 -ldflags "-X main.version=$RELEASE"
GOOS=linux GOARCH=amd64 go build -o ./bin/jms-linux-amd64 -ldflags "-X main.version=$RELEASE"
# GOOS=linux GOARCH=arm64 go build -o ./bin/jms-linux-arm64 -ldflags "-X main.version=$RELEASE"
# GOOS=windows GOARCH=amd64 go build -o ./bin/jms-windows-amd64.exe -ldflags "-X main.version=$RELEASE"

docker build -t zhoushoujian/jms:$RELEASE . --build-arg="RELEASE=$RELEASE"
if [ $? -eq 0 ]; then
    echo "build success"
else
    echo "build failed"
    exit 1
fi

echo "build success"

push() {
    docker push zhoushoujian/jms:$RELEASE && \
    docker tag zhoushoujian/jms:$RELEASE zhoushoujian/jms:latest && \
    docker push zhoushoujian/jms:latest
}

if [ "$1" = "push" ]; then
    echo "push"
    push
fi