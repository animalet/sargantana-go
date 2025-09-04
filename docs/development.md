# Development Guide

This guide covers local development setup, compilation, and development workflows for Sargantana Go.

## Prerequisites

- Go 1.25 or later
- Make
- Git (to clone the repository)
- Docker and Docker Compose (required for running tests locally)

## Installation

```bash
git clone https://github.com/animalet/sargantana-go.git
cd sargantana-go
make configure
make build
```

## Compilation Instructions

If you prefer to compile from source or need to build for a different platform, you can compile the binaries yourself.

### Clone and Build

```bash
# Clone the repository
git clone https://github.com/animalet/sargantana-go.git
cd sargantana-go

# Set up development environment (installs tools and dependencies)
make configure

# Build for your current platform (outputs to bin/ directory)
make build

# Or build manually
go build -o bin/sargantana-go ./main
```

### Cross-Platform Compilation

Build binaries for all supported platforms:

```bash
# Using the Makefile (recommended) - outputs to dist/ directory
make build-all

# Manual cross-compilation examples (using actual ldflags from Makefile):
# Linux AMD64
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w -X main.version=dev" -o sargantana-go-linux-amd64 ./main

# macOS AMD64 (Intel)
GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w -X main.version=dev" -o sargantana-go-macos-amd64 ./main

# macOS ARM64 (Apple Silicon)
GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="-s -w -X main.version=dev" -o sargantana-go-macos-arm64 ./main

# Windows AMD64
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w -X main.version=dev" -o sargantana-go-windows-amd64.exe ./main
```

The compiled binaries will be placed in the `dist/` directory when using `make build-all`, or in the `bin/` directory when using `make build`.

### Build Flags Explained

- `CGO_ENABLED=0`: Disables CGO for static linking (creates standalone binaries)
- `-ldflags="-s -w -X main.version=dev"`: 
  - `-s`: Omit symbol table and debug information
  - `-w`: Omit DWARF debug information
  - `-X main.version=dev`: Set version information at build time

## Development Commands

```bash
# Set up development environment (install tools and dependencies)
make configure

# Install development tools only
make install-tools

# Manage dependencies
make deps

# Run tests
make test

# Run tests with coverage
make test-with-coverage

# Check test coverage against thresholds
make check-coverage

# Run benchmarks
make bench

# Format code (includes goimports)
make format

# Run linting (includes go vet and golangci-lint)
make lint

# Run all CI checks locally
make ci

# Build the application (outputs to bin/ directory)
make build

# Build for all platforms (outputs to dist/ directory)
make build-all

# Clean build artifacts
make clean

# Install binary to system (requires sudo on most systems)
make install

# Uninstall binary from system
make uninstall
```

## Project Structure

```
sargantana-go/
├── .github/            # GitHub workflows and configuration
├── .secrets/           # Local secrets directory (gitignored)
├── main/               # Main application entry point
├── server/             # Core server implementation
├── controller/         # Built-in controllers (auth, static, load balancer)
├── config/             # Configuration management
├── database/           # Database clients (Redis, Neo4j)
├── session/            # Session storage implementations
├── docs/               # Project documentation
├── examples/           # Example configuration files
├── certs/              # SSL certificates directory
├── bin/                # Built binaries (created by make build)
├── dist/               # Cross-platform binaries (created by make build-all)
├── docker-compose.yml  # Docker services for development/testing
├── .golangci.yml       # Linter configuration
├── .testcoverage.yml   # Test coverage configuration
├── config.local.yaml   # Local configuration file
└── Makefile            # Development commands
```

## Development Workflow

### 1. Setting up your development environment

1. Clone the repository and install dependencies
2. Start the required services using docker-compose (see [Testing Guide](testing.md))
3. Run tests to ensure everything is working
4. Start coding!

### 2. Making changes

1. Create a feature branch: `git checkout -b feature/amazing-feature`
2. Make your changes
3. Run tests: `make test`
4. Run linting: `make lint`
5. Format code: `make format`

### 3. Before committing

1. Run all CI checks locally: `make ci`
2. Ensure all tests pass
3. Commit your changes with a descriptive message
4. Push to your feature branch

### 4. Submitting changes

1. Open a Pull Request against the main branch
2. Ensure CI passes
3. Wait for code review

## IDE Setup

### VS Code

Recommended extensions:
- Go extension by Google
- Docker extension
- GitLens

### GoLand/IntelliJ

The project should work out of the box with GoLand. Make sure to:
1. Enable Go modules support
2. Set up the test configuration to use docker-compose services

## Debugging

### Running with debugger

```bash
# Run the main application in debug mode (requires a valid config file)
go run -race ./main -debug -config config.local.yaml

# Or build and run with debug symbols
make build
./bin/sargantana-go -debug -config config.yaml

# Show version information (no config required)
./bin/sargantana-go -version

# Show help
./bin/sargantana-go -help
```

### Command-line options

- `-debug`: Enable debug mode with verbose logging
- `-version`: Show version information
- `-config <path>`: **Required** - Specify path to configuration file
- `-help`: Show command-line help

**Note**: The `-config` flag is mandatory for running the server. Only `-version` and `-help` work without it.

### Configuration Requirements

Before running in debug mode, ensure you have:

1. An existing (and valid) YAML configuration file (e.g., `config.yaml`)
2. Proper Vault configuration or disable Vault in your config
3. Required services running if using Redis/Neo4j (see docker-compose.yml)

### Common debugging scenarios

1. **Authentication issues**: Check the authentication providers documentation
2. **Database connection issues**: Ensure docker-compose services are running
3. **Session issues**: Verify Redis is running if using Redis sessions
4. **Static file serving**: Check file paths and permissions

## Contributing Guidelines

1. Follow Go best practices and idioms
2. Write tests for new functionality
3. Update documentation when adding features
4. Use meaningful commit messages
5. Keep pull requests focused and small

See the main [Contributing](../README.md#contributing) section for more details.
