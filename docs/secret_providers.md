# Secret Loaders

Secret Loaders offer an extensible mechanism for retrieving configuration values and secrets from various sources. This document explains how to use the built-in loaders and how to create custom ones.

## Overview

The secrets resolution system allows you to reference secrets and configuration values using a prefix-based syntax in your YAML configuration files:

```yaml
server:
  session_secret: ${vault:SESSION_SECRET}    # From HashiCorp Vault
  api_key: ${file:api_key}                   # From file in secrets directory
  database_password: ${aws:DB_PASSWORD}      # From AWS Secrets Manager
  port: ${env:PORT}                          # From environment variable
  host: ${DATABASE_HOST}                     # Defaults to env: prefix
```

**Important:** The secrets package is decoupled from config and provides both the infrastructure (interfaces and registry) and built-in secret loader implementations. Your application must explicitly register the loaders it needs before loading configuration.

## Registering Secret Loaders

Secret loaders must be registered **before** calling `cfg.Load()`. Here's a typical setup in your main function:

```go
func main() {
    // Read the configuration file
    cfg, err := config.ReadConfig("config.yaml")
    if err != nil {
        log.Fatal(err)
    }

    // Register secret loaders BEFORE calling Load()
    // Environment loader (default - always register first)
    secrets.Register("env", secrets.NewEnvLoader())

    // File loader (if file_resolver is configured)
    fileResolverCfg, err := config.LoadConfig[secrets.FileSecretConfig]("file_resolver", cfg)
    if err == nil {
        fileLoader, err := fileResolverCfg.CreateClient()
        if err != nil {
            log.Fatal().Err(err).Msg("Failed to create file secret loader")
        }
        secrets.Register("file", fileLoader)
        log.Info().Str("secrets_dir", fileResolverCfg.SecretsDir).Msg("File secret loader registered")
    }

    // Vault loader (if vault is configured)
    vaultCfg, err := config.LoadConfig[secrets.VaultConfig]("vault", cfg)
    if err == nil {
        vaultClient, err := vaultCfg.CreateClient()
        if err != nil {
            log.Fatal().Err(err).Msg("Failed to create Vault client")
        }
        secrets.Register("vault", secrets.NewVaultSecretLoader(vaultClient, vaultCfg.Path))
        log.Info().Msg("Vault secret loader registered")
    }

    // AWS Secrets Manager loader (if aws is configured)
    awsCfg, err := config.LoadConfig[secrets.AWSConfig]("aws", cfg)
    if err == nil {
        awsClient, err := awsCfg.CreateClient()
        if err != nil {
            log.Fatal().Err(err).Msg("Failed to create AWS client")
        }
        secrets.Register("aws", secrets.NewAWSSecretLoader(awsClient, awsCfg.SecretName))
        log.Info().Msg("AWS secret loader registered")
    }

    // Now load and expand the configuration
    err = cfg.Load()
    if err != nil {
        log.Fatal(err)
    }

    // Continue with server setup...
}
```

## Built-in Secret Loaders

### Environment Variable Loader (env:)

Retrieves values from environment variables. This is the default loader when no prefix is specified.

**Type:** `EnvLoader`

**Usage:**
```yaml
# Explicit prefix
database_host: ${env:DATABASE_HOST}

# Implicit (defaults to env:)
database_host: ${DATABASE_HOST}
```

**Registration:**
```go
secrets.Register("env", secrets.NewEnvLoader())
```

**Configuration:** No additional configuration needed.

### File Loader (file:)

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
fileSecretCfg, err := config.LoadConfig[secrets.FileSecretConfig]("file_resolver", cfg)
if err == nil {
    fileLoader, err := fileSecretCfg.CreateClient()
    if err != nil {
        log.Fatal().Err(err).Msg("Failed to create file secret loader")
    }
    secrets.Register("file", fileLoader)
}
```

**Configuration:** Add a `file_resolver` section to your YAML config:
- `secrets_dir`: Directory containing secret files (required)

**Validation:** The file provider validates that:
- The `secrets_dir` is not empty
- The directory exists
- The path is actually a directory (not a file)

**File Format:** Files should contain the secret value as plain text. Whitespace is automatically trimmed.

### Vault Loader (vault:)

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
vaultCfg, err := config.LoadConfig[secrets.VaultConfig]("vault", cfg)
if err == nil {
    vaultClient, err := vaultCfg.CreateClient()
    if err != nil {
        log.Fatal().Err(err).Msg("Failed to create Vault client")
    }
    secrets.Register("vault", secrets.NewVaultSecretLoader(vaultClient, vaultCfg.Path))
}
```

**Configuration:** Configure the `vault` section in your YAML config file with:
- `address`: Vault server URL
- `token`: Authentication token
- `path`: Path to read secrets from (e.g., "secret/data/myapp" for KV v2)
- `namespace`: Optional Vault namespace

### AWS Secrets Manager Loader (aws:)

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
awsCfg, err := config.LoadConfig[secrets.AWSConfig]("aws", cfg)
if err == nil {
    awsClient, err := awsCfg.CreateClient()
    if err != nil {
        log.Fatal().Err(err).Msg("Failed to create AWS client")
    }
    secrets.Register("aws", secrets.NewAWSSecretLoader(awsClient, awsCfg.SecretName))
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

**IAM Credentials:** If `access_key_id` and `secret_access_key` are not provided, the provider will use the AWS default credential chain (IAM role, environment variables, shared credentials file, etc.)

## Creating Custom Secret Providers

You can create custom secret providers to retrieve configuration from any source: databases, remote APIs, encrypted stores, etc.

### Step 1: Implement the SecretLoader Interface

```go
package mypackage

import (
    "github.com/animalet/sargantana-go/pkg/secrets"
    "github.com/pkg/errors"
)

// CustomSecretProvider retrieves secrets from your custom source
type CustomSecretProvider struct {
    apiEndpoint string
    apiKey      string
}

// NewCustomSecretProvider creates a new instance
func NewCustomSecretProvider(endpoint, apiKey string) secrets.SecretLoader {
    return &CustomSecretProvider{
        apiEndpoint: endpoint,
        apiKey:      apiKey,
    }
}

// Resolve retrieves a secret value
func (c *CustomSecretProvider) Resolve(key string) (string, error) {
    // Your custom logic here
    value, err := c.fetchFromAPI(key)
    if err != nil {
        return "", errors.Wrapf(err, "failed to retrieve %q from custom source", key)
    }
    return value, nil
}

func (c *CustomSecretProvider) fetchFromAPI(key string) (string, error) {
    // Implement your API call logic here
    // ...
    return "value", nil
}
```

### Step 2: Register Your Secret Provider

Register your custom secret provider before loading the configuration:

```go
package main

import (
    "github.com/animalet/sargantana-go/pkg/config"
    "github.com/animalet/sargantana-go/pkg/secrets"
    "mypackage"
)

func main() {
    // Register your custom secret provider with a prefix
    customProvider := mypackage.NewCustomSecretProvider(
        "https://api.example.com",
        "your-api-key",
    )
    secrets.Register("custom", customProvider)

    // Now load your configuration
    cfg, err := config.ReadConfig("config.yaml")
    if err != nil {
        panic(err)
    }

    err = cfg.Load()
    if err != nil {
        panic(err)
    }

    // Your custom secret provider is now available!
}
```

### Step 3: Use Your Custom Secret Provider in Configuration

```yaml
server:
  session_secret: ${custom:SESSION_SECRET}
  api_key: ${custom:API_KEY}
```

## Advanced Examples

### Database Secret Provider

Retrieve configuration from a database:

```go
type DatabaseSecretProvider struct {
    db *sql.DB
    tableName string
}

func NewDatabaseSecretProvider(db *sql.DB, tableName string) secrets.SecretLoader {
    return &DatabaseSecretProvider{db: db, tableName: tableName}
}

func (d *DatabaseSecretProvider) Resolve(key string) (string, error) {
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

When a secret provider fails to retrieve a value:
1. The provider should return a descriptive error
2. The `expand()` function will panic with the error wrapped
3. This typically happens during config loading, causing startup to fail

This fail-fast behavior ensures your application doesn't start with missing configuration.

## Thread Safety

The secrets registry is thread-safe using `sync.RWMutex`. You can:
- Register providers from multiple goroutines
- Resolve secrets concurrently during configuration expansion

However, it's recommended to register all providers during application initialization before starting concurrent operations.

## Best Practices

1. **Register providers before Load()**: ALWAYS register all secret providers before calling `cfg.Load()`. The config package does not automatically register any providers.

2. **Register env provider first**: Register the environment provider first, as it's the default when no prefix is specified.

3. **Use descriptive prefixes**: Choose short, memorable prefix names (e.g., "db", "aws", "consul")

4. **Handle errors gracefully**: Return clear error messages from your `Resolve()` implementation

5. **Log provider activity**: Consider logging successful retrievals for debugging (see built-in providers for examples)

6. **Secure sensitive data**: If your provider connects to remote services, ensure proper authentication and encryption

7. **Test thoroughly**: Write tests for your custom providers, including error cases. Remember to register providers in your tests!

8. **Document configuration**: Document what configuration your provider needs (connection strings, credentials, etc.)

9. **Decoupled design**: The config package only provides infrastructure. Your application controls which providers are available.

## Unregistering Secret Providers

You can unregister a secret provider if needed:

```go
secrets.Unregister("custom")
```

This is primarily useful in tests or when dynamically managing providers.

## Modular Configuration Pattern

All secret provider configurations (except the environment provider) use the modular `LoadConfig[T]()` pattern:

```yaml
# Optional secret provider configurations
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
1. **Optional by design**: If a provider section is not present in the config, it's simply not loaded
2. **Type-safe**: Each provider config implements `Validatable` interface
3. **ClientFactory pattern**: Each config has a `CreateClient()` method that returns the typed client
4. **Consistent loading**: All use `config.LoadConfig[T]("key", cfg)` pattern
5. **Validation**: Configs are validated before client creation

**Example configuration types:**
- `secrets.FileSecretConfig` - File-based secrets
- `secrets.VaultConfig` - HashiCorp Vault
- `secrets.AWSConfig` - AWS Secrets Manager

This pattern ensures the core `ServerConfig` remains minimal and focused only on essential server settings (address, session name/secret), while optional components are loaded modularly.

## Architecture Notes

The secrets resolution system is located in the `pkg/secrets/` package and uses:
- **SecretLoader interface**: Contract for all secret providers (`secrets` package)
- **Registry**: Thread-safe registry mapping prefixes to providers (`secrets` package)
- **Global functions**: `Register()`, `Resolve()` functions used by the config system
- **Parser**: Splits "prefix:key" syntax (defaults to "env:" if no prefix)
- **Expansion**: Integrated with Go's `os.Expand()` during config loading

The system is fully decoupled from the config package:
- The `config` package calls `secrets.Resolve()` to resolve properties
- The `secrets` package contains all provider implementations
- The `config` package has no knowledge of specific providers
- Applications control which providers are available by registering them
