SHELL:=/bin/bash
PROJECT_NAME=elastic-worker-pool
GO_BUILD_ENV=CGO_ENABLED=0 GOOS=linux GOARCH=amd64
GO_FILES=$(shell go list ./... | grep -v /vendor/)

BUILD_TAG=$(shell git rev-parse --short HEAD)

.SILENT:

all: fmt vet install test

vet:
	go vet $(GO_FILES)

lint:
	golangci-lint run $($GO_FILES | tail -n +2)

fmt:
	go fmt $(GO_FILES)

test:
	go test $(GO_FILES) -cover

integration_test:
	go test -tags=integration $(GO_FILES) -cover
