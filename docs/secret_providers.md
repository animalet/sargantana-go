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

**Important:** The secrets package is located at `pkg/config/secrets` and is decoupled from the main config package. It provides both the infrastructure (interfaces and registry) and built-in secret loader implementations. Your application must explicitly register the loaders it needs before loading configuration.

## Registering Secret Loaders

Secret loaders must be registered **before** calling `cfg.Load()`. Here's a typical setup in your main function:

```go
import (
    "github.com/animalet/sargantana-go/pkg/config"
    "github.com/animalet/sargantana-go/pkg/config/secrets"
)

func main() {
    // 1. Read the configuration file (without expanding secrets yet)
    // ReadModular returns a *config.Config struct with raw config data
    cfg, err := config.ReadModular("config.yaml")
    if err != nil {
        log.Fatal(err)
    }

    // 2. Register secret loaders
    // Environment loader is registered by default, but you can re-register it if needed
    secrets.Register("env", secrets.NewEnvLoader())

    // File loader (if file_resolver is configured)
    fileResolverCfg, err := config.Get[secrets.FileSecretConfig](cfg, "file_resolver")
    if err == nil && fileResolverCfg != nil {
        // CreateClient returns a SecretLoader
        fileLoader, err := fileResolverCfg.CreateClient()
        if err != nil {
            log.Fatal().Err(err).Msg("Failed to create file secret loader")
        }
        secrets.Register("file", fileLoader)
    }

    // Vault loader (if vault is configured)
    vaultCfg, err := config.Get[secrets.VaultConfig](cfg, "vault")
    if err == nil && vaultCfg != nil {
        vaultClient, err := vaultCfg.CreateClient()
        if err != nil {
            log.Fatal().Err(err).Msg("Failed to create Vault client")
        }
        // NewVaultSecretLoader adapts the client to the SecretLoader interface
        secrets.Register("vault", secrets.NewVaultSecretLoader(vaultClient, vaultCfg.Path))
    }

    // AWS Secrets Manager loader (if aws is configured)
    awsCfg, err := config.Get[secrets.AWSConfig](cfg, "aws")
    if err == nil && awsCfg != nil {
        awsClient, err := awsCfg.CreateClient()
        if err != nil {
            log.Fatal().Err(err).Msg("Failed to create AWS client")
        }
        secrets.Register("aws", secrets.NewAWSSecretLoader(awsClient, awsCfg.SecretName))
    }

    // 3. Now you can proceed to create the server
    // The server creation process will use the registered loaders to expand configuration
    srv, err := server.NewServer(cfg)
    // ...
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
fileSecretCfg, err := config.Get[secrets.FileSecretConfig](cfg, "file_resolver")
if err == nil {
    fileLoader, err := fileSecretCfg.CreateClient()
    // ... handle error ...
    secrets.Register("file", fileLoader)
}
```

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
vaultCfg, err := config.Get[secrets.VaultConfig](cfg, "vault")
if err == nil {
    vaultClient, err := vaultCfg.CreateClient()
    // ... handle error ...
    secrets.Register("vault", secrets.NewVaultSecretLoader(vaultClient, vaultCfg.Path))
}
```

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
awsCfg, err := config.Get[secrets.AWSConfig](cfg, "aws")
if err == nil {
    awsClient, err := awsCfg.CreateClient()
    // ... handle error ...
    secrets.Register("aws", secrets.NewAWSSecretLoader(awsClient, awsCfg.SecretName))
}
```

## Creating Custom Secret Providers

You can create custom secret providers to retrieve configuration from any source: databases, remote APIs, encrypted stores, etc.

### Step 1: Implement the SecretLoader Interface

```go
package mypackage

import (
    "github.com/animalet/sargantana-go/pkg/config/secrets"
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

func (c *CustomSecretProvider) Name() string {
    return "custom-api-provider"
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
    "github.com/animalet/sargantana-go/pkg/config/secrets"
    "mypackage"
)

func main() {
    // Register your custom secret provider with a prefix
    customProvider := mypackage.NewCustomSecretProvider(
        "https://api.example.com",
        "your-api-key",
    )
    secrets.Register("custom", customProvider)

    // ... proceed with server setup
}
```

### Step 3: Use Your Custom Secret Provider in Configuration

```yaml
server:
  session_secret: ${custom:SESSION_SECRET}
  api_key: ${custom:API_KEY}
```
