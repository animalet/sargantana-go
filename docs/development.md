# Development Guide

This guide covers local development setup, compilation, and development workflows for Sargantana Go.

## Prerequisites

- **Go 1.25** or later
- **Make**
- **Docker & Docker Compose** (for local testing)

## Quick Start

```bash
# 1. Clone the repository
git clone https://github.com/animalet/sargantana-go.git
cd sargantana-go

# 2. Setup development environment (installs tools)
make configure

# 3. Start dependency services (Redis, DBs, etc.)
docker-compose up -d

# 4. Run tests to verify everything is working
make test

# 5. Build the binary
make build
```

## Build Commands

The project uses a `Makefile` to simplify common tasks.

| Command | Description |
|---------|-------------|
| `make build` | Build the binary for the current platform (outputs to `bin/`) |
| `make build-all` | Cross-compile for Linux, macOS (Intel/Apple Silicon), and Windows (outputs to `dist/`) |
| `make clean` | Remove build artifacts |

### Cross-Compilation Details

`make build-all` generates binaries for:
- Linux AMD64
- macOS AMD64 (Intel)
- macOS ARM64 (Apple Silicon)
- Windows AMD64

The binaries are built with `CGO_ENABLED=0` for static linking and include version information (git tag/commit).

## Development Workflow

### 1. Code Quality Tools

We use several tools to maintain code quality, which are installed via `make configure`:
- **goimports**: Formats code and manages imports.
- **golangci-lint**: A fast, parallel linter runner.
- **gosec**: Security scanner for Go code.
- **go-test-coverage**: Enforces test coverage thresholds.

### 2. Common Tasks

```bash
# Format code
make format

# Lint code
make lint

# Run security scan
make security

# Run all CI checks locally (Lint + Test + Coverage + Security)
make ci
```

### 3. Running the Server Locally

To run the server locally during development:

```bash
# 1. Ensure services are running
docker-compose up -d

# 2. Run with a config file
go run ./cmd/sargantana -config config.yaml -debug
```

**Note**: You need a valid `config.yaml`. You can copy `examples/blog_example/config.yaml` as a starting point.

## Contributing

1.  Create a feature branch.
2.  Make your changes.
3.  Run `make ci` to ensure all checks pass.
4.  Submit a Pull Request.
