#!/bin/bash

curl -sfL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | sh -s -- -b $(go env GOPATH)/bin latest
golangci-lint run ./... --enable-all --disable=dupl,lll

