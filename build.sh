#!/bin/bash
APP="bytereel"
MAINFILE="cmd/${APP}/main.go"
VERSION=$(git describe --tags)

# Build for amd64
GOOS=darwin GOARCH=amd64 go build -ldflags "-X main.Version=$VERSION" -o ./bin/${APP}_amd64 $MAINFILE
echo "Build for amd64 complete: ./bin/${APP}_amd64"

# Build for arm64
GOOS=darwin GOARCH=arm64 go build -ldflags "-X main.Version=$VERSION" -o ./bin/${APP}_arm64 $MAINFILE
echo "Build for arm64 complete: ./bin/${APP}_arm64"
