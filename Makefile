# Makefile

# Variables
TOOLS_BIN_DIR := $(shell go env GOPATH)/bin
GOIMPORTS := $(TOOLS_BIN_DIR)/goimports
GOLANGCI_LINT := $(TOOLS_BIN_DIR)/golangci-lint
GO_TEST_COVERAGE := $(TOOLS_BIN_DIR)/go-test-coverage

# Build variables
BINARY_NAME := sargantana-go

# Automatically detect version from git
VERSION ?= $(shell git describe --tags --exact-match 2>/dev/null || git describe --tags --abbrev=0 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# If we're not on a tagged commit, append the commit hash and mark as dirty if needed
ifneq ($(shell git describe --tags --exact-match 2>/dev/null),)
    # We're on a tagged commit, use the tag as-is
    BUILD_VERSION := $(VERSION)
else
    # Not on a tagged commit, append commit info
    BUILD_VERSION := $(VERSION)-$(COMMIT)
    ifneq ($(shell git status --porcelain 2>/dev/null),)
        BUILD_VERSION := $(BUILD_VERSION)-dirty
    endif
endif

LDFLAGS := -s -w -X main.version=$(BUILD_VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)

# Install variables
PREFIX ?= /usr/local
BINDIR := $(PREFIX)/bin
INSTALL := install

# Tasks
.PHONY: all configure install uninstall format test clean lint deps test bench build build-all test-with-coverage check-coverage ci clean-dist

# Standard targets
all: configure build

configure: deps install-tools
	@echo "Configuring development environment..."

install: build
	@echo "Installing $(BINARY_NAME) to $(BINDIR)..."
	$(INSTALL) -d $(BINDIR)
	$(INSTALL) -m 755 bin/$(BINARY_NAME) $(BINDIR)/$(BINARY_NAME)
	@echo "Installation complete. Run '$(BINARY_NAME)' to start the server."

uninstall:
	@echo "Uninstalling $(BINARY_NAME) from $(BINDIR)..."
	rm -f $(BINDIR)/$(BINARY_NAME)
	@echo "Uninstallation complete."

# Development tools installation
install-tools: install-goimports install-golangci-lint install-go-test-with-coverage
	@echo "Development tools installed."

install-goimports:
	@if ! command -v goimports &> /dev/null; then \
		echo "Installing goimports..."; \
		go install golang.org/x/tools/cmd/goimports@latest; \
	fi

install-golangci-lint:
	@echo "Installing/updating golangci-lint to latest version..."; \
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin latest

install-go-test-with-coverage:
	@if ! command -v go-test-with-coverage &> /dev/null; then \
		echo "Installing go-test-with-coverage..."; \
		go install github.com/vladopajic/go-test-coverage/v2@latest; \
	fi

# Dependency management
deps:
	@echo "Tidying go.mod and go.sum..."
	go mod tidy
	go mod verify
	go mod download

# Testing
test:
	@echo "Running tests..."
	go test ./...

test-with-coverage:
	@echo "Running tests with coverage..."
	go test -tags=unit,integration $(shell go list ./... | grep -v '/examples/') -covermode=atomic -coverprofile=coverage.out

test-unit:
	@echo "Running unit tests..."
	go test -tags=unit ./...

test-integration:
	@echo "Running integration tests..."
	go test -tags=integration ./...

check-coverage: test-with-coverage install-go-test-with-coverage
	$(GO_TEST_COVERAGE) --config=./.testcoverage.yml

bench:
	@echo "Running benchmarks..."
	go test -bench=. ./... -benchmem

# Code quality
format: install-goimports
	@echo "Formatting code..."
	go fmt ./... && $(GOIMPORTS) -w .

lint: format install-golangci-lint
	@echo "Linting code..."
	go vet ./...
	$(GOLANGCI_LINT) run ./...

# Building
build:
	@echo "Building application..."
	@mkdir -p bin
	go build -v -ldflags="$(LDFLAGS)" -o bin/$(BINARY_NAME) ./cmd/sargantana

# Build for all platforms
build-all: clean-dist
	@echo "Building for all platforms..."
	@mkdir -p dist
	@echo "Building for Linux AMD64..."
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o dist/$(BINARY_NAME)-linux-amd64 ./cmd/sargantana
	@echo "Building for macOS AMD64..."
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o dist/$(BINARY_NAME)-macos-amd64 ./cmd/sargantana
	@echo "Building for macOS ARM64..."
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o dist/$(BINARY_NAME)-macos-arm64 ./cmd/sargantana
	@echo "Building for Windows AMD64..."
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o dist/$(BINARY_NAME)-windows-amd64.exe ./cmd/sargantana
	@echo "All builds completed successfully!"
	@ls -la dist/

# CI pipeline
ci: deps test-with-coverage lint

# Cleanup
clean: clean-dist
	@echo "Cleaning up..."
	go clean
	rm -rf bin/ certs/
	rm -f *.out

clean-dist:
	@echo "Cleaning dist directory..."
	rm -rf dist/
