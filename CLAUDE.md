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
     - `auth`: OAuth authentication via Goth (45+ providers: Google, GitHub, Keycloak, etc.)
     - `static`: Static file and directory serving
     - `template`: HTML template rendering
     - `load_balancer`: Round-robin load balancing with optional authentication
   - Controllers are registered via configuration
   - Support for custom controllers with dependency injection (see `ControllerContext`)

3. **Configuration** (`config/`)
   - YAML-based configuration
   - Environment variable expansion (`${VAR}` syntax)
   - Generic `ClientFactory[T]` interface for type-safe client creation
   - Generic unmarshaling with validation

4. **Secrets** (`secrets/`)
   - Automatic secrets resolution based on prefix in configuration values
   - Built-in secret providers:
     - Environment variables (`env:VAR_NAME` or no prefix)
     - Files (`file:filename`)
     - HashiCorp Vault (`vault:secret/path`)
     - AWS Secrets Manager (`aws:secret-name`)
   - Secrets are resolved at configuration load time
   - Configure secret sources (Vault, AWS) in the YAML config file

5. **Database** (`database/`)
   - Redis connection pooling with TLS support
   - PostgreSQL connection pooling with comprehensive configuration
   - MongoDB connection with TLS and authentication
   - Memcached connection with multiple servers support
   - All use `ClientFactory[T]` pattern for type-safe client creation

6. **Session Management** (`session/`)
   - Cookie-based sessions (default)
   - Redis-backed sessions (distributed)
   - PostgreSQL-backed sessions (persistent)
   - MongoDB-backed sessions (NoSQL)
   - Memcached-backed sessions (high-performance)
   - All session stores integrate with Gin sessions middleware
   - Configurable via `SetSessionStore()` method

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

# Optional: MongoDB database
mongodb:
  uri: "mongodb://localhost:27017"
  database: "myapp"
  username: "${MONGO_USER}"
  password: "${MONGO_PASSWORD}"
  auth_source: "admin"
  max_pool_size: 100
  min_pool_size: 10
  tls:
    enabled: false

# Optional: Memcached
memcached:
  servers:
    - "localhost:11211"
  timeout: 100ms
  max_idle_conns: 5

# Optional: AWS Secrets Manager
aws:
  region: "us-east-1"
  access_key_id: "${AWS_ACCESS_KEY_ID}"      # Optional: uses IAM role if omitted
  secret_access_key: "${AWS_SECRET_ACCESS_KEY}"  # Optional: uses IAM role if omitted
  endpoint: ""                                # Optional: for LocalStack testing
  secret_name: "myapp/secrets"

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

  - type: "load_balancer"
    name: "api-lb"
    config:
      path: "/api"
      require_auth: true
      endpoints:
        - "http://backend1:8080"
        - "http://backend2:8080"
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
- `MongoDBConfig` implements `ClientFactory[*mongo.Client]`
- `MemcachedConfig` implements `ClientFactory[*memcache.Client]`
- `AWSConfig` implements `ClientFactory[*secretsmanager.Client]`

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

// Load and create MongoDB client
mongoCfg, err := config.LoadConfig[database.MongoDBConfig]("mongodb", cfg)
mongoClient, err := mongoCfg.CreateClient()  // Returns *mongo.Client directly
defer mongoClient.Disconnect(context.Background())

// Load and create Memcached client
memcachedCfg, err := config.LoadConfig[database.MemcachedConfig]("memcached", cfg)
memcachedClient, err := memcachedCfg.CreateClient()  // Returns *memcache.Client directly

// Load and create AWS Secrets Manager client
awsCfg, err := config.LoadConfig[secrets.AWSConfig]("aws", cfg)
awsClient, err := awsCfg.CreateClient()  // Returns *secretsmanager.Client directly
```

**Benefits:**
- Type-safe: No type assertions needed
- Consistent: All data sources use the same pattern
- Validated: Configuration is validated before client creation

## Session Store Configuration

The framework supports multiple session storage backends. By default, cookie-based sessions are used. To use a different backend, create the appropriate client and session store in your application:

### Redis Sessions

```go
redisCfg, err := config.LoadConfig[database.RedisConfig]("redis", cfg)
if err == nil {
    redisPool, err := redisCfg.CreateClient()
    if err != nil {
        log.Fatal().Err(err).Msg("Failed to create Redis client")
    }
    store, err := session.NewRedisSessionStore(*debugMode, []byte(cfg.ServerConfig.SessionSecret), redisPool)
    if err != nil {
        log.Fatal().Err(err).Msg("Failed to create Redis session store")
    }
    sargantana.SetSessionStore(&store)
}
```

### PostgreSQL Sessions

```go
postgresCfg, err := config.LoadConfig[database.PostgresConfig]("postgres", cfg)
if err == nil {
    pool, err := postgresCfg.CreateClient()
    if err != nil {
        log.Fatal().Err(err).Msg("Failed to create PostgreSQL client")
    }
    store, err := session.NewPostgresSessionStore(*debugMode, []byte(cfg.ServerConfig.SessionSecret), pool, "sessions")
    if err != nil {
        log.Fatal().Err(err).Msg("Failed to create PostgreSQL session store")
    }
    sargantana.SetSessionStore(&store)
}
```

### MongoDB Sessions

```go
mongoCfg, err := config.LoadConfig[database.MongoDBConfig]("mongodb", cfg)
if err == nil {
    client, err := mongoCfg.CreateClient()
    if err != nil {
        log.Fatal().Err(err).Msg("Failed to create MongoDB client")
    }
    store, err := session.NewMongoDBSessionStore(*debugMode, []byte(cfg.ServerConfig.SessionSecret), client, mongoCfg.Database, "sessions")
    if err != nil {
        log.Fatal().Err(err).Msg("Failed to create MongoDB session store")
    }
    sargantana.SetSessionStore(&store)
}
```

### Memcached Sessions

```go
memcachedCfg, err := config.LoadConfig[database.MemcachedConfig]("memcached", cfg)
if err == nil {
    client, err := memcachedCfg.CreateClient()
    if err != nil {
        log.Fatal().Err(err).Msg("Failed to create Memcached client")
    }
    store, err := session.NewMemcachedSessionStore(*debugMode, []byte(cfg.ServerConfig.SessionSecret), client)
    if err != nil {
        log.Fatal().Err(err).Msg("Failed to create Memcached session store")
    }
    sargantana.SetSessionStore(&store)
}
```

## Adding a New Controller

### Basic Controller (No Dependencies)

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

func NewMyController(configData config.ControllerConfig, ctx controller.ControllerContext) (IController, error) {
    cfg, err := config.UnmarshalTo[MyControllerConfig](configData)
    if err != nil {
        return nil, err
    }
    // Access runtime dependencies from ctx (ServerConfig, SessionStore, etc.)
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

2. Register the controller type in `main/main.go`:

```go
func init() {
    server.AddControllerType("mycontroller", controller.NewMyController)
}
```

3. Add to configuration:

```yaml
controllers:
  - type: "mycontroller"
    config:
      # controller-specific config
```

### Controller with Dependencies (Constructor Pattern)

For controllers that need external dependencies (e.g., database connections):

```go
// Controller with database dependency
func NewBlogController(db *pgxpool.Pool) server.Constructor {
    return func(configData config.ControllerConfig, ctx controller.ControllerContext) (controller.IController, error) {
        cfg, err := config.UnmarshalTo[BlogConfig](configData)
        if err != nil {
            return nil, err
        }
        return &blogController{
            config: cfg,
            db:     db,
        }, nil
    }
}
```

Register with dependency:

```go
// Create database connection
pool, err := postgresCfg.CreateClient()
if err != nil {
    log.Fatal(err)
}

// Register controller with database
server.AddControllerType("blog", NewBlogController(pool))
```

See `examples/blog_example/` for a complete working example.

## Testing Guidelines

- Use `t.TempDir()` for temporary directories in tests
- Always create actual files/directories that tests validate
- Reset global state between tests (e.g., `vaultManagerInstance`)
- Use `gin.SetMode(gin.TestMode)` for Gin-based tests
- Test both success and error paths
- **IMPORTANT**: Some tests (AWS Secrets Manager, Vault) require Docker Compose services to be running. Run `docker-compose up -d` in the project root before running the full test suite

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
- **markbates/goth**: OAuth authentication (45+ providers)
- **gomodule/redigo**: Redis client
- **hashicorp/vault/api**: Vault client
- **aws/aws-sdk-go-v2/service/secretsmanager**: AWS Secrets Manager client
- **jackc/pgx/v5**: PostgreSQL driver with connection pooling
- **mongodb/mongo-go-driver**: MongoDB driver
- **bradfitz/gomemcache**: Memcached client
- **rs/zerolog**: Structured logging

## Development Workflow

1. Make changes to code
2. Run tests: `make test`
3. Run linter: `make lint`
4. Build: `make build`
5. Test locally
6. Commit changes
7. CI runs automatically

## Key Architectural Patterns

1. **ClientFactory[T] Pattern** - All database and secrets configurations use this generic interface for type-safe, validated client creation
2. **Constructor Pattern** - Controllers with custom initialization logic and dependency injection
3. **ControllerContext Pattern** - Separation of YAML config from runtime dependencies
4. **Controller Registry Pattern** - Dynamic controller type registration via `AddControllerType()`
5. **Secret Resolution Pattern** - Automatic secret resolution based on prefixes in configuration values (`env:`, `file:`, `vault:`, `aws:`)

## Examples

- **Blog Application**: See `examples/blog_example/` for a complete working example with:
  - Custom controller with PostgreSQL integration
  - Keycloak authentication setup
  - Docker Compose configuration with all services
  - Session management and database integration

## Notes for Claude

- Go version: 1.25.0
- Always check test files for context on how components are used
- Configuration uses YAML with validation via `Validatable` interface
- Controllers are dynamically registered and configured via `AddControllerType()`
- Controller registration happens in `main/main.go`, not in `server/server.go`
- Session management supports 5 backends: Cookie, Redis, PostgreSQL, MongoDB, Memcached
- Secret resolution is automatic based on prefixes in configuration values
- Secret providers: Environment (`env:` or no prefix), File (`file:`), Vault (`vault:`), AWS Secrets Manager (`aws:`)
- Configure Vault/AWS in the YAML config; secrets are resolved at load time
- All database configs use `ClientFactory[T]` pattern for type-safe client creation
- Configuration structs implement `Validate()` with **value receivers** (not pointer receivers)
- Use `config.LoadConfig[T]` to load partial configuration sections with validation from the `Other` map
- Controllers use `ControllerContext` for runtime dependencies (ServerConfig, SessionStore)
- The `Constructor` type allows parameterized controller creation (e.g., with database connections)
- Load balancer controller provides round-robin distribution with header filtering for security
