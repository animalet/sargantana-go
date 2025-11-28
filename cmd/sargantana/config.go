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
	vaultCfg, err := config.Get[secrets.VaultConfig](cfg, "vault")
	if err != nil {
		return errors.Wrap(err, "failed to load Vault configuration")
	}
	if vaultCfg != nil {
		vaultClient, err := vaultCfg.CreateClient()
		if err != nil {
			return errors.Wrap(err, "failed to create Vault client")
		}
		secrets.Register("vault", secrets.NewVaultSecretLoader(vaultClient, vaultCfg.Path))
	}

	// Register file provider if configured
	fileResolverCfg, err := config.Get[secrets.FileSecretConfig](cfg, "file_resolver")
	if err != nil {
		return errors.Wrap(err, "failed to load file secret resolver configuration")
	}
	if fileResolverCfg != nil {
		fileResolver, err := fileResolverCfg.CreateClient()
		if err != nil {
			return errors.Wrap(err, "failed to create file secret provider")
		}
		secrets.Register("file", fileResolver)
	}

	// Register AWS Secrets Manager provider if configured
	awsCfg, err := config.Get[secrets.AWSConfig](cfg, "aws")
	if err != nil {
		return errors.Wrap(err, "failed to load AWS Secrets Manager configuration")
	}
	if awsCfg != nil {
		awsClient, err := awsCfg.CreateClient()
		if err != nil {
			return errors.Wrap(err, "failed to create AWS Secrets Manager client")
		}
		secrets.Register("aws", secrets.NewAWSSecretLoader(awsClient, awsCfg.SecretName))
	}

	return nil
}
