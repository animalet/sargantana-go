# Makefile

# Variables
FRONTEND_DIR := frontend
PROJECT_NAME := $(shell basename $(CURDIR))

# Tasks
.PHONY: all format test clean lint mod-tidy docs test bench install-golangci-lint

test:
	@echo "Running backend tests..."
	go test ./...

test-coverage:
	@echo "Running backend tests with coverage..."
	go test -covermode=atomic -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

install-tools:
	go install golang.org/x/tools/cmd/goimports@latest

install-golangci-lint:
	@if ! command -v golangci-lint &> /dev/null; then \
		echo "Installing golangci-lint..."; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin v1.55.2; \
	fi

TOOLS_BIN_DIR := $(shell go env GOPATH)/bin
GOIMPORTS := $(TOOLS_BIN_DIR)/goimports

format: install-tools
	@echo "Formatting code..."
	go fmt ./... && $(GOIMPORTS) -w .

lint: format install-golangci-lint
	@echo "Linting code..."
	go vet ./...
	golangci-lint run ./...

security:
	@echo "Running security checks..."
	@if ! command -v gosec &> /dev/null; then \
		echo "Installing gosec..."; \
		go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest; \
	fi
	gosec ./...

mod-tidy:
	@echo "Tidying go.mod and go.sum..."
	go mod tidy

docs:
	@echo "Generating documentation..."
	# godoc -http=:6060 # or use another doc tool

bench:
	@echo "Running benchmarks..."
	go test -bench=. ./...

build:
	@echo "Building application..."
	go build -v -o bin/sargantana-go ./main

ci: mod-tidy format lint security test

all: ci build

clean:
	@echo "Cleaning up..."
	rm -rf ./bin
	rm -f coverage.out coverage.html
