# Sargantana Go - Claude Context

## Project Overview

Sargantana Go is a Go web framework built on top of Gin. It provides a modular, controller-based architecture for building web applications with support for session management, OAuth authentication, template rendering, and static file serving.

## Architecture

### Core Components

1. **Server** (`server/`)
   - Main server initialization and lifecycle management
   - Gin engine configuration
   - Controller registration and binding
   - Session management (Cookie or Redis-based)
   - Graceful shutdown with cleanup hooks

2. **Controllers** (`controller/`)
   - Modular controller system with `IController` interface
   - Built-in controllers:
     - `auth`: OAuth authentication via Goth (Google, GitHub, etc.)
     - `static`: Static file and directory serving
     - `template`: HTML template rendering
   - Controllers are registered via configuration

3. **Configuration** (`config/`)
   - YAML-based configuration
   - Environment variable expansion (`${VAR}` syntax)
   - Generic `ClientFactory[T]` interface for type-safe client creation
   - Generic unmarshaling with validation

4. **Resolver** (`resolver/`)
   - Pluggable property resolver system
   - Built-in resolvers:
     - Environment variables (`env:VAR_NAME`)
     - Files (`file:filename`)
     - HashiCorp Vault (`vault:secret/path`)
   - Custom resolver support

5. **Database** (`database/`)
   - Redis connection pooling with TLS support
   - PostgreSQL connection pooling with comprehensive configuration
   - Neo4j graph database support
   - All use `ClientFactory[T]` pattern for type-safe client creation

6. **Session Management** (`session/`)
   - Cookie-based sessions
   - Redis-backed sessions
   - Integration with Gin sessions middleware

## Configuration Structure

### Main Config (`config.yaml`)

```yaml
server:
  address: ":8080"
  session_name: "app-session"
  session_secret: "${SESSION_SECRET}"  # Supports env var expansion
  secrets_dir: "./secrets"             # Optional: for file: secrets
  redis_session_store:                 # Optional: use Redis for sessions
    address: "localhost:6379"
    max_idle: 10
    idle_timeout: 240s

vault:                                  # Optional: HashiCorp Vault integration
  address: "http://localhost:8200"
  token: "${VAULT_TOKEN}"
  path: "secret/data/myapp"

# Optional: PostgreSQL database
postgres:
  host: "localhost"
  port: 5432
  database: "myapp"
  user: "${file:DB_USER}"
  password: "${vault:DB_PASSWORD}"
  ssl_mode: "prefer"
  max_conns: 10
  min_conns: 2

controllers:
  - type: "auth"
    name: "oauth"
    config:
      success_redirect_path: "/dashboard"
      error_redirect_path: "/login"
      providers:
        - name: "google"
          client_id: "${GOOGLE_CLIENT_ID}"
          client_secret: "${GOOGLE_CLIENT_SECRET}"

  # Multiple static controller instances for different paths
  - type: "static"
    name: "public-assets"
    config:
      path: "/static"
      dir: "./static"

  - type: "static"
    name: "favicon"
    config:
      path: "/favicon.ico"
      file: "./static/favicon.ico"

  - type: "template"
    config:
      path: "./templates"
```

### Multiple Controller Instances

**Each controller type can have multiple instances.** The `name` field in the controller binding is used to identify each instance:

```yaml
controllers:
  # First static controller instance
  - type: "static"
    name: "public"
    config:
      path: "/public"
      dir: "./public"

  # Second static controller instance
  - type: "static"
    name: "admin"
    config:
      path: "/admin"
      dir: "./admin/static"

  # Third instance - name is optional (auto-generated as "static-3")
  - type: "static"
    config:
      path: "/assets"
      dir: "./assets"
```

**Instance naming rules:**
- If `name` is provided, it's used as-is
- If `name` is omitted:
  - First instance: uses controller type name (e.g., `static`)
  - Subsequent instances: appends number (e.g., `static-2`, `static-3`)

This allows you to:
- Configure multiple instances of the same controller type
- Each instance serves different purposes with different configurations
- Easily identify instances in logs and errors

## Development Setup

### Prerequisites

- Go 1.25.0 or higher
- Make

### Installation

```bash
# Install development tools
make configure

# Install dependencies
make deps

# Build the project
make build
```

### Running Tests

```bash
# Run all tests
make test

# Run tests with coverage
make test-with-coverage

# Check coverage thresholds
make check-coverage
```

### Linting

```bash
# Run linter (automatically installs/updates golangci-lint v2)
make lint

# Format code
make format
```

## Project Structure

```
sargantana-go/
├── main/             # Entry point
│   └── main.go
├── pkg/              # Core packages
│   ├── config/       # Configuration management
│   ├── controller/   # Controller implementations
│   │   ├── auth.go
│   │   ├── static.go
│   │   └── template.go
│   ├── database/     # Database connections with ClientFactory
│   │   ├── neo4j.go
│   │   ├── postgres.go
│   │   └── redis.go
│   ├── resolver/     # Property resolvers
│   │   ├── env.go
│   │   ├── file.go
│   │   ├── vault.go
│   │   └── registry.go
│   ├── server/       # Server core
│   └── session/      # Session management
├── examples/         # Example applications
│   └── blog_example/
├── docs/             # Documentation
└── Makefile          # Build automation
```

## Key Interfaces

### IController

All controllers must implement:

```go
type IController interface {
    Bind(engine *gin.Engine, authMiddleware gin.HandlerFunc)
    Close() error
}
```

### Validatable

Configuration structs should implement:

```go
type Validatable interface {
    Validate() error
}
```

### ClientFactory

Database and service configurations implement the generic `ClientFactory[T]` interface for type-safe client creation:

```go
type ClientFactory[T any] interface {
    Validatable
    CreateClient() (T, error)
}
```

**Examples:**
- `VaultConfig` implements `ClientFactory[*api.Client]`
- `RedisConfig` implements `ClientFactory[*redis.Pool]`
- `PostgresConfig` implements `ClientFactory[*pgxpool.Pool]`

**Usage:**
```go
// Load and create PostgreSQL client
postgresCfg, err := config.LoadConfig[database.PostgresConfig]("postgres", cfg)
pool, err := postgresCfg.CreateClient()  // Returns *pgxpool.Pool directly
defer pool.Close()

// Load and create Redis client
redisCfg, err := config.LoadConfig[database.RedisConfig]("redis", cfg)
redisPool, err := redisCfg.CreateClient()  // Returns *redis.Pool directly
defer redisPool.Close()
```

**Benefits:**
- Type-safe: No type assertions needed
- Consistent: All data sources use the same pattern
- Validated: Configuration is validated before client creation

## Adding a New Controller

1. Create a new controller type in `controller/`:

```go
type myController struct {
    IController
    // fields
}

type MyControllerConfig struct {
    // config fields
}

func (c MyControllerConfig) Validate() error {
    // validation logic
    return nil
}

func NewMyController(configData config.ControllerConfig, serverCfg config.ServerConfig) (IController, error) {
    cfg, err := config.UnmarshalTo[MyControllerConfig](configData)
    if err != nil {
        return nil, err
    }
    return &myController{/* ... */}, nil
}

func (m *myController) Bind(engine *gin.Engine, auth gin.HandlerFunc) {
    // register routes
}

func (m *myController) Close() error {
    // cleanup
    return nil
}
```

2. Register the controller type in `server/server.go`:

```go
func init() {
    AddControllerType("mycontroller", controller.NewMyController)
}
```

3. Add to configuration:

```yaml
controllers:
  - type: "mycontroller"
    config:
      # controller-specific config
```

## Testing Guidelines

- Use `t.TempDir()` for temporary directories in tests
- Always create actual files/directories that tests validate
- Reset global state between tests (e.g., `vaultManagerInstance`)
- Use `gin.SetMode(gin.TestMode)` for Gin-based tests
- Test both success and error paths

## Common Issues and Solutions

### Linter Version Mismatch

The project uses golangci-lint v2. The Makefile automatically installs the latest version:

```bash
make install-golangci-lint
```

### Session Secret Missing

Always set `session_secret` in configuration. Use environment variables for security:

```yaml
session_secret: "${SESSION_SECRET}"
```

### Static File Not Found

Ensure paths in static controller config are absolute or relative to the working directory:

```yaml
- type: "static"
  config:
    - path: "/static"
      dir: "./static"  # Relative to working directory
```

## Build and Release

### Building

```bash
# Build for current platform
make build

# Build for all platforms
make build-all
```

### Version Information

Version is automatically determined from git tags:
- On tagged commit: uses tag as-is
- Otherwise: `{tag}-{commit}` or `{tag}-{commit}-dirty`

## CI/CD

The `make ci` command runs:
1. Dependency management (`make deps`)
2. Tests with coverage (`make test-with-coverage`)
3. Linting (`make lint`)

## Environment Variables

Common environment variables:

- `SESSION_SECRET`: Session encryption secret
- `GOOGLE_CLIENT_ID`: Google OAuth client ID
- `GOOGLE_CLIENT_SECRET`: Google OAuth client secret
- `VAULT_TOKEN`: HashiCorp Vault authentication token
- `GIN_MODE`: Set to `release` for production

## Dependencies

Key dependencies:
- **gin-gonic/gin**: Web framework
- **gin-contrib/sessions**: Session management
- **markbates/goth**: OAuth authentication
- **gomodule/redigo**: Redis client
- **hashicorp/vault/api**: Vault client
- **neo4j/neo4j-go-driver**: Neo4j driver
- **jackc/pgx/v5**: PostgreSQL driver with connection pooling
- **rs/zerolog**: Structured logging

## Development Workflow

1. Make changes to code
2. Run tests: `make test`
3. Run linter: `make lint`
4. Build: `make build`
5. Test locally
6. Commit changes
7. CI runs automatically

## Notes for Claude

- Go version: 1.25.0
- Always check test files for context on how components are used
- Configuration uses YAML with validation
- Controllers are dynamically registered and configured
- Session management can be either cookie-based or Redis-based
- Property resolvers are in `pkg/resolver/` package (decoupled from config)
- Vault configuration and client creation are in `resolver` package, not `config`
- All database configs use `ClientFactory[T]` pattern for type-safe client creation
- Configuration structs implement `Validate()` with **value receivers** (not pointer receivers)
- Use `config.LoadConfig[T]` to load partial configuration sections with validation
