package resolver

import (
	"fmt"

	"github.com/hashicorp/vault/api"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

// VaultConfig holds configuration for connecting to HashiCorp Vault
type VaultConfig struct {
	Address   string `yaml:"address"`
	Token     string `yaml:"token"`
	Path      string `yaml:"path"`
	Namespace string `yaml:"namespace"`
}

// Validate checks if the VaultConfig has all required fields set
func (v VaultConfig) Validate() error {
	if v.Address == "" {
		return errors.New("Vault address is required")
	}
	if v.Token == "" {
		return errors.New("Vault token is required")
	}
	if v.Path == "" {
		return errors.New("Vault path is required")
	}
	return nil
}

// CreateVaultClient is a helper function to create and configure a Vault client
// from a VaultConfig. This is typically called by applications during startup
// to set up the Vault resolver.
//
// Example:
//
//	client, err := resolver.CreateVaultClient(cfg.Vault)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	resolver.Register("vault", resolver.NewVaultResolver(client, cfg.Vault.Path))
func CreateVaultClient(vaultCfg *VaultConfig) (*api.Client, error) {
	if vaultCfg == nil {
		return nil, errors.New("vault configuration is nil")
	}

	if err := vaultCfg.Validate(); err != nil {
		return nil, errors.Wrap(err, "invalid Vault configuration")
	}

	config := api.DefaultConfig()
	config.Address = vaultCfg.Address

	client, err := api.NewClient(config)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create Vault client")
	}

	client.SetToken(vaultCfg.Token)

	if vaultCfg.Namespace != "" {
		client.SetNamespace(vaultCfg.Namespace)
	}

	return client, nil
}

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
