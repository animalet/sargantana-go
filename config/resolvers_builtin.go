package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/vault/api"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

// ========================================
// Environment Variable Resolver
// ========================================

// EnvResolver resolves properties from environment variables.
// This is the default resolver when no prefix is specified.
//
// Example usage in config:
//
//	address: ${PORT}           # Resolves from env (implicit)
//	address: ${env:PORT}       # Resolves from env (explicit)
type EnvResolver struct{}

// NewEnvResolver creates a new environment variable resolver
func NewEnvResolver() *EnvResolver {
	return &EnvResolver{}
}

// Resolve retrieves an environment variable value
func (e *EnvResolver) Resolve(key string) (string, error) {
	value := os.Getenv(key)
	// Note: We don't treat empty/missing as an error - Go's os.Expand behavior
	return value, nil
}

// Name returns the resolver name
func (e *EnvResolver) Name() string {
	return "Environment"
}

// ========================================
// File Resolver
// ========================================

// FileResolver reads secrets from files in a configured directory.
// Useful for Docker secrets, Kubernetes secrets, or local development.
//
// Example usage in config:
//
//	password: ${file:db_password}  # Reads from <secretsDir>/db_password
//
// The file contents are trimmed of whitespace.
type FileResolver struct {
	secretsDir string
}

// NewFileResolver creates a new file-based resolver
//
// Parameters:
//   - secretsDir: The directory containing secret files
func NewFileResolver(secretsDir string) *FileResolver {
	return &FileResolver{
		secretsDir: secretsDir,
	}
}

// Resolve reads a secret from a file
func (f *FileResolver) Resolve(key string) (string, error) {
	if f.secretsDir == "" {
		return "", errors.New("no secrets directory configured")
	}

	key = strings.TrimSpace(key)
	if key == "" {
		return "", errors.New("no file specified for file secret")
	}

	filePath := filepath.Join(f.secretsDir, key)
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", errors.Wrapf(err, "error reading secret file %q", filePath)
	}

	secret := strings.TrimSpace(string(content))
	log.Info().Str("file", filePath).Msg("Retrieved secret from file")
	return secret, nil
}

// Name returns the resolver name
func (f *FileResolver) Name() string {
	return "File"
}

// ========================================
// Vault Resolver
// ========================================

// VaultResolver retrieves secrets from HashiCorp Vault.
// Supports both KV v1 and KV v2 secret engines.
//
// Example usage in config:
//
//	password: ${vault:DATABASE_PASSWORD}  # Reads from configured Vault path
//
// The Vault path is configured when creating the resolver.
type VaultResolver struct {
	logical *api.Logical
	path    string
}

// NewVaultResolver creates a new Vault-based resolver
//
// Parameters:
//   - client: Configured Vault API client
//   - path: The Vault path to read secrets from (e.g., "secret/data/myapp")
func NewVaultResolver(client *api.Client, path string) *VaultResolver {
	return &VaultResolver{
		logical: client.Logical(),
		path:    path,
	}
}

// Resolve retrieves a secret from Vault
func (v *VaultResolver) Resolve(key string) (string, error) {
	secret, err := v.logical.Read(v.path)
	if err != nil {
		return "", errors.Wrapf(err, "failed to read secret from Vault path %q", v.path)
	}

	if secret == nil || secret.Data == nil {
		return "", errors.Errorf("no secret found at Vault path %q", v.path)
	}

	// Handle both KV v1 and KV v2 formats
	var data map[string]interface{}
	if secret.Data["data"] != nil {
		// KV v2 format
		if dataMap, ok := secret.Data["data"].(map[string]interface{}); ok {
			data = dataMap
		} else {
			return "", fmt.Errorf("unexpected data format in KV v2 secret")
		}
	} else {
		// KV v1 format
		data = secret.Data
	}

	// Extract the requested key
	if strValue, ok := data[key].(string); ok {
		log.Info().
			Str("secret_name", key).
			Str("vault_path", v.path).
			Msg("Retrieved secret from Vault")
		return strValue, nil
	}

	return "", errors.Errorf("secret %q not found in Vault at path %q", key, v.path)
}

// Name returns the resolver name
func (v *VaultResolver) Name() string {
	return "Vault"
}
