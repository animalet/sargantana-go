# Makefile

# Variables
TOOLS_BIN_DIR := $(shell go env GOPATH)/bin
GOIMPORTS := $(TOOLS_BIN_DIR)/goimports
GOLANGCI_LINT := $(TOOLS_BIN_DIR)/golangci-lint
GO_TEST_COVERAGE := $(TOOLS_BIN_DIR)/go-test-coverage

# Tasks
.PHONY: all format test clean lint mod-tidy test bench install-golangci-lint

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

mod-tidy:
	@echo "Tidying go.mod and go.sum..."
	go mod tidy
	go mod verify
	go mod download

bench:
	@echo "Running benchmarks..."
	go test -bench=. ./...

build:
	@echo "Building application..."
	go build -v -o bin/sargantana-go ./main

ci: mod-tidy format lint test

all: ci build

clean:
	@echo "Cleaning up..."
	go clean
	rm -f coverage.out coverage.html
