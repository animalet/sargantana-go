package main

import (
	"github.com/animalet/sargantana-go/pkg/config"
	"github.com/animalet/sargantana-go/pkg/config/secrets"
	"github.com/pkg/errors"
)

// loadConfig reads the configuration file and registers all secret providers
func loadConfig(configPath string) (*config.Config, error) {
	cfg, err := config.NewConfig(configPath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load configuration file")
	}

	// Register all secret providers based on configuration
	if err := registerSecretProviders(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// registerSecretProviders registers all configured secret providers
func registerSecretProviders(cfg *config.Config) error {
	// Register Vault provider if configured
	vaultClient, err := config.GetClient[secrets.VaultConfig](cfg, "vault")
	if err != nil {
		return errors.Wrap(err, "failed to load or create Vault client")
	}
	if vaultClient != nil {
		// Still need to get config for the Path field
		vaultCfg, err := config.Get[secrets.VaultConfig](cfg, "vault")
		if err != nil {
			return errors.Wrap(err, "failed to load Vault configuration")
		}
		secrets.Register("vault", secrets.NewVaultSecretLoader(*vaultClient, vaultCfg.Path))
	}

	// Register file provider if configured
	fileResolver, err := config.GetClient[secrets.FileSecretConfig](cfg, "file_resolver")
	if err != nil {
		return errors.Wrap(err, "failed to load or create file secret provider")
	}
	if fileResolver != nil {
		secrets.Register("file", *fileResolver)
	}

	// Register AWS Secrets Manager provider if configured
	awsClient, err := config.GetClient[secrets.AWSConfig](cfg, "aws")
	if err != nil {
		return errors.Wrap(err, "failed to load or create AWS Secrets Manager client")
	}
	if awsClient != nil {
		// Still need to get config for the SecretName field
		awsCfg, err := config.Get[secrets.AWSConfig](cfg, "aws")
		if err != nil {
			return errors.Wrap(err, "failed to load AWS Secrets Manager configuration")
		}
		secrets.Register("aws", secrets.NewAWSSecretLoader(*awsClient, awsCfg.SecretName))
	}

	return nil
}
