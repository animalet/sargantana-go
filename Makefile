# Makefile

# Variables for development tools
# Tools are managed via go.mod and can be run using 'go run <package>'
GOIMPORTS := go run golang.org/x/tools/cmd/goimports
GOLANGCI_LINT := go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint
GO_TEST_COVERAGE := go run github.com/vladopajic/go-test-coverage/v2
GOSEC := go run github.com/securego/gosec/v2/cmd/gosec

# Development tools list (extracted from tools.go)
TOOL_PACKAGES := \
	github.com/golangci/golangci-lint/v2/cmd/golangci-lint \
	github.com/securego/gosec/v2/cmd/gosec \
	github.com/vladopajic/go-test-coverage/v2 \
	golang.org/x/tools/cmd/goimports

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

configure: deps
	@echo "Configuring development environment..."
	@echo "Tools are managed via go.mod and will be downloaded on first use."

install: build
	@echo "Installing $(BINARY_NAME) to $(BINDIR)..."
	$(INSTALL) -d $(BINDIR)
	$(INSTALL) -m 755 bin/$(BINARY_NAME) $(BINDIR)/$(BINARY_NAME)
	@echo "Installation complete. Run '$(BINARY_NAME)' to start the server."

uninstall:
	@echo "Uninstalling $(BINARY_NAME) from $(BINDIR)..."
	rm -f $(BINDIR)/$(BINARY_NAME)
	@echo "Uninstallation complete."

# Development tools installation (optional - installs to GOPATH/bin for faster execution)
# By default, tools are run via 'go run' which uses the module cache
install-tools:
	@echo "Installing development tools to GOPATH/bin..."
	@for tool in $(TOOL_PACKAGES); do \
		echo "  → Installing $$tool"; \
		go install $$tool; \
	done
	@echo "Development tools installed to $(shell go env GOPATH)/bin"
	@echo "Note: 'make' targets use 'go run' by default, which doesn't require installation"

list-tools:
	@echo "Development tools (from tools.go):"
	@for tool in $(TOOL_PACKAGES); do \
		echo "  • $$tool"; \
	done
	@echo ""
	@echo "Tools are run via 'go run' and don't require separate installation."
	@echo "Run 'make install-tools' to optionally install them to GOPATH/bin for faster execution."

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
ci: configure lint test-with-coverage check-coverage security

# Cleanup
clean: clean-dist
	@echo "Cleaning up..."
	go clean
	rm -rf bin/
	rm -f *.out gosec-report.json

clean-dist:
	@echo "Cleaning dist directory..."
	rm -rf dist/
