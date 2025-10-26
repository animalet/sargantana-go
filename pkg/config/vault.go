package config

import (
	"github.com/hashicorp/vault/api"
	"github.com/pkg/errors"
)

// CreateVaultClient is a helper function to create and configure a Vault client
// from a VaultConfig. This is typically called by applications during startup
// to set up the Vault resolver.
//
// Example:
//
//	client, err := config.CreateVaultClient(cfg.Vault)
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
