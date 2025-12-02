# Makefile

# Variables for development tools
# Tools are managed via go.mod and can be run using 'go run <package>'
GOIMPORTS := go tool goimports
GOLANGCI_LINT := go tool golangci-lint
GO_TEST_COVERAGE := go tool go-test-coverage
GOSEC := go tool gosec

# Build variables
BINARY_NAME := sargantana-go

# Build platforms (GOOS:GOARCH:output-name:extension)
PLATFORMS := \
	linux:amd64 \
	darwin:amd64 \
	darwin:arm64 \
	windows:amd64:.exe

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
.PHONY: all help install uninstall format test test-unit test-integration test-with-coverage \
	check-coverage bench lint security build build-all clean clean-dist deps ci

# Standard targets
.DEFAULT_GOAL := help

all: deps build ## Build the project with dependencies

help: ## Show this help message
	@echo "Sargantana Go - Available targets:"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ""

install: build ## Install binary to system (default: /usr/local/bin)
	@echo "Installing $(BINARY_NAME) to $(BINDIR)..."
	@$(INSTALL) -d $(BINDIR)
	@$(INSTALL) -m 755 bin/$(BINARY_NAME) $(BINDIR)/$(BINARY_NAME)
	@echo "Installation complete"

uninstall: ## Remove installed binary from system
	@echo "Uninstalling $(BINARY_NAME)..."
	@rm -f $(BINDIR)/$(BINARY_NAME)

# Dependency management
deps: ## Download and verify Go module dependencies
	@echo "Managing dependencies..."
	@go mod tidy
	@go mod verify
	@go mod download

# Testing
test: ## Run all tests
	go test ./...

test-unit: ## Run unit tests only
	go test -tags=unit ./...

test-integration: ## Run integration tests only
	go test -tags=integration ./...

test-with-coverage: ## Run tests with coverage report
	@echo "Running tests with coverage..."
	@go test -tags=unit,integration $(shell go list ./... | grep -v '/examples/') -covermode=atomic -coverprofile=coverage.out

check-coverage: test-with-coverage ## Verify coverage meets threshold requirements
	@$(GO_TEST_COVERAGE) --config=./.testcoverage.yml

bench: ## Run performance benchmarks
	go test -bench=. ./... -benchmem

# Code quality
format: ## Format code with go fmt and goimports
	@echo "Formatting code..."
	@go fmt ./...
	@$(GOIMPORTS) -w .

lint: format ## Run code quality checks (format, vet, golangci-lint)
	@echo "Linting code..."
	@go vet ./...
	@$(GOLANGCI_LINT) run ./...

security: ## Run security vulnerability scan with gosec
	@echo "Running security scan..."
	@$(GOSEC) -tests -severity=low -confidence=low -track-suppressions \
		-fmt=json -out=gosec-report.json -stdout -verbose=text ./...

# Building
build: ## Build the binary for current platform
	@mkdir -p bin
	@go build -v -ldflags="$(LDFLAGS)" -o bin/$(BINARY_NAME) ./cmd/sargantana

build-all: clean-dist ## Build binaries for all supported platforms
	@echo "Building for all platforms..."
	@mkdir -p dist
	@for platform in $(PLATFORMS); do \
		GOOS=$$(echo $$platform | cut -d: -f1); \
		GOARCH=$$(echo $$platform | cut -d: -f2); \
		EXT=$$(echo $$platform | cut -d: -f3); \
		GOOS=$$GOOS GOARCH=$$GOARCH CGO_ENABLED=0 go build \
			-ldflags="$(LDFLAGS)" \
			-o dist/$(BINARY_NAME)-$$GOOS-$$GOARCH$$EXT \
			./cmd/sargantana || exit 1; \
	done
	@ls -la dist/

# CI pipeline
ci: deps lint test-with-coverage check-coverage security ## Run complete CI pipeline

# Cleanup
clean: clean-dist ## Remove all build artifacts and generated files
	@go clean
	@rm -rf bin/
	@rm -f *.out gosec-report.json

clean-dist: ## Remove distribution builds
	@rm -rf dist/
