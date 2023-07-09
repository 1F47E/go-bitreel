#!/bin/bash
rm -rf ./dist
#goreleaser release --snapshot --clean
goreleaser release
