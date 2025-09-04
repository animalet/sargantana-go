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
make all
```

## Compilation Instructions

If you prefer to compile from source or need to build for a different platform, you can compile the binaries yourself.

### Clone and Build

```bash
# Clone the repository
git clone https://github.com/animalet/sargantana-go.git
cd sargantana-go

# Build for your current platform
go build -o sargantana-go ./main

# Or use the Makefile
make build
```

### Cross-Platform Compilation

Build binaries for all supported platforms:

```bash
# Using the Makefile (recommended)
make build-all

# Manual cross-compilation examples:
# Linux AMD64
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o sargantana-go-linux-amd64 ./main

# macOS AMD64 (Intel)
GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o sargantana-go-macos-amd64 ./main

# macOS ARM64 (Apple Silicon)
GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="-s -w" -o sargantana-go-macos-arm64 ./main

# Windows AMD64
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o sargantana-go-windows-amd64.exe ./main
```

The compiled binaries will be placed in the `dist/` directory when using `make build-all`, or in the current directory when building manually.

### Build Flags Explained

- `CGO_ENABLED=0`: Disables CGO for static linking (creates standalone binaries)
- `-ldflags="-s -w"`: Strips debug information to reduce binary size
  - `-s`: Omit symbol table and debug information
  - `-w`: Omit DWARF debug information

## Development Commands

```bash
# Run tests (requires docker-compose services)
make test

# Run tests with coverage
make test-with-coverage

# Run linting
make lint

# Format code
make format

# Run all CI checks locally (requires docker-compose)
make ci

# Clean build artifacts
make clean

# Build the basic server binary
make build
```

## Project Structure

```
sargantana-go/
├── main/           # Main application entry point
├── server/         # Core server implementation
├── controller/     # Built-in controllers (auth, static, load balancer)
├── config/         # Configuration management
├── database/       # Database clients (Redis, Neo4j)
├── logger/         # Logging utilities
├── session/        # Session storage implementations
├── docs/           # Project documentation
├── examples/       # Example configuration files
└── Makefile        # Development commands
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
# Run the main application in debug mode
go run -race ./main -debug

# Or build and run with debug symbols
go build -o sargantana-go ./main
./sargantana-go -debug
```

### Common debugging scenarios

1. **Authentication issues**: Check the authentication providers documentation
2. **Database connection issues**: Ensure docker-compose services are running
3. **Session issues**: Verify Redis is running if using Redis sessions
4. **Static file serving**: Check file paths and permissions

## Performance Profiling

Sargantana Go includes built-in profiling support when running in debug mode:

```bash
# Run with profiling enabled
./sargantana-go -debug

# Access profiling endpoints
curl http://localhost:8080/debug/pprof/
```

## Contributing Guidelines

1. Follow Go best practices and idioms
2. Write tests for new functionality
3. Update documentation when adding features
4. Use meaningful commit messages
5. Keep pull requests focused and small

See the main [Contributing](../README.md#contributing) section for more details.
