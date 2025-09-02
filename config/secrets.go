package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/animalet/sargantana-go/logger"
	"github.com/hashicorp/vault/api"
	"github.com/pkg/errors"
)

type vaultManager struct {
	logical *api.Logical
	path    string
}

var vaultManagerInstance *vaultManager

func (c *Config) createVaultManager() error {
	if !c.Vault.IsValid() {
		logger.Info("Vault configuration incomplete, skipping Vault secrets loading")
		return nil
	}
	config := api.DefaultConfig()
	config.Address = c.Vault.Address
	client, err := api.NewClient(config)
	if err != nil {
		return err
	}

	client.SetToken(c.Vault.Token)

	if c.Vault.Namespace != "" {
		client.SetNamespace(c.Vault.Namespace)
	}

	vaultManagerInstance = &vaultManager{
		logical: client.Logical(),
		path:    c.Vault.Path,
	}

	logger.Info("Vault client created successfully")
	return nil
}

// secret retrieves a secret from Vault at the configured path.
func (v *vaultManager) secret(name string) (*string, error) {
	secret, err := v.logical.Read(v.path)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("failed to read secret from path %q", v.path))
	}

	if secret == nil || secret.Data == nil {
		return nil, errors.Errorf("no secret found at the specified path: %q", v.path)
	}

	// Handle both KV v1 and KV v2 formats
	var data map[string]interface{}
	if secret.Data["data"] != nil {
		// KV v2 format
		if dataMap, ok := secret.Data["data"].(map[string]interface{}); ok {
			data = dataMap
		} else {
			return nil, fmt.Errorf("unexpected data format in KV v2 secret")
		}
	} else {
		// KV v1 format
		data = secret.Data
	}

	if strValue, ok := data[name].(string); ok {
		logger.Infof("Retrieved secret %q from Vault at path %q", name, v.path)
		return &strValue, nil
	} else {
		return nil, errors.Errorf("secret %q not found in Vault at path %q", name, v.path)
	}
}

// secretFromFile reads the content of a file and returns it as a trimmed string.
// It returns an error if the file cannot be read.
func secretFromFile(file string) (string, error) {
	if secretDir == "" {
		return "", errors.New("no secrets directory configured")
	}
	file = strings.TrimSpace(file)
	if file == "" {
		return "", errors.New("no file specified for file secret")
	}
	file = filepath.Join(secretDir, file)
	b, err := os.ReadFile(file)
	if err != nil {
		return "", errors.Wrap(err, "error reading secret file")
	}
	secret := strings.TrimSpace(string(b))
	logger.Infof("Retrieved secret %q from file", file)
	return secret, nil
}
