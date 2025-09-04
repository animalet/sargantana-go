# Vault Secrets Integration

Sargantana Go supports loading secrets from HashiCorp Vault. This document explains how to configure and use Vault for secret management.

## Configuration

To enable Vault integration, add the following section to your YAML configuration file:

```yaml
vault:
  address: "https://vault.example.com:8200"  # Vault server address
  token: "${VAULT_TOKEN}"                     # Vault authentication token
  path: "secret/data/myapp"                   # Path to secrets in Vault
  namespace: ""                               # Optional: Vault namespace (for Enterprise)
```

If the `vault` section is present and valid, the Vault client will be initialized at startup.

### Configuration Fields

-   **address**: The URL of your Vault server (required).
-   **token**: An authentication token for Vault access (required). It's recommended to supply this via an environment variable.
-   **path**: The path where your secrets are stored in Vault (required).
-   **namespace**: The Vault namespace for Enterprise installations (optional).

## Usage

To use a secret from Vault, use the `${vault:secret-name}` syntax in your configuration file. The application will automatically fetch the secret from the configured Vault path and substitute it.

For example, to use a database password stored in Vault:

```yaml
database:
  password: "${vault:db_password}"
```

When the configuration is loaded, `${vault:db_password}` will be replaced with the value of the `db_password` secret from Vault.

This works for any field in your configuration file.

### Supported Vault Engines

The implementation supports both KV v1 and KV v2 secrets engines:

-   **KV v1**: Secrets are read directly from the configured `path`.
-   **KV v2**: Secrets are automatically extracted from the `data` field within the response from the configured `path`.

## Authentication

Currently, the only supported authentication method is:

-   **Token authentication**: Provide a Vault token directly in the configuration.

### Using Environment Variables for Tokens

For security, avoid hardcoding tokens in configuration files. Instead, use environment variables with the `${VAR}` syntax:

```yaml
vault:
  token: "${VAULT_TOKEN}"
```

Then, set the environment variable in your shell:

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

# Example of using a Vault secret
server:
  session_secret: "${vault:session-secret-key}"
```

### Enterprise Configuration with Namespace

```yaml
# Vault Enterprise with a namespace
vault:
  address: "https://vault.company.com:8200"
  token: "${VAULT_TOKEN}"
  path: "secret/data/production/myapp"
  namespace: "my-namespace"

# Using vault secrets in configuration
server:
  session_secret: "${vault:session-secret}"
  
controllers:
  - type: "auth"
    config:
      providers:
        github:
          key: "${vault:github-client-id}"
          secret: "${vault:github-client-secret}"
```

### Multiple Secret Types

```yaml
vault:
  address: "https://vault.company.com:8200"
  token: "${VAULT_TOKEN}"
  path: "secret/data/myapp"

server:
  # Vault secret
  session_secret: "${vault:session-secret}"
  # Environment variable
  address: "${SERVER_ADDRESS}"
  # File secret (Docker secrets)
  redis_session_store:
    password: "${file:redis_password}"
```
