# Security Policy

## Secret Handling

Sargantana Go provides a pluggable property resolver system for securely loading secrets from various sources (environment variables, files, HashiCorp Vault, AWS Secrets Manager). This document outlines security considerations and best practices.

## Logging and Secret Exposure

### Production (Release Mode)

When running in **release mode** (default), the framework uses **Info** log level:

- ✅ **Secret values are NEVER logged**
- ✅ **Secret metadata (names, keys, paths) is NOT logged**
- ✅ Secret access only logs at **Debug** level, which is disabled in production

### Development (Debug Mode)

When running in **debug mode** (`server.SetDebug(true)`), the framework uses **Debug** log level:

- ⚠️ **Secret metadata IS logged** (secret names, keys, Vault paths, file paths)
- ✅ **Secret values are NEVER logged** (even in debug mode)
- ⚠️ Logs may reveal which secrets your application uses

**Recommendation:** Never enable debug mode in production environments.

### Example Log Output

**Release Mode (Info level):**
```
2025-10-27 10:00:00 INF Configuration loaded successfully
2025-10-27 10:00:01 INF Bootstrapping server...
```

**Debug Mode:**
```
2025-10-27 10:00:00 INF Configuration loaded successfully
2025-10-27 10:00:01 DBG Retrieved secret from Vault secret_name=DATABASE_PASSWORD vault_path=secret/data/myapp
2025-10-27 10:00:01 DBG Retrieved secret from file file=.secrets/API_KEY
2025-10-27 10:00:01 INF Bootstrapping server...
```

## Secret Sources

### 1. Environment Variables (`env:`)

```yaml
database_password: ${env:DB_PASSWORD}  # or ${DB_PASSWORD}
```

**Security Notes:**
- Environment variables are visible to all processes running under the same user
- They appear in process listings (`ps aux`)
- Consider using more secure backends for sensitive secrets

### 2. File-Based Secrets (`file:`)

```yaml
server:
  secrets_dir: ".secrets"  # or /run/secrets for Docker

config:
  api_key: ${file:API_KEY}
```

**Security Notes:**
- Files must have restrictive permissions (e.g., `chmod 600`)
- Ideal for Docker secrets or Kubernetes mounted secrets
- Files are read once at startup (no rotation without restart)

### 3. HashiCorp Vault (`vault:`)

```yaml
vault:
  address: "https://vault.example.com"
  token: "${VAULT_TOKEN}"
  path: "secret/data/myapp"

config:
  database_password: ${vault:DATABASE_PASSWORD}
```

**Security Notes:**
- Supports both KV v1 and KV v2 secret engines
- Token should be provided via environment variable, not hardcoded
- Consider using Vault's auto-renewal and lease management
- Secrets are cached in memory after first read

### 4. AWS Secrets Manager (`aws:`)

```yaml
aws:
  region: "us-east-1"
  secret_name: "myapp/production"
  # Uses IAM role credentials by default

config:
  database_password: ${aws:DATABASE_PASSWORD}
```

**Security Notes:**
- Prefer IAM role authentication over access keys
- Supports both JSON secrets and plain text secrets
- Consider enabling secret rotation in AWS
- Secrets are cached in memory after first read

## Memory Safety

### Limitations

Go strings are **immutable** and cannot be zeroed after use. This means:

- ⚠️ Secret values remain in memory until garbage collected
- ⚠️ Secrets may appear in memory dumps or core dumps
- ⚠️ Secrets may be swapped to disk if memory pressure occurs

### Mitigations

1. **Disable core dumps** in production:
   ```bash
   ulimit -c 0
   ```

2. **Disable swap** for security-critical systems:
   ```bash
   swapoff -a
   ```

3. **Use encrypted swap** if swap is required

4. **Use memory limits** to prevent swapping

5. **Consider using HSMs** or external secret services for extremely sensitive data

## Secret Rotation

### Current Limitations

⚠️ **No Hot Reload**: Sargantana Go does **not** support runtime secret rotation:

- Secrets are loaded **once at application startup** during configuration expansion
- Changes to secrets (env vars, files, Vault, AWS) are **not detected** at runtime
- There is **no background process** monitoring for secret updates
- Updating a secret requires **application restart**

### Why No Hot Reload?

**Design Decision**: Startup-only secret loading was chosen for:
1. **Simplicity** - No complex refresh logic or timing issues
2. **Predictability** - Secrets are immutable after startup
3. **Security** - Reduces attack surface (no background secret fetching)
4. **Performance** - No ongoing secret API calls

### Secret Rotation Strategies

#### Strategy 1: Rolling Deployment (Recommended)

**Best for:** Production environments with zero-downtime requirements

1. Update secret in backend (Vault/AWS/file system)
2. Deploy new instance with updated secret
3. Wait for health checks to pass
4. Route traffic to new instance
5. Terminate old instance

**Example with Kubernetes:**
```bash
# 1. Update secret in Vault
vault kv put secret/myapp/prod DATABASE_PASSWORD=newpass123

# 2. Trigger rolling update
kubectl rollout restart deployment/myapp

# 3. Monitor rollout
kubectl rollout status deployment/myapp
```

#### Strategy 2: Blue-Green Deployment

**Best for:** Critical services requiring instant rollback

1. Deploy new version (green) with new secrets alongside old version (blue)
2. Test green environment thoroughly
3. Switch traffic from blue to green
4. Keep blue running briefly for quick rollback
5. Terminate blue when confident

#### Strategy 3: Dual Secret Period

**Best for:** Database credentials, API keys with overlap support

1. Add new secret while keeping old one active
2. Deploy application with new secret
3. Wait for all instances to restart
4. Remove old secret from backend

**Example:**
```yaml
# Old config
database_password: ${vault:DB_PASSWORD}

# Transition period - both work
# Backend has both DB_PASSWORD and DB_PASSWORD_NEW

# New config after rotation
database_password: ${vault:DB_PASSWORD_NEW}
```

#### Strategy 4: External Secret Operator (Kubernetes)

**Best for:** Kubernetes environments

Use [External Secrets Operator](https://external-secrets.io/) to:
- Sync secrets from Vault/AWS to Kubernetes secrets
- Automatically restart pods when secrets change
- Integrate with Sargantana's file resolver

```yaml
# Kubernetes secret mounted as file
server:
  secrets_dir: "/etc/secrets"

config:
  database_password: ${file:DB_PASSWORD}
```

### Monitoring Secret Health

**Implement health checks** that verify secrets are valid:

```go
// Example health check
func healthCheck(db *sql.DB) error {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    if err := db.PingContext(ctx); err != nil {
        return errors.Wrap(err, "database health check failed - invalid credentials?")
    }
    return nil
}
```

**Set up alerts** for:
- Secret expiration approaching (Vault leases, AWS rotation schedules)
- Authentication failures after secret rotation
- Application startup failures due to invalid secrets

### Secret Rotation Checklist

Before rotating secrets:

- [ ] Identify all services using the secret
- [ ] Plan rotation strategy (rolling/blue-green/dual)
- [ ] Schedule maintenance window if needed
- [ ] Update secret in backend (Vault/AWS/files)
- [ ] Test new secret in staging environment
- [ ] Deploy to production
- [ ] Monitor for errors and authentication failures
- [ ] Verify all instances using new secret
- [ ] Remove old secret after confirmation
- [ ] Document rotation in incident log

### Future Improvements

If runtime secret rotation becomes necessary, consider:

1. **Implement a refresh mechanism** with configurable interval
2. **Add callback hooks** for secret changes
3. **Support graceful connection recycling** for database clients
4. **Add metrics** for tracking secret age and rotation events

For now, the restart-based approach is recommended as it's simple, reliable, and secure.

## Error Handling

### Panic Safety

The framework panics if a secret cannot be resolved during startup. Error messages are sanitized to avoid leaking sensitive information:

```go
// Safe: does not include property name or value
panic: error resolving property: no resolver registered for prefix "vault"
```

**Note:** The property name is intentionally NOT included to prevent leaking secret identifiers.

### Missing Environment Variables

The EnvResolver logs warnings for missing or empty environment variables to prevent silent failures:

**Production (Info level):**
```
WARN Environment variable not set or empty - using empty string env_var=DATABASE_PASSWORD
```

**Why this matters:**
- Prevents silent failures with empty secrets
- Makes misconfiguration immediately visible in logs
- Helps diagnose startup issues quickly

**Best practice:** Set all required environment variables before starting the application. Use startup validation to ensure critical secrets are present:

```go
func validateRequiredSecrets(cfg *config.Config) error {
    required := []string{"DATABASE_PASSWORD", "API_KEY", "SESSION_SECRET"}
    for _, key := range required {
        if os.Getenv(key) == "" {
            return fmt.Errorf("required environment variable %q is not set", key)
        }
    }
    return nil
}
```

## Best Practices

### 1. Use Appropriate Secret Backends

- **Environment variables**: Development only, non-sensitive configuration
- **File-based**: Docker secrets, Kubernetes secrets, moderate security
- **Vault/AWS**: Production, high security, credential rotation

### 2. Restrict Access

- Use minimal IAM permissions for AWS
- Use scoped Vault tokens with specific policies
- Set file permissions to `600` (owner read/write only)

### 3. Audit and Monitor

- Enable audit logging in Vault/AWS
- Monitor secret access patterns
- Alert on unexpected secret access

### 4. Principle of Least Privilege

- Grant only the secrets each application needs
- Use separate secret paths/namespaces per application
- Avoid sharing secrets across environments

### 5. Regular Security Reviews

- Review who has access to secrets
- Rotate secrets periodically
- Remove unused secrets
- Update dependencies regularly

## Reporting Security Issues

If you discover a security vulnerability, please email security@example.com (replace with your actual contact). Do not create a public GitHub issue.

## Security Updates

This document will be updated as new security features are added or vulnerabilities are discovered.

**Last Updated:** 2025-10-27
