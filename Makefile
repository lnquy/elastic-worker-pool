SHELL:=/bin/bash
PROJECT_NAME=elastic-worker-pool
GO_BUILD_ENV=CGO_ENABLED=0 GOOS=linux GOARCH=amd64
GO_FILES=$(shell go list ./... | grep -v /vendor/ | grep -v /examples/ )

BUILD_TAG=$(shell git rev-parse --short HEAD)

.SILENT:

all: fmt vet test

vet:
	go vet -race $(GO_FILES)

lint:
	GO111MODULE=on golangci-lint run .

fmt:
	go fmt $(GO_FILES)

test:
	go test -race -cover -v $(GO_FILES)

integration-test:
	go test -race -cover -tags=integration -v $(GO_FILES)
