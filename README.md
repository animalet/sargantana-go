# Sargantana Go

<img src="logo.png" alt="Sargantana Go Logo" width="500"/>

```
      🦎 Fast • Flexible • Full-Stack Go Web Framework
```

[![CI](https://github.com/animalet/sargantana-go/workflows/CI/badge.svg)](https://github.com/animalet/sargantana-go/actions/workflows/ci.yml)
[![coverage](https://raw.githubusercontent.com/animalet/sargantana-go/badges/.badges/main/coverage.svg)](https://github.com/animalet/sargantana-go)
[![Go Report Card](https://goreportcard.com/badge/github.com/animalet/sargantana-go)](https://goreportcard.com/report/github.com/animalet/sargantana-go)
[![Go Reference](https://pkg.go.dev/badge/github.com/animalet/sargantana-go.svg)](https://pkg.go.dev/github.com/animalet/sargantana-go)
[![Release](https://img.shields.io/github/v/release/animalet/sargantana-go?include_prereleases&label=release&color=blue)](https://github.com/animalet/sargantana-go/releases/latest)
[![License](https://img.shields.io/github/license/animalet/sargantana-go)](LICENSE)

## What is this?

Sargantana Go is a modular configuration-driven web framework built on [Gin](https://github.com/gin-gonic/gin) that provides:

**Configuration System with Secret Management**
- YAML-based configuration with environment variable expansion
- Pluggable `SecretLoader` system (env, file, Vault, AWS Secrets Manager)
- Type-safe configuration loading with validation via `Validatable[T]`

**Modular Web Server Architecture**
- Controller-based system where each controller type can have multiple instances
- Built-in controllers: OAuth authentication (via Goth), static file serving, template rendering, load balancing
- Easy to extend with custom controllers
- Graceful shutdown with cleanup hooks

**Data Source Integration**
- **Databases**: PostgreSQL (pgxpool), Redis, MongoDB, Memcached
- **Secret Management**: HashiCorp Vault, AWS Secrets Manager, file-based secrets, environment variables
- All use the `ClientFactory[T]` pattern for type-safe, validated client creation

**Flexible Session Management**
- Five session storage backends: Cookie, Redis, PostgreSQL, MongoDB, Memcached
- Inject custom session stores via `SetSessionStore()`
- All integrate seamlessly with Gin sessions middleware

**The key differentiator** is the tight integration between configuration, secret management, and the web framework. It's a **"batteries-included but swappable"** framework - you get sensible defaults and common integrations out of the box, but every piece is designed to be replaceable or extended.

### Personal Learning Project

This is a personal side project I created for my own learning and practicing with both Go and vibe coding (AI-assisted development). While it's functional and includes comprehensive tests, it's primarily an educational endeavor to explore Go's ecosystem, web framework patterns, and modern development workflows. Feel free to use it, learn from it, or contribute to it!

## Features

- **Web Server**: High-performance HTTP server using [Gin](https://github.com/gin-gonic/gin)
- **Authentication**: 
    - Multi-provider authentication support via [Goth](https://github.com/markbates/goth) with 50+ providers.
    - Customizable `Authenticator` interface for any auth strategy (JWT, API Keys, etc.).
- **Session Management**: Five flexible session storage backends (Cookie, Redis, PostgreSQL, MongoDB, Memcached).
- **Static File Serving**: Built-in static file and template serving capabilities.
- **Load Balancing**: Round-robin load balancer with optional authentication and header filtering.
- **Database Support**: PostgreSQL, Redis, MongoDB, and Memcached integration.
- **Configuration**: YAML-based configuration with pluggable secret management.
- **Extensibility**: Custom controllers with dependency injection via Constructor pattern.

## Documentation

- [Authentication Providers](docs/authentication-providers.md): Configure OAuth2 providers and custom authenticators.
- [Secret Providers](docs/secret_providers.md): Use Vault, AWS Secrets Manager, and other secret sources.
- [Configuration Guide](docs/configuration.md): Learn about the type-safe, modular configuration system.
- [Configuration Immutability](docs/immutability.md): Understand how the framework ensures runtime configuration safety.
- [Testing Guide](docs/testing.md): Run tests locally and understand CI workflows.
- [Development Guide](docs/development.md): Build, compile, and contribute to the project.

## Quick Start

### Prerequisites

- Go 1.25 or later
- Make (for development)
- A configuration file (required for running the server)

### Installation

```bash
go get github.com/animalet/sargantana-go
```

### Basic Usage

Create a simple web application with YAML configuration:

```go
package main

import (
    "github.com/animalet/sargantana-go/pkg/config"
    "github.com/animalet/sargantana-go/pkg/controller"
    "github.com/animalet/sargantana-go/pkg/server"
    "log"
)

func main() {
    // 1. Read configuration
    cfg, err := config.ReadModular("config.yaml")
    if err != nil {
        log.Fatal(err)
    }

    // 2. Register available controller types
    server.RegisterController("auth", controller.NewAuthController)
    server.RegisterController("static", controller.NewStaticController)

    // 3. Create server
    // Note: You need to extract the server config from the modular config
    serverCfg, err := config.Get[server.SargantanaConfig](cfg, "server")
    if err != nil {
        log.Fatal(err)
    }
    
    sargantana := server.NewServer(*serverCfg)

    // 4. Start server
    if err := sargantana.StartAndWaitForSignal(); err != nil {
        log.Fatal(err)
    }
}
```

### Examples

Check out the [Blog Example](examples/blog_example/README.md) for a complete, production-ready application demonstrating:
- **Authentication** with Keycloak (OAuth2/OIDC)
- **Database** integration with PostgreSQL
- **Secrets Management** with Vault and Files
- **Session Management** with Redis

### API Gateway with Load Balancing

```yaml
# config.yaml
server:
  address: "0.0.0.0:8080"
  session_name: "gateway"
  session_secret: "${SESSION_SECRET}"

controllers:
  - type: "auth"
    config:
      providers:
        github:
          key: "${GITHUB_KEY}"
          secret: "${GITHUB_SECRET}"

  - type: "load_balancer"
    config:
      path: "/api"
      require_auth: true
      endpoints:
        - "http://api1:8080"
        - "http://api2:8080"
        - "http://api3:8080"
```

## Production Deployment

### Docker Compose Example

```yaml
version: '3.8'

services:
  app:
    build: .
    ports:
      - "8080:8080"
    environment:
      - GIN_MODE=release
    secrets:
      - SESSION_SECRET
      - GITHUB_KEY
      - GITHUB_SECRET
    volumes:
      - ./config.yaml:/app/config.yaml:ro
    command: [
      "/app/sargantana-go",
      "-config", "/app/config.yaml"
    ]
    depends_on:
      - redis

  redis:
    image: redis:7-alpine
    volumes:
      - redis_data:/data

secrets:
  SESSION_SECRET:
    file: ./secrets/session_secret
  GITHUB_KEY:
    file: ./secrets/github_key
  GITHUB_SECRET:
    file: ./secrets/github_secret

volumes:
  redis_data:
```

### Dockerfile

```dockerfile
FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o sargantana-go main/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /app/sargantana-go .
COPY --from=builder /app/frontend ./frontend
COPY --from=builder /app/templates ./templates
CMD ["./sargantana-go", "-config", "config.yaml"]
```

## Development

For detailed development setup, compilation instructions, and workflows, see
the [Development Guide](docs/development.md).

For testing setup, running tests locally, and understanding the test infrastructure, see
the [Testing Guide](docs/testing.md).

### Quick Start for Developers

```bash
# Clone and setup
git clone https://github.com/animalet/sargantana-go.git
cd sargantana-go

# Start test services (required for integration tests)
docker-compose up -d

# Run unit tests
make test-unit

# Run integration tests
make test-integration

# Run all tests
make test

# Build the project
make build
```
