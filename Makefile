# Makefile

# Variables
TOOLS_BIN_DIR := $(shell go env GOPATH)/bin
TOOL_DIR := tools
GOIMPORTS := $(TOOLS_BIN_DIR)/goimports
GOLANGCI_LINT := $(TOOLS_BIN_DIR)/golangci-lint
GO_TEST_COVERAGE := $(TOOLS_BIN_DIR)/go-test-coverage
GOSEC := $(TOOLS_BIN_DIR)/gosec

# Development tools (automatically extracted from tools/main.go with versions from tools/go.mod)
TOOLS := $(shell cd $(TOOL_DIR) && go list -e -f '{{join .Imports "\n"}}' -tags tools | while read pkg; do echo "$$pkg@$$(go list -m -f '{{.Version}}' $$pkg 2>/dev/null || echo latest)"; done)

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
.PHONY: all configure install uninstall format test clean lint deps test bench build build-all test-with-coverage check-coverage security ci clean-dist list-tools

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
# Tools are automatically discovered from tools/main.go and installed with versions from tools/go.mod
install-tools:
	@echo "Installing development tools from $(TOOL_DIR)/go.mod..."
	@cd $(TOOL_DIR) && for tool in $(TOOLS); do \
		echo "  → Installing $$tool"; \
		go install $$tool; \
	done
	@echo "Development tools installed."

list-tools:
	@echo "Development tools (auto-discovered from $(TOOL_DIR)/main.go):"
	@for tool in $(TOOLS); do \
		echo "  • $$tool"; \
	done

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

check-coverage: test-with-coverage
	@$(GO_TEST_COVERAGE) --config=./.testcoverage.yml

bench:
	@echo "Running benchmarks..."
	go test -bench=. ./... -benchmem

# Code quality
format:
	@echo "Formatting code..."
	@go fmt ./...
	@$(GOIMPORTS) -w .

lint: format
	@echo "Linting code..."
	@go vet ./...
	@$(GOLANGCI_LINT) run ./...

security:
	@echo "Running security scan with gosec..."
	@echo "  • Test files included"
	@echo "  • Tracking suppression comments"
	@echo "  • Respecting #nosec directives"
	@$(GOSEC) \
		-tests \
		-enable-audit \
		-severity=low \
		-confidence=low \
		-track-suppressions \
		-fmt=json \
		-out=gosec-report.json \
		-stdout \
		-verbose=text \
		./...

# Building
build:
	@echo "Building application..."
	@mkdir -p bin
	go build -v -ldflags="$(LDFLAGS)" -o bin/$(BINARY_NAME) ./cmd/sargantana

# Build for all platforms
build-all: clean-dist
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
	@echo "All builds completed successfully!"
	@ls -la dist/

# CI pipeline
ci: configure test-with-coverage lint security

# Cleanup
clean: clean-dist
	@echo "Cleaning up..."
	go clean
	rm -rf bin/
	rm -f *.out gosec-report.json

clean-dist:
	@echo "Cleaning dist directory..."
	rm -rf dist/
