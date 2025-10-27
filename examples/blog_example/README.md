# Blog Example - Sargantana Go

A complete blog application demonstrating the Sargantana Go framework with authentication, database integration, and secrets management.

## Features

- **Authentication**: Keycloak OAuth2/OIDC for local development
- **Database**: PostgreSQL with connection pooling
- **Sessions**: Redis-backed session storage (unlimited size for OAuth tokens)
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
   - Keycloak Admin: http://localhost:8081 (admin/admin)
   - Vault UI: http://localhost:8200 (token: dev-root-token)

### Authentication

** Keycloak Login**
- Pre-configured test user: `test` / `test`
- Click "Login with OAuth" on the blog
- Redirects to Keycloak at http://localhost:8081
- After login, returns to blog with authenticated session
- Full OAuth2/OIDC flow for realistic development testing

## Architecture

### Services

| Service | Port | Purpose |
|---------|------|---------|
| **blog** | 8080 | Main application |
| **postgres** | 5432 | Database |
| **redis** | 6379 | Session storage |
| **vault** | 8200 | Secrets management |
| **keycloak** | 8081 | OAuth2/OIDC provider |

### Configuration

The example demonstrates **modular configuration** with multiple secret sources:

```yaml
# Hybrid secret sources (file and Vault)
session_secret: "${vault:SESSION_SECRET}"        # From Vault
user: "${vault:DB_USER}"                         # From Vault
password: "${vault:DB_PASSWORD}"                 # From Vault
key: "${file:OPENID_CONNECT_KEY}"                # From secrets/OPENID_CONNECT_KEY
secret: "${vault:OPENID_CONNECT_SECRET}"         # From Vault
```

**Key files:**
- `config.docker.yaml` - Docker-specific configuration
- `main.go` - Application entry point showing resolver setup
- `blog/blog.go` - Blog controller implementation
- `content/templates/` - HTML templates

## Technical Details

### Secrets Management

**Hybrid approach** mixing file-based and Vault secrets:

**Vault Secrets** (`secret/data/blog`) - auto-initialized on startup:
- `SESSION_SECRET` - Session encryption key
- `DB_USER` - PostgreSQL username
- `DB_PASSWORD` - PostgreSQL password
- `OPENID_CONNECT_SECRET` - OAuth client secret

**File Secrets** (`secrets/` directory):
- `OPENID_CONNECT_KEY` - OAuth client ID (value: `sargantana`)

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

1. User clicks "Login with OAuth" on blog
2. Redirected to Keycloak at `http://localhost:8081`
3. User enters credentials (`test` / `test`)
4. Keycloak validates and redirects back with authorization code
5. Blog exchanges code for access token
6. Session created in Redis with OAuth data
7. User redirected to `/blog/feed` as authenticated

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
- Check all services are healthy: `docker-compose ps`
- Verify Vault secrets exist: `docker exec blog-vault vault kv get secret/blog`
- Verify file secret exists: `cat secrets/OPENID_CONNECT_KEY`
- Check logs: `docker-compose logs blog`

**Authentication fails:**
- Verify Keycloak is running: `docker-compose ps keycloak`
- Check Keycloak logs: `docker-compose logs keycloak`
- Verify test user exists in Keycloak Admin UI (http://localhost:8081)
- Check callback URL matches: `http://localhost:8080/auth/openid-connect/callback`

**Session "too big" error:**
- Redis session store should be configured automatically
- Check blog logs for "Redis session store configured"
- Verify Redis is healthy: `docker-compose ps redis`

## What This Example Demonstrates

- **Modular Architecture**: Controllers, resolvers, and services are independently configured
- **OAuth2/OIDC Authentication**: Full OAuth2 flow integration
- **Hybrid Secrets**: Mix file-based and Vault secrets in same configuration
- **Production Patterns**: Connection pooling, graceful shutdown
- **Template Rendering**: Server-side HTML with Go templates
- **Session Management**: Redis-backed sessions.
- **Database Integration**: PostgreSQL with automatic schema creation and CRUD operations

## License

See the main project LICENSE file.
