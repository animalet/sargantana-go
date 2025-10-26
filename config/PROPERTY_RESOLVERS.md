# Property Resolvers

Property Resolvers provide an extensible mechanism for retrieving configuration values from various sources. This document explains how to use the built-in resolvers and how to create custom ones.

## Overview

The property resolver system allows you to reference secrets and configuration values using a prefix-based syntax in your YAML configuration files:

```yaml
server:
  session_secret: ${vault:SESSION_SECRET}    # From HashiCorp Vault
  api_key: ${file:api_key}                   # From file in secrets directory
  port: ${env:PORT}                          # From environment variable
  host: ${DATABASE_HOST}                     # Defaults to env: prefix
```

## Built-in Resolvers

### Environment Variable Resolver (env:)

Retrieves values from environment variables. This is the default resolver when no prefix is specified.

**Usage:**
```yaml
# Explicit prefix
database_host: ${env:DATABASE_HOST}

# Implicit (defaults to env:)
database_host: ${DATABASE_HOST}
```

**Configuration:** No configuration needed - always available.

### File Resolver (file:)

Reads secrets from files in a configured directory. Useful for Docker secrets, Kubernetes secrets, or local development.

**Usage:**
```yaml
server:
  secrets_dir: "/run/secrets"  # Configure the directory
  api_key: ${file:api_key}     # Reads from /run/secrets/api_key
```

**Configuration:** Set `server.secrets_dir` in your config to enable file-based secrets.

**File Format:** Files should contain the secret value as plain text. Whitespace is automatically trimmed.

### Vault Resolver (vault:)

Retrieves secrets from HashiCorp Vault. Supports both KV v1 and KV v2 secret engines.

**Usage:**
```yaml
vault:
  address: "http://localhost:8200"
  token: "${VAULT_TOKEN}"              # Can use env vars in vault config!
  path: "secret/data/myapp"
  namespace: "my-namespace"            # Optional

server:
  session_secret: ${vault:SESSION_SECRET}
  database_password: ${vault:DB_PASSWORD}
```

**Configuration:** Configure the `vault` section in your config file with:
- `address`: Vault server URL
- `token`: Authentication token
- `path`: Path to read secrets from (e.g., "secret/data/myapp" for KV v2)
- `namespace`: Optional Vault namespace

## Creating Custom Resolvers

You can create custom resolvers to retrieve configuration from any source: databases, remote APIs, encrypted stores, etc.

### Step 1: Implement the PropertyResolver Interface

```go
package mypackage

import (
    "github.com/animalet/sargantana-go/config"
    "github.com/pkg/errors"
)

// CustomResolver retrieves properties from your custom source
type CustomResolver struct {
    apiEndpoint string
    apiKey      string
}

// NewCustomResolver creates a new instance
func NewCustomResolver(endpoint, apiKey string) *CustomResolver {
    return &CustomResolver{
        apiEndpoint: endpoint,
        apiKey:      apiKey,
    }
}

// Resolve retrieves a property value
func (c *CustomResolver) Resolve(key string) (string, error) {
    // Your custom logic here
    value, err := c.fetchFromAPI(key)
    if err != nil {
        return "", errors.Wrapf(err, "failed to retrieve %q from custom source", key)
    }
    return value, nil
}

// Name returns the resolver name (for logging/debugging)
func (c *CustomResolver) Name() string {
    return "Custom API"
}

func (c *CustomResolver) fetchFromAPI(key string) (string, error) {
    // Implement your API call logic here
    // ...
    return "value", nil
}
```

### Step 2: Register Your Resolver

Register your custom resolver before loading the configuration:

```go
package main

import (
    "github.com/animalet/sargantana-go/config"
    "mypackage"
)

func main() {
    // Register your custom resolver with a prefix
    customResolver := mypackage.NewCustomResolver(
        "https://api.example.com",
        "your-api-key",
    )
    config.RegisterPropertyResolver("custom", customResolver)

    // Now load your configuration
    cfg, err := config.ReadConfig("config.yaml")
    if err != nil {
        panic(err)
    }

    err = cfg.Load()
    if err != nil {
        panic(err)
    }

    // Your custom resolver is now available!
}
```

### Step 3: Use Your Custom Resolver in Configuration

```yaml
server:
  session_secret: ${custom:SESSION_SECRET}
  api_key: ${custom:API_KEY}
```

## Advanced Examples

### Database Resolver

Retrieve configuration from a database:

```go
type DatabaseResolver struct {
    db *sql.DB
    tableName string
}

func NewDatabaseResolver(db *sql.DB, tableName string) *DatabaseResolver {
    return &DatabaseResolver{db: db, tableName: tableName}
}

func (d *DatabaseResolver) Resolve(key string) (string, error) {
    var value string
    query := fmt.Sprintf("SELECT value FROM %s WHERE key = $1", d.tableName)
    err := d.db.QueryRow(query, key).Scan(&value)
    if err != nil {
        return "", errors.Wrapf(err, "failed to retrieve %q from database", key)
    }
    return value, nil
}

func (d *DatabaseResolver) Name() string {
    return "Database"
}
```

### AWS Secrets Manager Resolver

```go
import (
    "github.com/aws/aws-sdk-go/aws/session"
    "github.com/aws/aws-sdk-go/service/secretsmanager"
)

type AWSSecretsResolver struct {
    client *secretsmanager.SecretsManager
}

func NewAWSSecretsResolver(region string) *AWSSecretsResolver {
    sess := session.Must(session.NewSession())
    return &AWSSecretsResolver{
        client: secretsmanager.New(sess),
    }
}

func (a *AWSSecretsResolver) Resolve(key string) (string, error) {
    input := &secretsmanager.GetSecretValueInput{
        SecretId: aws.String(key),
    }

    result, err := a.client.GetSecretValue(input)
    if err != nil {
        return "", errors.Wrapf(err, "failed to retrieve %q from AWS Secrets Manager", key)
    }

    return *result.SecretString, nil
}

func (a *AWSSecretsResolver) Name() string {
    return "AWS Secrets Manager"
}
```

Usage:
```go
config.RegisterPropertyResolver("aws", NewAWSSecretsResolver("us-east-1"))
```

Configuration:
```yaml
server:
  database_password: ${aws:prod/db/password}
```

## Error Handling

When a resolver fails to retrieve a value:
1. The resolver should return a descriptive error
2. The `expand()` function will panic with the error wrapped
3. This typically happens during config loading, causing startup to fail

This fail-fast behavior ensures your application doesn't start with missing configuration.

## Thread Safety

The resolver registry is thread-safe using `sync.RWMutex`. You can:
- Register resolvers from multiple goroutines
- Resolve properties concurrently during configuration expansion

However, it's recommended to register all resolvers during application initialization before starting concurrent operations.

## Best Practices

1. **Register resolvers early**: Register all custom resolvers before calling `cfg.Load()`

2. **Use descriptive prefixes**: Choose short, memorable prefix names (e.g., "db", "aws", "consul")

3. **Handle errors gracefully**: Return clear error messages from your `Resolve()` implementation

4. **Log resolver activity**: Consider logging successful retrievals for debugging (see built-in resolvers for examples)

5. **Secure sensitive data**: If your resolver connects to remote services, ensure proper authentication and encryption

6. **Test thoroughly**: Write tests for your custom resolvers, including error cases

7. **Document configuration**: Document what configuration your resolver needs (connection strings, credentials, etc.)

## Unregistering Resolvers

You can unregister a resolver if needed:

```go
config.UnregisterPropertyResolver("custom")
```

This is primarily useful in tests or when dynamically managing resolvers.

## Architecture Notes

The property resolver system uses:
- **PropertyResolver interface**: Contract for all resolvers
- **PropertyResolverRegistry**: Thread-safe registry mapping prefixes to resolvers
- **Global registry**: `globalResolverRegistry` used by the config system
- **Parser**: `parseProperty()` splits "prefix:key" (defaults to "env:" if no prefix)
- **Expansion**: `expand()` function called by Go's `os.Expand()` during config loading

The system integrates seamlessly with Go's standard `os.Expand()` function, which processes `${...}` placeholders in configuration strings.
