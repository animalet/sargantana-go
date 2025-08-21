# Makefile

# Variables
FRONTEND_DIR := frontend
PROJECT_NAME := $(shell basename $(CURDIR))

# Tasks
.PHONY: all format test clean lint mod-tidy docs test bench

test:
	@echo "Running backend tests..."
	go test ./...

test-coverage:
	@echo "Running backend tests with coverage..."
	go test -covermode=atomic -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

install-tools:
	go install golang.org/x/tools/cmd/goimports@latest

TOOLS_BIN_DIR := $(shell go env GOPATH)/bin
GOIMPORTS := $(TOOLS_BIN_DIR)/goimports

format: install-tools
	@echo "Formatting code..."
	go fmt ./... && $(GOIMPORTS) -w .

lint: format
	@echo "Linting code..."
	go vet ./...
	# golangci-lint run ./... # Uncomment if you use golangci-lint

mod-tidy:
	@echo "Tidying go.mod and go.sum..."
	go mod tidy

docs:
	@echo "Generating documentation..."
	# godoc -http=:6060 # or use another doc tool

bench:
	@echo "Running benchmarks..."
	go test -bench=. ./...

all: mod-tidy format lint test

clean:
	@echo "Cleaning up..."
	rm -rf ./bin
