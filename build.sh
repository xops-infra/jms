#!/bin/bash
# build for linux&windows&mac
RELEASE="v1.0.0-beta.$(date +%Y%m%d)"

# GOOS=darwin GOARCH=arm64 go build -o ./bin/jms-darwin-arm64 -ldflags "-X main.version=$RELEASE"
# GOOS=darwin GOARCH=amd64 go build -o ./bin/jms-darwin-amd64 -ldflags "-X main.version=$RELEASE"
# GOOS=linux GOARCH=amd64 go build -o ./bin/jms-linux-amd64 -ldflags "-X main.version=$RELEASE"
# GOOS=linux GOARCH=arm64 go build -o ./bin/jms-linux-arm64 -ldflags "-X main.version=$RELEASE"
# GOOS=windows GOARCH=amd64 go build -o ./bin/jms-windows-amd64.exe -ldflags "-X main.version=$RELEASE"

push() {
    docker build -t zhoushoujian/jms:$RELEASE . --build-arg="RELEASE=$RELEASE"
    if [ $? -eq 0 ]; then
        docker push zhoushoujian/jms:$RELEASE
    else
        echo "build failed"
    fi
}

if [ "$1" = "push" ]; then
    push
fi