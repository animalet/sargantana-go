# Sargantana Go

[![CI](https://github.com/animalet/sargantana-go/workflows/CI/badge.svg)](https://github.com/animalet/sargantana-go/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/animalet/sargantana-go/branch/main/graph/badge.svg)](https://codecov.io/gh/animalet/sargantana-go)
[![Coverage Status](https://coveralls.io/repos/github/animalet/sargantana-go/badge.svg?branch=main)](https://coveralls.io/github/animalet/sargantana-go?branch=main)
[![Go Report Card](https://goreportcard.com/badge/github.com/animalet/sargantana-go)](https://goreportcard.com/report/github.com/animalet/sargantana-go)
[![Go Reference](https://pkg.go.dev/badge/github.com/animalet/sargantana-go.svg)](https://pkg.go.dev/github.com/animalet/sargantana-go)
[![License](https://img.shields.io/github/license/animalet/sargantana-go)](LICENSE)

## What is this?
I needed to build a performant web application but I wanted it to be simple to maintain and provide support for regular real life scenarios like OAuth2, modern frontend or database... So I came up with this.

## Features
- Web server using [Gin](https://github.com/gin-gonic/gin)
- OAuth2 support via [Goth: Multi-Provider Authentication for Go](https://github.com/markbates/goth)
- Out of the box OAuth2 user authentication flow implemented in the backend.
- Extensible and dead simple controller mechanism with session support via Redis or cookies, you choose your preferred flavour.
- Easy configuration for production.
- The full stack can be run locally using Docker Compose. It also has proper secrets management via [Docker Compose secrets](https://docs.docker.com/compose/how-tos/use-secrets/). This means that the Dockerfile is pretty much ready for production too.

## Quick Start

### Prerequisites
- Go 1.25.0 or later
- Make

### Installation
```bash
git clone https://github.com/animalet/sargantana-go.git
cd sargantana-go
make all
```

### Development
```bash
# Run tests
make test

# Run tests with coverage
make test-coverage

# Run linting
make lint

# Run security checks
make security

# Format code
make format

# Run all CI checks locally
make ci

# Build the application
make build
```

## CI/CD

This project uses GitHub Actions for continuous integration with the following features:

- **Automated Testing**: Tests run on multiple platforms (Ubuntu, Windows, macOS)
- **Code Coverage**: Coverage reports are automatically generated and uploaded to Codecov and Coveralls
- **Linting**: Code quality checks using golangci-lint with comprehensive rules
- **Security Scanning**: Automated security vulnerability scanning with Gosec
- **Dependency Management**: Automated dependency updates via Dependabot
- **Build Artifacts**: Compiled binaries are generated and stored as artifacts

### Coverage Reports
- [Codecov Report](https://codecov.io/gh/animalet/sargantana-go)
- [Coveralls Report](https://coveralls.io/github/animalet/sargantana-go)

## To do list (in priority order)
- Add more database support (Postgres, MySQL, MongoDB, etc).

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes and ensure all tests pass (`make ci`)
4. Commit your changes (`git commit -m 'Add some amazing feature'`)
5. Push to the branch (`git push origin feature/amazing-feature`)
6. Open a Pull Request

## License

This project is licensed under the terms specified in the [LICENSE](LICENSE) file.
