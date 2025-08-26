# Makefile

# Variables
TOOLS_BIN_DIR := $(shell go env GOPATH)/bin
GOIMPORTS := $(TOOLS_BIN_DIR)/goimports
GOLANGCI_LINT := $(TOOLS_BIN_DIR)/golangci-lint
GO_TEST_COVERAGE := $(TOOLS_BIN_DIR)/go-test-coverage

# Build variables
BINARY_NAME := sargantana-go
VERSION ?= dev
LDFLAGS := -s -w -X main.version=$(VERSION)

# Tasks
.PHONY: all format test clean lint deps test bench install-golangci-lint build build-all test-coverage check-coverage install-goimports install-go-test-coverage ci clean-dist

test:
	@echo "Running backend tests..."
	go test ./...

test-coverage:
	@echo "Running backend tests with coverage..."
	go test -covermode=atomic -coverprofile=coverage.out ./...

check-coverage: install-go-test-coverage
	go test ./... -coverprofile=./coverage.out -covermode=atomic -coverpkg=./...
	$(GO_TEST_COVERAGE) --config=./.testcoverage.yml

install-goimports:
	go install golang.org/x/tools/cmd/goimports@latest

install-go-test-coverage:
	go install github.com/vladopajic/go-test-coverage/v2@latest

install-golangci-lint:
	@if ! command -v golangci-lint &> /dev/null; then \
		echo "Installing golangci-lint..."; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin v2.4.0; \
	fi

format: install-goimports
	@echo "Formatting code..."
	go fmt ./... && $(GOIMPORTS) -w .

lint: format install-golangci-lint
	@echo "Linting code..."
	go vet ./...
	$(GOLANGCI_LINT) run ./...

deps:
	@echo "Tidying go.mod and go.sum..."
	go mod tidy
	go mod verify
	go mod download

bench:
	@echo "Running benchmarks..."
	go test -bench=. ./...

build:
	@echo "Building application..."
	go build -v -ldflags="$(LDFLAGS)" -o bin/$(BINARY_NAME) ./main

# Build for all platforms
build-all: clean-dist
	@echo "Building for all platforms..."
	@mkdir -p dist
	@echo "Building for Linux AMD64..."
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o dist/$(BINARY_NAME)-linux-amd64 ./main
	@echo "Building for macOS AMD64..."
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o dist/$(BINARY_NAME)-macos-amd64 ./main
	@echo "Building for macOS ARM64..."
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o dist/$(BINARY_NAME)-macos-arm64 ./main
	@echo "Building for Windows AMD64..."
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o dist/$(BINARY_NAME)-windows-amd64.exe ./main
	@echo "All builds completed successfully!"
	@ls -la dist/

ci: deps format lint test-coverage

all: ci build

clean: clean-dist
	@echo "Cleaning up..."
	go clean
	rm -f coverage.out coverage.html

clean-dist:
	@echo "Cleaning dist directory..."
	rm -rf dist/
