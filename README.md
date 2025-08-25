# Sargantana Go

```
▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄
██░▄▄▄░█░▄▄▀█░▄▄▀█░▄▄▄█░▄▄▀█░▄▄▀█▄░▄█░▄▄▀█░▄▄▀█░▄▄▀██░▄▄░█▀▄▄▀
██▄▄▄▀▀█░▀▀░█░▀▀▄█░█▄▀█░▀▀░█░██░██░██░▀▀░█░██░█░▀▀░██░█▀▀█░██░
██░▀▀▀░█▄██▄█▄█▄▄█▄▄▄▄█▄██▄█▄██▄██▄██▄██▄█▄██▄█▄██▄██░▀▀▄██▄▄█
▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀                                    

      🦎 Fast • Flexible • Full-Stack Go Web Framework
```

[![CI](https://github.com/animalet/sargantana-go/workflows/CI/badge.svg)](https://github.com/animalet/sargantana-go/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/animalet/sargantana-go/branch/main/graph/badge.svg)](https://codecov.io/gh/animalet/sargantana-go)
[![Go Report Card](https://goreportcard.com/badge/github.com/animalet/sargantana-go)](https://goreportcard.com/report/github.com/animalet/sargantana-go)
[![Go Reference](https://pkg.go.dev/badge/github.com/animalet/sargantana-go.svg)](https://pkg.go.dev/github.com/animalet/sargantana-go)
[![License](https://img.shields.io/github/license/animalet/sargantana-go)](LICENSE)

## What is this?

Sargantana Go is a performant web application framework built on top of [Gin](https://github.com/gin-gonic/gin) that
provides simple solutions for common web development scenarios. It includes built-in support for OAuth2 authentication,
session management, static file serving, load balancing, and database integration.

I started this as a side project to improve my Go skills and to have a solid base for building web applications quickly. It is designed to be easy to use and extend, allowing developers to focus on building their applications rather than dealing with boilerplate code.

## Features

- **Web Server**: High-performance HTTP server using [Gin](https://github.com/gin-gonic/gin)
- **OAuth2 Authentication**: Multi-provider OAuth2 support via [Goth](https://github.com/markbates/goth) with 50+
  providers
- **Session Management**: Flexible session storage with Redis or cookie-based options
- **Static File Serving**: Built-in static file and template serving capabilities
- **Load Balancing**: Round-robin load balancer with optional authentication
- **Database Support**: Redis and Neo4j integration
- **Configuration**: Command-line flags and Docker secrets support
- **Production Ready**: Docker Compose deployment with proper secrets management

## Quick Start

### Prerequisites

- Go 1.25.0 or later
- Make (for development)

### Installation

```bash
go get github.com/animalet/sargantana-go
```

### Basic Usage

#### Create a simple web application configured via command line flags

```go
package main

import (
    "flag"
    "github.com/animalet/sargantana-go/controller"
    "github.com/animalet/sargantana-go/server"
)

func main() {
    // Define controllers you want to use
    controllerInitializers := []func(*flag.FlagSet) func() controller.IController{
        controller.NewStaticFromFlags,       // Static file serving
        controller.NewAuthFromFlags,         // OAuth2 authentication
        controller.NewLoadBalancerFromFlags, // Load balancing
    }

    // Create server and controllers from command line flags
    sargantana, controllers := server.NewServerFromFlags(controllerInitializers...)

    // Start server and wait for shutdown signal
    err := sargantana.StartAndWaitForSignal(controllers...)
    if err != nil {
        panic(err)
    }
}
```

#### Create a simple web application with programmatic configuration

```go
package main

import (
    "net/url"
    "github.com/animalet/sargantana-go/controller"
    "github.com/animalet/sargantana-go/server"
)

func main() {
    // Define static file controller
    static := controller.NewStatic("./public", "./templates")

    // Define auth controller with a callback URL (can be customised if you run behind a proxy) that will issue OAuth callbacks to path http://myapplication.com/auth/{provider}/callback
    auth := controller.NewAuth("http://myapplication.com")

    // Define load balancer with endpoints
    endpoints := []url.URL{
        {Scheme: "http", Host: "api1:8080"},
        {Scheme: "http", Host: "api2:8080"},
    }
    lb := controller.NewLoadBalancer(endpoints, "api", true)

    // Create server with controllers
    sargantana := server.NewServer("localhost", 8080, "" /* No Redis means cookie sessions*/, "/run/secrets", true, "my-session-identifier")

    // Start server and wait for shutdown signal
    err := sargantana.StartAndWaitForSignal(static, auth, lb)
    if err != nil {
        panic(err)
    }
}
```

#### Create a simple web application with custom lifecycle control

```go
package main

import (
    "net/url"
    "os"
    "os/signal"
    "syscall"
    "github.com/animalet/sargantana-go/controller"
    "github.com/animalet/sargantana-go/server"
)

func main() {
    // Define controllers as before
    static := controller.NewStatic("./public", "./templates")
    auth := controller.NewAuth("http://myapplication.com")
    endpoints := []url.URL{
        {Scheme: "http", Host: "api1:8080"},
        {Scheme: "http", Host: "api2:8080"},
    }
    lb := controller.NewLoadBalancer(endpoints, "api", true)
    sargantana := server.NewServer("localhost", 8080, "", "/run/secrets", true, "my-session-identifier")
    // Start server
    err := sargantana.Start(static, auth, lb)
    if err != nil {
        panic(err)
    }

    // Wait for termination signal
    sigs := make(chan os.Signal, 1)
    signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
    <-sigs
    // Shutdown server and controllers
    err = sargantana.Shutdown()
    if err != nil {
        panic(err)
    }
}
```

### Running the Application

```bash
# Basic server on localhost:8080
go run main.go

# With custom configuration
go run main.go \
  -host 0.0.0.0 \
  -port 3000 \
  -frontend ./public \
  -templates ./views \
  -debug
```

## Configuration

### Command Line Flags

| Flag          | Description                    | Default         | Example                                     |
|---------------|--------------------------------|-----------------|---------------------------------------------|
| `-host`       | Host to listen on              | `localhost`     | `-host 0.0.0.0`                             |
| `-port`       | Port to listen on              | `8080`          | `-port 3000`                                |
| `-debug`      | Enable debug mode              | `false`         | `-debug`                                    |
| `-secrets`    | Path to secrets directory      | `""`            | `-secrets ./secrets`                        |
| `-redis`      | Redis address for sessions     | `""`            | `-redis localhost:6379`                     |
| `-cookiename` | Session cookie name            | `sargantana-go` | `-cookiename myapp`                         |
| `-frontend`   | Static files directory         | `./frontend`    | `-frontend ./public`                        |
| `-templates`  | Templates directory            | `./templates`   | `-templates ./views`                        |
| `-lbpath`     | Load balancer path             | `lb`            | `-lbpath api`                               |
| `-lbauth`     | Require auth for load balancer | `false`         | `-lbauth`                                   |
| `-lb`         | Load balancer endpoints        | `[]`            | `-lb http://api1:8080 -lb http://api2:8080` |

### Environment Variables

Set these environment variables for OAuth2 providers:

```bash
# Session security
SESSION_SECRET=your-session-secret-key

# OAuth2 Providers (choose the ones you need)
GOOGLE_KEY=your-google-client-id
GOOGLE_SECRET=your-google-client-secret

GITHUB_KEY=your-github-client-id
GOOGLE_SECRET=your-github-client-secret

TWITTER_KEY=your-twitter-api-key
TWITTER_SECRET=your-twitter-api-secret

# See full list of supported providers below
```

### Docker Secrets

You can also use Docker secrets by placing secret files in a directory and using the `-secrets` flag:

```bash
# Directory structure
secrets/
├── session_secret
├── google_key
├── google_secret
├── github_key
└── github_secret

# Run with secrets
go run main.go -secrets ./secrets
```

## Controllers

Sargantana Go uses a controller-based architecture. Each controller handles a specific aspect of your application.

### Static Controller

Serves static files and HTML templates:

```go
// Programmatic usage
static := controller.NewStatic("./public", "./templates")

// With flags
go run main.go -frontend./public -templates./templates
```

Features:

- Serves files from `/static/*` route
- Serves `index.html` at root `/`
- Loads HTML templates with `{{ }}` syntax
- Automatic template discovery

### Auth Controller

Provides OAuth2 authentication with 50+ providers:

```go
// Programmatic usage  
auth := controller.NewAuth("http://localhost:8080")

// With flags (authentication is automatic when providers are configured)
go run main.go
```

**Supported OAuth2 Providers:**

- Google, GitHub, Facebook, Twitter/X
- Microsoft, Apple, Amazon, Discord
- LinkedIn, Instagram, Spotify, Twitch
- Auth0, Okta, Azure AD
- And 35+ more providers

**Authentication Flow:**

1. Visit `/auth/{provider}` to start OAuth flow
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

```go
// Programmatic usage
endpoints := []url.URL{
{Scheme: "http", Host: "api1:8080"},
{Scheme: "http", Host: "api2:8080"},
}
lb := controller.NewLoadBalancer(endpoints, "api", true)

// With flags
go run main.go \
-lb http: //api1:8080 \
-lb http: //api2:8080 \
-lbpath api \
-lbauth
```

Features:

- Round-robin load balancing
- Optional authentication requirement
- Support for all HTTP methods
- Automatic failover
- Request forwarding with headers

## Session Management

### Cookie-based Sessions (Default)

```bash
# Uses secure cookies for session storage
go run main.go -cookiename myapp
```

### Redis Sessions

```bash
# Use Redis for distributed session storage
go run main.go -redis localhost:6379
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

// Flag-based constructor
func NewMyControllerFromFlags(flagSet *flag.FlagSet) func () controller.IController {
// Define your flags here
return func () controller.IController { return &MyController{} }
}
```

## Database Integration

### Redis

Redis support is very basic and is currently used for session storage. You can also use it directly in your controllers.

There is currently no support for authentication or TLS, but you can extend the `NewRedisPool` function to add these
features. Pull requests are welcome!

```go
import "github.com/animalet/sargantana-go/database"

// Create Redis connection pool
pool := database.NewRedisPool("localhost:6379")
defer pool.Close()

// Get connection
conn := pool.Get()
defer conn.Close()

// Use Redis commands
conn.Do("SET", "key", "value")
```

### Neo4j

```go
import "github.com/animalet/sargantana-go/database"

// Configure via environment variables:
// NEO4J_URI=bolt://localhost:7687
// NEO4J_USERNAME=neo4j  
// NEO4J_PASSWORD=password

driver := database.NewNeo4jDriver()
defer driver.Close()
```

## Examples

### Simple Blog Application

```go
func main() {
controllers := []func (*flag.FlagSet) func () controller.IController{
controller.NewStaticFromFlags,
controller.NewAuthFromFlags,
NewBlogControllerFromFlags,
}

server, controllerInstances := server.NewServerFromFlags(controllers...)
server.StartAndWaitForSignal(controllerInstances...)
}

type BlogController struct{}

func (b *BlogController) Bind(engine *gin.Engine, config config.Config, loginMiddleware gin.HandlerFunc) {
api := engine.Group("/api")
{
api.GET("/posts", b.getPosts)
api.POST("/posts", loginMiddleware, b.createPost)
api.DELETE("/posts/:id", loginMiddleware, b.deletePost)
}
}

func (b *BlogController) Close() error { return nil }

func NewBlogControllerFromFlags(flagSet *flag.FlagSet) func () controller.IController {
return func () controller.IController { return &BlogController{} }
}
```

### API Gateway with Load Balancing

```bash
# Start multiple backend services
go run backend.go -port 8081 &
go run backend.go -port 8082 &

# Start API gateway with load balancing
go run main.go \
  -port 8080 \
  -lb http://localhost:8081 \
  -lb http://localhost:8082 \
  -lbpath api \
  -lbauth
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
      - session_secret
      - google_client_id
      - google_client_secret
    command: [
      "/app/sargantana-go",
      "-host", "0.0.0.0",
      "-port", "8080",
      "-redis", "redis:6379",
      "-secrets", "/run/secrets"
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
  GOOGLE_KEY:
    file: ./secrets/google_client_id
  GOOGLE_SECRET:
    file: ./secrets/google_client_secret

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
RUN go build -o sargantana-go main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/sargantana-go .
COPY --from=builder /app/frontend ./frontend
COPY --from=builder /app/templates ./templates
CMD ["./sargantana-go"]
```

## Development

### Prerequisites

- Go 1.25.0 or later
- Make

### Installation

```bash
git clone https://github.com/animalet/sargantana-go.git
cd sargantana-go
make all
```

### Development Commands

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
```

### Project Structure

```
sargantana-go/
├── main/           # Main application entry point
├── server/         # Core server implementation
├── controller/     # Built-in controllers (auth, static, load balancer)
├── config/         # Configuration management
├── session/        # Session storage implementations
├── database/       # Database integrations (Redis, Neo4j)
├── frontend/       # Example frontend assets
├── templates/      # Example HTML templates
└── Makefile        # Development commands
```

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Support

- 📖 [Documentation](https://pkg.go.dev/github.com/animalet/sargantana-go)
- 🐛 [Bug Reports](https://github.com/animalet/sargantana-go/issues)
- 💡 [Feature Requests](https://github.com/animalet/sargantana-go/issues)
- 💬 [Discussions](https://github.com/animalet/sargantana-go/discussions)
