# Property Resolvers

Property Resolvers provide an extensible mechanism for retrieving configuration values from various sources. This document explains how to use the built-in resolvers and how to create custom ones.

## Overview

The property resolver system allows you to reference secrets and configuration values using a prefix-based syntax in your YAML configuration files:

```yaml
server:
  session_secret: ${vault:SESSION_SECRET}    # From HashiCorp Vault
  api_key: ${file:api_key}                   # From file in secrets directory
  database_password: ${aws:DB_PASSWORD}      # From AWS Secrets Manager
  port: ${env:PORT}                          # From environment variable
  host: ${DATABASE_HOST}                     # Defaults to env: prefix
```

**Important:** The resolver package is decoupled from config and provides both the infrastructure (interfaces and registry) and built-in resolver implementations. Your application must explicitly register the resolvers it needs before loading configuration.

## Registering Resolvers

Resolvers must be registered **before** calling `cfg.Load()`. Here's a typical setup in your main function:

```go
func main() {
    // Read the configuration file
    cfg, err := config.ReadConfig("config.yaml")
    if err != nil {
        log.Fatal(err)
    }

    // Register resolvers BEFORE calling Load()
    // Environment resolver (default - always register first)
    resolver.Register("env", resolver.NewEnvResolver())

    // File resolver (if file_resolver is configured)
    fileResolverCfg, err := config.LoadConfig[resolver.FileResolverConfig]("file_resolver", cfg)
    if err == nil {
        fileResolver, err := fileResolverCfg.CreateClient()
        if err != nil {
            log.Fatal().Err(err).Msg("Failed to create file resolver")
        }
        resolver.Register("file", fileResolver)
        log.Info().Str("secrets_dir", fileResolverCfg.SecretsDir).Msg("File resolver registered")
    }

    // Vault resolver (if vault is configured)
    vaultCfg, err := config.LoadConfig[resolver.VaultConfig]("vault", cfg)
    if err == nil {
        vaultClient, err := vaultCfg.CreateClient()
        if err != nil {
            log.Fatal().Err(err).Msg("Failed to create Vault client")
        }
        resolver.Register("vault", resolver.NewVaultResolver(vaultClient, vaultCfg.Path))
        log.Info().Msg("Vault resolver registered")
    }

    // AWS Secrets Manager resolver (if aws is configured)
    awsCfg, err := config.LoadConfig[resolver.AWSConfig]("aws", cfg)
    if err == nil {
        awsClient, err := awsCfg.CreateClient()
        if err != nil {
            log.Fatal().Err(err).Msg("Failed to create AWS client")
        }
        resolver.Register("aws", resolver.NewAWSResolver(awsClient, awsCfg.SecretName))
        log.Info().Msg("AWS resolver registered")
    }

    // Now load and expand the configuration
    err = cfg.Load()
    if err != nil {
        log.Fatal(err)
    }

    // Continue with server setup...
}
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

**Registration:**
```go
resolver.Register("env", resolver.NewEnvResolver())
```

**Configuration:** No additional configuration needed.

### File Resolver (file:)

Reads secrets from files in a configured directory. Useful for Docker secrets, Kubernetes secrets, or local development.

**Usage:**
```yaml
file_resolver:
  secrets_dir: "/run/secrets"  # Configure the directory

server:
  api_key: ${file:api_key}     # Reads from /run/secrets/api_key
  db_password: ${file:db_password}
```

**Registration:**
```go
fileResolverCfg, err := config.LoadConfig[resolver.FileResolverConfig]("file_resolver", cfg)
if err == nil {
    fileResolver, err := fileResolverCfg.CreateClient()
    if err != nil {
        log.Fatal().Err(err).Msg("Failed to create file resolver")
    }
    resolver.Register("file", fileResolver)
}
```

**Configuration:** Add a `file_resolver` section to your YAML config:
- `secrets_dir`: Directory containing secret files (required)

**Validation:** The file resolver validates that:
- The `secrets_dir` is not empty
- The directory exists
- The path is actually a directory (not a file)

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

**Registration:**
```go
vaultCfg, err := config.LoadConfig[resolver.VaultConfig]("vault", cfg)
if err == nil {
    vaultClient, err := vaultCfg.CreateClient()
    if err != nil {
        log.Fatal().Err(err).Msg("Failed to create Vault client")
    }
    resolver.Register("vault", resolver.NewVaultResolver(vaultClient, vaultCfg.Path))
}
```

**Configuration:** Configure the `vault` section in your YAML config file with:
- `address`: Vault server URL
- `token`: Authentication token
- `path`: Path to read secrets from (e.g., "secret/data/myapp" for KV v2)
- `namespace`: Optional Vault namespace

### AWS Secrets Manager Resolver (aws:)

Retrieves secrets from AWS Secrets Manager. Supports both JSON-formatted secrets (with multiple key-value pairs) and plain text secrets.

**Usage:**
```yaml
aws:
  region: "us-east-1"
  access_key_id: "${AWS_ACCESS_KEY_ID}"    # Optional - uses IAM role if not provided
  secret_access_key: "${AWS_SECRET_ACCESS_KEY}"  # Optional
  secret_name: "myapp/prod"
  endpoint: "http://localhost:4566"         # Optional - for LocalStack or custom endpoints

server:
  session_secret: ${aws:SESSION_SECRET}      # From JSON secret
  api_key: ${aws:API_KEY}                    # From JSON secret
```

**Registration:**
```go
awsCfg, err := config.LoadConfig[resolver.AWSConfig]("aws", cfg)
if err == nil {
    awsClient, err := awsCfg.CreateClient()
    if err != nil {
        log.Fatal().Err(err).Msg("Failed to create AWS client")
    }
    resolver.Register("aws", resolver.NewAWSResolver(awsClient, awsCfg.SecretName))
}
```

**Configuration:** Configure the `aws` section in your YAML config file with:
- `region`: AWS region (required)
- `access_key_id`: AWS access key (optional - uses default credential chain if not provided)
- `secret_access_key`: AWS secret key (optional - uses default credential chain if not provided)
- `secret_name`: Name of the secret in AWS Secrets Manager (required)
- `endpoint`: Custom endpoint URL (optional - useful for LocalStack or custom endpoints)

**Secret Formats:**
1. **JSON format** (multiple key-value pairs):
   ```json
   {
     "SESSION_SECRET": "my-session-secret",
     "API_KEY": "my-api-key",
     "DB_PASSWORD": "my-db-password"
   }
   ```
   Use `${aws:SESSION_SECRET}` to retrieve individual keys.

2. **Plain text format** (single value):
   ```
   my-secret-value
   ```
   For plain text secrets, the key parameter is ignored and the entire secret value is returned.

**IAM Credentials:** If `access_key_id` and `secret_access_key` are not provided, the resolver will use the AWS default credential chain (IAM role, environment variables, shared credentials file, etc.)

## Creating Custom Resolvers

You can create custom resolvers to retrieve configuration from any source: databases, remote APIs, encrypted stores, etc.

### Step 1: Implement the PropertyResolver Interface

```go
package mypackage

import (
    "github.com/animalet/sargantana-go/pkg/resolver"
    "github.com/pkg/errors"
)

// CustomResolver retrieves properties from your custom source
type CustomResolver struct {
    apiEndpoint string
    apiKey      string
}

// NewCustomResolver creates a new instance
func NewCustomResolver(endpoint, apiKey string) resolver.PropertyResolver {
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
    "github.com/animalet/sargantana-go/pkg/config"
    "github.com/animalet/sargantana-go/pkg/resolver"
    "mypackage"
)

func main() {
    // Register your custom resolver with a prefix
    customResolver := mypackage.NewCustomResolver(
        "https://api.example.com",
        "your-api-key",
    )
    resolver.Register("custom", customResolver)

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

func NewDatabaseResolver(db *sql.DB, tableName string) resolver.PropertyResolver {
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

1. **Register resolvers before Load()**: ALWAYS register all resolvers before calling `cfg.Load()`. The config package does not automatically register any resolvers.

2. **Register env resolver first**: Register the environment resolver first, as it's the default when no prefix is specified.

3. **Use descriptive prefixes**: Choose short, memorable prefix names (e.g., "db", "aws", "consul")

4. **Handle errors gracefully**: Return clear error messages from your `Resolve()` implementation

5. **Log resolver activity**: Consider logging successful retrievals for debugging (see built-in resolvers for examples)

6. **Secure sensitive data**: If your resolver connects to remote services, ensure proper authentication and encryption

7. **Test thoroughly**: Write tests for your custom resolvers, including error cases. Remember to register resolvers in your tests!

8. **Document configuration**: Document what configuration your resolver needs (connection strings, credentials, etc.)

9. **Decoupled design**: The config package only provides infrastructure. Your application controls which resolvers are available.

## Unregistering Resolvers

You can unregister a resolver if needed:

```go
resolver.Unregister("custom")
```

This is primarily useful in tests or when dynamically managing resolvers.

## Modular Configuration Pattern

All resolver configurations (except the environment resolver) use the modular `LoadConfig[T]()` pattern:

```yaml
# Optional resolver configurations
file_resolver:
  secrets_dir: "./secrets"

vault:
  address: "http://localhost:8200"
  token: "${env:VAULT_TOKEN}"
  path: "secret/data/myapp"

aws:
  region: "us-east-1"
  secret_name: "myapp/prod"
```

**Key principles:**
1. **Optional by design**: If a resolver section is not present in the config, it's simply not loaded
2. **Type-safe**: Each resolver config implements `Validatable` interface
3. **ClientFactory pattern**: Each config has a `CreateClient()` method that returns the typed client
4. **Consistent loading**: All use `config.LoadConfig[T]("key", cfg)` pattern
5. **Validation**: Configs are validated before client creation

**Example configuration types:**
- `resolver.FileResolverConfig` - File-based secrets
- `resolver.VaultConfig` - HashiCorp Vault
- `resolver.AWSConfig` - AWS Secrets Manager

This pattern ensures the core `ServerConfig` remains minimal and focused only on essential server settings (address, session name/secret), while optional components are loaded modularly.

## Architecture Notes

The property resolver system is located in the `pkg/resolver/` package and uses:
- **PropertyResolver interface**: Contract for all resolvers (`resolver` package)
- **PropertyResolverRegistry**: Thread-safe registry mapping prefixes to resolvers (`resolver` package)
- **Global registry**: `Global()` instance used by the config system
- **Parser**: Splits "prefix:key" syntax (defaults to "env:" if no prefix)
- **Expansion**: Integrated with Go's `os.Expand()` during config loading

The system is fully decoupled from the config package:
- The `config` package calls `resolver.Global().Resolve()` to resolve properties
- The `resolver` package contains all resolver implementations
- The `config` package has no knowledge of specific resolvers
- Applications control which resolvers are available by registering them
