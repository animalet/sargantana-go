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

** Configuration System with Secret Management**
- YAML-based configuration with environment variable expansion
- Pluggable secret loaders implementing `SecretLoader` interface (env, file, Vault, AWS Secrets Manager)
- Type-safe configuration loading with validation via `Validatable[T]`, also datasource creation from those configurations via `ClientFactory[T]`

** Modular Web Server Architecture**
- Controller-based system where each controller type can have multiple instances
- Built-in controllers: OAuth authentication (via Goth), static file serving, template rendering, load balancing
- Easy to extend with custom controllers
- Graceful shutdown with cleanup hooks

** Data Source Integration**
- **Databases**: PostgreSQL (pgxpool), Redis, MongoDB, Memcached
- **Secret Management**: HashiCorp Vault, AWS Secrets Manager, file-based secrets, environment variables
- All use the `ClientFactory[T]` pattern for type-safe, validated client creation

** Flexible Session Management**
- Five session storage backends: Cookie, Redis, PostgreSQL, MongoDB, Memcached
- Inject custom session stores via `SetSessionStore()`
- All integrate seamlessly with Gin sessions middleware

**The key differentiator** is the tight integration between configuration, secret management, and the web framework - allowing you to build highly customized web applications where secrets are resolved at runtime from multiple sources, controllers are dynamically registered and configured, and database clients are created with validated, type-safe configs. It's a **"batteries-included but swappable"** framework - you get sensible defaults and common integrations out of the box, but every piece is designed to be replaceable or extended.

### Personal Learning Project

This is a personal side project I created for my own learning and practicing with both Go and vibe coding (AI-assisted development). While it's functional and includes comprehensive tests, it's primarily an educational endeavor to explore Go's ecosystem, web framework patterns, and modern development workflows. Feel free to use it, learn from it, or contribute to it!

## Features

- **Web Server**: High-performance HTTP server using [Gin](https://github.com/gin-gonic/gin)
- **Authentication**: Multi-provider authentication support via [Goth](https://github.com/markbates/goth) with 45+ providers (Google, GitHub, Keycloak, and more)
- **Session Management**: Five flexible session storage backends (Cookie, Redis, PostgreSQL, MongoDB, Memcached). Any session store supported by [gin-contrib/sessions](https://github.com/gin-contrib/sessions) can be used
- **Static File Serving**: Built-in static file and template serving capabilities
- **Load Balancing**: Round-robin load balancer with optional authentication and header filtering
- **Database Support**: PostgreSQL, Redis, MongoDB, and Memcached integration with type-safe client factory pattern
- **Configuration**: YAML-based configuration with pluggable secret management (environment variables, files, Vault, AWS Secrets Manager)
- **Extensibility**: Custom controllers with dependency injection via Constructor pattern

## Quick Start

### Prerequisites

- Go 1.25 or later
- Make (for development)
- A configuration file (required for running the server)

### Binary Distribution

Pre-built binaries are available for multiple platforms. Download the appropriate binary for your operating system from
the [releases page](https://github.com/animalet/sargantana-go/releases/latest).

#### Available Platforms

- **Linux AMD64**: `sargantana-go-linux-amd64`
- **macOS AMD64**: `sargantana-go-macos-amd64` (Intel-based Macs)
- **macOS ARM64**: `sargantana-go-macos-arm64` (Apple Silicon Macs)
- **Windows AMD64**: `sargantana-go-windows-amd64.exe`

### Quick Start with Binary

1. Download the binary for your platform from the releases page
2. Make it executable (Linux/macOS only):
   ```bash
   chmod +x sargantana-go-*
   ```
3. Create a configuration file (see Configuration section below)
4. Run it:
   ```bash
   # Linux/macOS
   ./sargantana-go-linux-amd64 -config config.yaml -debug
   
   # Windows
   sargantana-go-windows-amd64.exe -config config.yaml -debug
   ```

### Installation from Source

```bash
go get github.com/animalet/sargantana-go
```

### Basic usage

#### Create a simple web application with YAML configuration

```go
package main

import (
    "github.com/animalet/sargantana-go/pkg/controller"
    "github.com/animalet/sargantana-go/pkg/server"
)

func main() {
    // Register available controller types
    server.AddControllerType("auth", controller.NewAuthController)
    server.AddControllerType("mycontroller", mycontrollers.NewMyWebappController)

    // Create server from YAML configuration
    sargantana, err := server.NewServer("config.yaml")
    if err != nil {
        panic(err)
    }

    // Start server and wait for shutdown signal
    err = sargantana.StartAndWaitForSignal()
    if err != nil {
        panic(err)
    }
}
```

#### Create a simple web application with programmatic configuration

```go
package main

import (
    "github.com/animalet/sargantana-go/pkg/config"
    "github.com/animalet/sargantana-go/pkg/controller"
    "github.com/animalet/sargantana-go/pkg/server"
)

func main() {
    // Define configuration programmatically
    // Note: Programmatic configuration is currently best supported by creating a temporary config file
    // or by manually constructing the Server struct, as NewServer expects a file path.
    // However, for advanced usage, you can use config.ReadModular to load from a file
    // and then modify the returned *config.Config struct before passing it to components.
    
    // Register controllers and create server
    server.AddControllerType("auth", controller.NewAuthController)
    server.AddControllerType("static", controller.NewStaticController)
}
```

### Running the Application

```bash
# Basic server with configuration file
./sargantana-go -config config.yaml

# With debug mode enabled
./sargantana-go -config config.yaml -debug

# Display version information
./sargantana-go -version
```

## Configuration

Sargantana Go uses YAML configuration files for setup. The configuration is divided into several sections:

### Basic Configuration Structure

```yaml
server:
  address: "localhost:8080"
  session_name: "myapp"
  session_secret: "${SESSION_SECRET}"
  secrets_dir: "/run/secrets"  # Optional
  redis_session_store:         # Optional
    address: "localhost:6379"
    database: 0

vault:  # Optional
  address: "https://vault.example.com:8200"
  token: "${VAULT_TOKEN}"
  path: "secret/data/myapp"

controllers:
  - type: "static"
    config:
      statics_dir: "./public"
      templates_dir: "./templates"
  - type: "auth"
    config:
      providers:
        github:
          key: "${GITHUB_KEY}"
          secret: "${GITHUB_SECRET}"
```

### Command Line Options

| Flag       | Description                    | Default | Example                     |
|------------|--------------------------------|---------|-----------------------------|
| `-config`  | Path to configuration file     | None    | `-config config.yaml`      |
| `-debug`   | Enable debug mode              | `false` | `-debug`                    |
| `-version` | Show version information       | `false` | `-version`                  |

### Environment Variables

Configuration values can reference environment variables using `${VAR_NAME}` syntax:

```bash
# Session security
export SESSION_SECRET="your-session-secret-key"

# Authentication Providers
export GITHUB_KEY="your-github-client-id"
export GITHUB_SECRET="your-github-client-secret"
export GOOGLE_KEY="your-google-client-id"
export GOOGLE_SECRET="your-google-client-secret"

# Vault integration
export VAULT_TOKEN="your-vault-token"
```

### Docker Secrets

You can also use Docker secrets by placing secret files in a directory and configuring the `secrets_dir`:

```yaml
server:
  secrets_dir: "/run/secrets"
```

```bash
# Directory structure
/run/secrets/
├── SESSION_SECRET
├── GITHUB_KEY
├── GITHUB_SECRET
└── GOOGLE_KEY
```

### Vault Integration

For advanced secret management, configure Vault integration:

```yaml
vault:
  address: "https://vault.example.com:8200"
  token: "${VAULT_TOKEN}"
  path: "secret/data/myapp"
  namespace: "my-namespace"  # Optional, for Enterprise

# Use vault secrets in configuration
server:
  session_secret: "vault:session-secret"
```

See the [Vault Secrets Documentation](docs/vault-secrets.md) for detailed configuration options.

### AWS Secrets Manager Integration

For AWS-based secret management:

```yaml
aws:
  region: "us-east-1"
  access_key_id: "${AWS_ACCESS_KEY_ID}"      # Optional: uses IAM role if omitted
  secret_access_key: "${AWS_SECRET_ACCESS_KEY}"  # Optional: uses IAM role if omitted
  endpoint: ""                                # Optional: for LocalStack testing
  secret_name: "myapp/secrets"

# Use AWS secrets in configuration
server:
  session_secret: "aws:session-secret"
```

## Controllers

Sargantana Go uses a controller-based architecture. Each controller handles a specific aspect of your application and is configured in the YAML file.

### Static Controller

Serves static files and HTML templates:

```yaml
controllers:
  - type: "static"
    name: "static-files"
    config:
      statics_dir: "./public"
      templates_dir: "./templates"
```

Features:

- Serves files from `/static/*` route
- Serves `index.html` at root `/`
- Loads HTML templates with `{{ }}` syntax
- Automatic template discovery

### Auth Controller

Provides authentication with 50+ providers:

```yaml
controllers:
  - type: "auth"
    name: "authentication"
    config:
      callback_host: "https://myapp.example.com"  # Optional
      providers:
        github:
          key: "${GITHUB_KEY}"
          secret: "${GITHUB_SECRET}"
          scopes:
            - "read:user"
            - "user:email"
        google:
          key: "${GOOGLE_KEY}"
          secret: "${GOOGLE_SECRET}"
```

**Supported Authentication Providers:**

For the complete list of 50+ supported providers, configuration details, and provider IDs, see
the [Authentication Providers Documentation](docs/authentication-providers.md).

**Authentication Flow:**

1. Visit `/auth/{provider}` to start authentication flow
2. User redirects to provider for authentication
3. Provider redirects back to `/auth/{provider}/callback`
4. User session is created automatically
5. Access user info via session in your handlers

**Protected Routes:**

```go
// Use the LoginFunc middleware for protected routes
engine.GET("/protected", controller.LoginFunc, func(c *gin.Context) {
    session := sessions.Default(c)
    user := session.Get("user").(controller.UserObject)
    c.JSON(200, gin.H{"user": user.User.Name})
})
```

### Load Balancer Controller

Round-robin load balancer for backend services:

```yaml
controllers:
  - type: "load_balancer"
    name: "api-proxy"
    config:
      path: "/api"
      require_auth: true
      endpoints:
        - "http://api1:8080"
        - "http://api2:8080"
        - "http://api3:8080"
```

Features:

- Round-robin load balancing
- Optional authentication requirement
- Support for all HTTP methods
- Automatic failover
- Request forwarding with headers

## Session Management

### Cookie-based Sessions (Default)

```yaml
server:
  session_name: "myapp"
  session_secret: "${SESSION_SECRET}"
```

### Redis Sessions

For distributed session storage:

```yaml
server:
  session_name: "myapp"
  session_secret: "${SESSION_SECRET}"

redis:
  address: "localhost:6379"
  database: 0
  max_idle: 10
  idle_timeout: 240s
```

```go
// In your application startup
redisCfg, err := config.Get[database.RedisConfig](cfg, "redis")
pool, err := redisCfg.CreateClient()
store, err := session.NewRedisSessionStore(debugMode, []byte(serverCfg.SessionSecret), pool)
sargantana.SetSessionStore(&store)
```

### PostgreSQL, MongoDB, or Memcached Sessions

Similarly, you can configure other session backends:

```go
// PostgreSQL sessions
postgresCfg, _ := config.Get[database.PostgresConfig](cfg, "postgres")
pool, _ := postgresCfg.CreateClient()
store, _ := session.NewPostgresSessionStore(debugMode, []byte(secret), pool, "sessions")
sargantana.SetSessionStore(&store)

// MongoDB sessions
mongoCfg, _ := config.Get[database.MongoConfig](cfg, "mongodb")
client, _ := mongoCfg.CreateClient()
store, _ := session.NewMongoDBSessionStore(debugMode, []byte(secret), client, "myapp", "sessions")
sargantana.SetSessionStore(&store)

// Memcached sessions
memcachedCfg, _ := config.Get[database.MemcachedConfig](cfg, "memcached")
client, _ := memcachedCfg.CreateClient()
store, _ := session.NewMemcachedSessionStore(debugMode, []byte(secret), client)
sargantana.SetSessionStore(&store)
```

### Session Usage in Handlers

```go
func myHandler(c *gin.Context) {
    session := sessions.Default(c)

    // Set session value
    session.Set("key", "value")
    session.Save()

    // Get session value
    value := session.Get("key")

    // Get authenticated user (if using auth controller)
    if user := session.Get("user"); user != nil {
        userObj := user.(controller.UserObject)
        name := userObj.User.Name
        email := userObj.User.Email
    }
}
```

## Custom Controllers

Create your own controllers by implementing the `IController` interface:

```go
type MyController struct {
    // Your fields
}

func (m *MyController) Bind(engine *gin.Engine, config config.Config, loginMiddleware gin.HandlerFunc) {
    // Register your routes
    engine.GET("/api/hello", m.hello)
    engine.GET("/api/protected", loginMiddleware, m.protected)
}

func (m *MyController) Close() error {
    // Cleanup resources
    return nil
}

func (m *MyController) hello(c *gin.Context) {
    c.JSON(200, gin.H{"message": "Hello, World!"})
}

func (m *MyController) protected(c *gin.Context) {
    session := sessions.Default(c)
    user := session.Get("user").(controller.UserObject)
    c.JSON(200, gin.H{"user": user.User.Name})
}

// Constructor function
func NewMyController(configData config.ControllerConfig, serverConfig config.ServerConfig) (controller.IController, error) {
    // Parse your configuration
    // Return configured controller instance
    return &MyController{}, nil
}
```

Register your controller:

```go
server.AddControllerType("my_controller", NewMyController)
```

Use it in configuration:

```yaml
controllers:
  - type: "my_controller"
    name: "my-custom-controller"
    config:
      # Your controller-specific configuration
```

## Database Integration

All database integrations use the `ClientFactory[T]` pattern for type-safe, validated client creation.

### Redis

Redis support includes TLS configuration and connection pooling.

```go
import "github.com/animalet/sargantana-go/pkg/database"
import "github.com/animalet/sargantana-go/pkg/config"

// Load Redis configuration from YAML
redisCfg, err := config.Get[database.RedisConfig](cfg, "redis")
if err != nil {
    log.Fatal(err)
}

// Create connection pool using ClientFactory
pool, err := redisCfg.CreateClient()
if err != nil {
    log.Fatal(err)
}
defer pool.Close()

// Get connection
conn := pool.Get()
defer conn.Close()

// Use Redis commands
conn.Do("SET", "key", "value")
```

Configuration example:
```yaml
redis:
  address: "localhost:6379"
  username: "redisuser"     # Optional
  password: "${REDIS_PASS}" # Optional
  database: 0
  max_idle: 10
  idle_timeout: 240s
  tls:                      # Optional TLS configuration
    insecure_skip_verify: false
    cert_file: "/path/to/cert.pem"
    key_file: "/path/to/key.pem"
    ca_file: "/path/to/ca.pem"
```

### PostgreSQL

PostgreSQL support with connection pooling using pgx/v5.

```go
import "github.com/animalet/sargantana-go/pkg/database"
import "github.com/animalet/sargantana-go/pkg/config"

// Load PostgreSQL configuration from YAML
postgresCfg, err := config.Get[database.PostgresConfig](cfg, "postgres")
if err != nil {
    log.Fatal(err)
}

// Create connection pool using ClientFactory
pool, err := postgresCfg.CreateClient()
if err != nil {
    log.Fatal(err)
}
defer pool.Close()

// Use the connection pool
var result string
err = pool.QueryRow(context.Background(), "SELECT version()").Scan(&result)
```

Configuration example:
```yaml
postgres:
  host: "localhost"
  port: 5432
  database: "myapp"
  user: "${DB_USER}"
  password: "${DB_PASSWORD}"
  ssl_mode: "prefer"              # disable, allow, prefer, require, verify-ca, verify-full
  max_conns: 10                   # Optional: maximum connections
  min_conns: 2                    # Optional: minimum connections
  max_conn_lifetime: 1h           # Optional: max connection lifetime
  max_conn_idle_time: 30m         # Optional: max idle time
  health_check_period: 1m         # Optional: health check interval
```

### MongoDB

MongoDB support with connection pooling and TLS.

```yaml
mongodb:
  uri: "mongodb://localhost:27017"
  database: "myapp"
  username: "${MONGO_USER}"
  password: "${MONGO_PASSWORD}"
  auth_source: "admin"
  max_pool_size: 100
  min_pool_size: 10
  tls:
    enabled: true
    ca_file: "/path/to/ca.pem"
```

### Memcached

Memcached support for high-performance caching.

```yaml
memcached:
  servers:
    - "localhost:11211"
    - "localhost:11212"
  timeout: 100ms
  max_idle_conns: 5
```

## Examples

### Simple Blog Application

```yaml
# config.yaml
server:
  address: "localhost:8080"
  session_name: "blog"
  session_secret: "${SESSION_SECRET}"

controllers:
  - type: "static"
    config:
      statics_dir: "./public"
      templates_dir: "./templates"
  
  - type: "auth"
    config:
      providers:
        github:
          key: "${GITHUB_KEY}"
          secret: "${GITHUB_SECRET}"
```

```go
// main.go
package main

import (
    "flag"
    "fmt"
    "os"

    "github.com/animalet/sargantana-go/config"
    "github.com/animalet/sargantana-go/controller"
    "github.com/animalet/sargantana-go/database"
    "github.com/animalet/sargantana-go/server"
    "github.com/gin-gonic/gin"
    "github.com/jackc/pgx/v5/pgxpool"
)

func main() {
    configFile := flag.String("config", "", "Path to configuration file")
    flag.Parse()

    if *configFile == "" {
        fmt.Fprintln(os.Stderr, "Error: -config flag is required")
        os.Exit(1)
    }

    // Load configuration
    cfg, err := config.ReadModular(*configFile)
    if err != nil {
        panic(err)
    }

    // Setup database connection
    postgresCfg, err := config.Get[database.PostgresConfig](cfg, "postgres")
    if err != nil {
        panic(err)
    }

    pool, err := postgresCfg.CreateClient()
    if err != nil {
        panic(err)
    }
    defer pool.Close()

    // Register built-in controllers
    server.AddControllerType("auth", controller.NewAuthController)
    server.AddControllerType("static", controller.NewStaticController)

    // Register custom controller with database dependency
    server.AddControllerType("blog", NewBlogController(pool))

    // Create and start server
    sargantana, err := server.NewServer(*configFile)
    if err != nil {
        panic(err)
    }

    err = sargantana.StartAndWaitForSignal()
    if err != nil {
        panic(err)
    }
}

type BlogController struct {
    config *BlogConfig
    db     *pgxpool.Pool
}

type BlogConfig struct {
    // Add blog-specific configuration fields here
}

func (c BlogConfig) Validate() error {
    return nil
}

func (b *BlogController) Bind(engine *gin.Engine, loginMiddleware gin.HandlerFunc) {
    api := engine.Group("/api")
    {
        api.GET("/posts", b.getPosts)
        api.POST("/posts", loginMiddleware, b.createPost)
        api.DELETE("/posts/:id", loginMiddleware, b.deletePost)
    }
}

func (b *BlogController) Close() error {
    return nil
}

// Constructor pattern - returns a function that creates controller instances
func NewBlogController(db *pgxpool.Pool) server.Constructor {
    return func(configData config.ModuleRawConfig, ctx server.ControllerContext) (server.IController, error) {
        cfg, err := config.Unmarshal[BlogConfig](configData)
        if err != nil {
            return nil, err
        }
        return &BlogController{config: cfg, db: db}, nil
    }
}

func (b *BlogController) getPosts(c *gin.Context) {
    // Query database using b.db
    // Implementation here
}

func (b *BlogController) createPost(c *gin.Context) {
    // Insert into database using b.db
    // Implementation here
}

func (b *BlogController) deletePost(c *gin.Context) {
    // Delete from database using b.db
    // Implementation here
}
```

See `examples/blog_example/` for a complete working blog application with PostgreSQL integration and Keycloak authentication.

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
