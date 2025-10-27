# Blog Example - Sargantana Go

A complete blog application demonstrating the Sargantana Go framework with authentication, database integration, and secrets management.

## Features

- **Authentication**: Mock OAuth2 provider (Google-like) for easy local development
- **Database**: PostgreSQL with connection pooling
- **Sessions**: Redis-backed session storage
- **Secrets Management**: Hybrid approach using file-based secrets and HashiCorp Vault
- **Template Rendering**: Server-side HTML templates
- **Static Assets**: CSS and favicon serving

## Quick Start

### Prerequisites

- Docker 20.10+
- Docker Compose 2.0+

### Running the Example

1. **Start all services:**
   ```bash
   cd examples/blog_example
   docker-compose up
   ```

   All secrets are automatically initialized in Vault on startup - no manual secret files needed!

2. **Access the application:**
   - Blog: http://localhost:8080/blog/feed
   - Vault UI: http://localhost:8200 (token: dev-root-token)

### Authentication

**🌐 Google OAuth (Mock)**
- Works immediately, no configuration needed
- Simulates Google OAuth for development
- Enter any username to log in
- Perfect for testing without real OAuth credentials

## Architecture

### Services

| Service | Port | Purpose |
|---------|------|---------|
| **blog** | 8080 | Main application |
| **postgres** | 5432 | Database |
| **redis** | 6379 | Session storage |
| **vault** | 8200 | Secrets management |
| **mockoidc** | 8082 | Mock OAuth2 server |

### Configuration

The example demonstrates **modular configuration** with multiple secret sources:

```yaml
# All secrets from Vault (auto-initialized on startup)
session_secret: "${vault:SESSION_SECRET}"
user: "${vault:DB_USER}"
password: "${vault:DB_PASSWORD}"
key: "${vault:OPENID_CONNECT_KEY}"
secret: "${vault:OPENID_CONNECT_SECRET}"
```

**Key files:**
- `config.docker.yaml` - Docker-specific configuration
- `main.go` - Application entry point showing resolver setup
- `blog/blog.go` - Blog controller implementation
- `content/templates/` - HTML templates

## Technical Details

### Secrets Management

**All secrets are stored in Vault** (`secret/data/blog`) and auto-initialized on startup:
- `SESSION_SECRET` - Session encryption key
- `DB_USER` - PostgreSQL username
- `DB_PASSWORD` - PostgreSQL password
- `OPENID_CONNECT_KEY` - OAuth client ID
- `OPENID_CONNECT_SECRET` - OAuth client secret

**Vault Commands:**
```bash
# View secrets
docker exec blog-vault vault kv get secret/blog

# Update secrets
docker exec blog-vault vault kv put secret/blog KEY=value
```

### Database Schema

The blog controller automatically creates the required table on startup:

```sql
CREATE TABLE IF NOT EXISTS posts (
    id SERIAL PRIMARY KEY,
    title TEXT NOT NULL,
    content TEXT NOT NULL,
    publication_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    owner TEXT NOT NULL
);
```

### Authentication Flow

1. User clicks "Login with Google"
2. Redirected to mock OAuth2 server
3. User enters any username
4. After successful login, redirected to callback endpoint
5. Session created with user information
6. User redirected to `/blog/feed`

### Controller Registration

The example demonstrates modular controller setup:

```go
// Register resolvers
resolver.Register("env", resolver.NewEnvResolver())
resolver.Register("file", fileResolver)
resolver.Register("vault", vaultResolver)

// Register controllers
server.AddControllerType("blog", blog.NewBlogController(pool))
server.AddControllerType("auth", controller.NewAuthController)
server.AddControllerType("template", controller.NewTemplateController)
```

## Development

### Local Development (without Docker)

Requires: Go 1.25+, PostgreSQL, Redis, Vault

```bash
# Start dependencies
docker-compose up postgres redis vault

# Create secrets
mkdir -p secrets
echo "dev-secret" > secrets/SESSION_SECRET

# Run application
go run main.go
```

### Project Structure

```
blog_example/
├── blog/                 # Blog controller
│   └── blog.go
├── content/              # Static assets
│   ├── templates/        # HTML templates
│   │   ├── articles.html
│   │   └── admin.html
│   ├── css/             # Stylesheets
│   └── favicon.ico
├── secrets/             # Secret files (gitignored)
├── main.go              # Application entry point
├── config.docker.yaml   # Docker configuration
├── docker-compose.yml   # Service orchestration
└── Dockerfile           # Multi-stage build
```

## Troubleshooting

**Blog service won't start:**
- Check that Vault is healthy: `docker-compose ps`
- Verify secrets exist: `docker exec blog-vault vault kv get secret/blog`
- Check logs: `docker-compose logs blog`

**Authentication fails:**
- Verify mock OAuth2 server is running: `docker-compose ps mockoidc`
- Check mockoidc logs: `docker-compose logs mockoidc`
- Verify callback URL in config matches: `http://localhost:8080/auth/google/callback`

## What This Example Demonstrates

✅ **Modular Architecture**: Controllers, resolvers, and services are independently configured
✅ **OAuth2 Authentication**: Mock OAuth2 provider for easy development testing
✅ **Hybrid Secrets**: Mix file-based and Vault secrets in same configuration
✅ **Production Patterns**: Connection pooling, health checks, graceful shutdown
✅ **Template Rendering**: Server-side HTML with Go templates
✅ **Session Management**: Redis-backed sessions with authentication
✅ **Database Integration**: PostgreSQL with automatic schema creation and CRUD operations
✅ **Docker Best Practices**: Multi-stage builds, health checks, dependency ordering

## License

See the main project LICENSE file.
