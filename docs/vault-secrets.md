# Vault Secrets Integration

Sargantana Go now supports loading secrets from HashiCorp Vault in addition to file-based secrets. This document explains how to configure and use Vault for secret management.

## Configuration

Add the following Vault configuration to your YAML config file:

```yaml
vault:
  address: "https://vault.example.com:8200"  # Vault server address
  token: "${VAULT_TOKEN}"                     # Vault authentication token
  path: "secret/data/myapp"                   # Path to secrets in Vault
  namespace: ""                               # Optional: Vault namespace (for Enterprise)
```

### Configuration Fields

- **address**: The URL of your Vault server (required)
- **token**: Authentication token for Vault access (required)
- **path**: The path where your secrets are stored in Vault (required)
- **namespace**: Vault namespace for Enterprise installations (optional)

## Supported Vault Engines

The implementation supports both KV v1 and KV v2 secrets engines:

- **KV v1**: Secrets are read directly from the path
- **KV v2**: Secrets are automatically extracted from the nested `data` field

## Usage

### Loading Secrets

Use the new `LoadSecrets()` function to load secrets from both directory and Vault sources:

```go
import "github.com/animalet/sargantana-go/config"

// Load configuration from YAML
var cfg config.Config
err := config.LoadYaml("config.yaml", &cfg)
if err != nil {
    log.Fatal(err)
}

// Load secrets from both directory and Vault
err = config.LoadSecrets(&cfg)
if err != nil {
    log.Fatal(err)
}
```

### Secret Priority

When both directory and Vault secrets are configured:

1. Directory secrets are loaded first
2. Vault secrets are loaded second and will **override** directory secrets with the same name
3. All secret names are converted to uppercase environment variables

## Authentication

Currently supported authentication methods:

- **Token authentication**: Provide a Vault token directly in the configuration

### Using Environment Variables for Tokens

For security, avoid hardcoding tokens in configuration files. Instead, use environment variables:

```yaml
vault:
  token: "${VAULT_TOKEN}"
```

Then set the environment variable:
```bash
export VAULT_TOKEN="your-vault-token"
```

## Examples

### Basic Configuration

```yaml
# Minimal Vault configuration
vault:
  address: "https://vault.company.com:8200"
  token: "${VAULT_TOKEN}"
  path: "secret/data/myapp"
```

### Enterprise Configuration with Namespace

```yaml
# Vault Enterprise with namespace
vault:
  address: "https://vault.company.com:8200"
  token: "${VAULT_TOKEN}"
  path: "secret/data/production/myapp"
  namespace: "production"
```

### Combined Directory and Vault

```yaml
# Use both directory and Vault secrets
secrets_dir: "/run/secrets"
vault:
  address: "https://vault.company.com:8200"
  token: "${VAULT_TOKEN}"
  path: "secret/data/myapp"
```

## Error Handling

The Vault integration includes comprehensive error handling:

- **Missing configuration**: Vault loading is skipped if configuration is incomplete
- **Connection errors**: Network and authentication errors are properly reported
- **Path not found**: Non-existent paths are handled gracefully
- **Invalid data**: Non-string values in secrets are skipped with warnings

## Security Considerations

1. **Token Security**: Never commit Vault tokens to version control
2. **TLS**: Always use HTTPS for Vault communication in production
3. **Token Rotation**: Implement token rotation policies
4. **Least Privilege**: Grant minimal required permissions to Vault tokens
5. **Audit**: Enable Vault audit logging for security monitoring

## Troubleshooting

### Common Issues

1. **403 Permission Denied**: Check token permissions and path access
2. **Connection Refused**: Verify Vault server address and network connectivity
3. **Empty Secrets**: Verify the correct path format for your KV engine version
4. **SSL Errors**: Ensure proper TLS configuration for production Vault instances

### Debug Logging

Enable debug mode in your configuration to see detailed secret loading logs:

```yaml
debug: true
```

This will log the number of secrets loaded from each source.
