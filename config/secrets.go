package config

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/vault/api"
	"github.com/pkg/errors"
)

// LoadSecretsFromDir loads secrets from a directory and sets them as environment variables.
// Each file in the directory becomes an environment variable with the filename as the key
// and the file content as the value.
func LoadSecretsFromDir(dir string) error {
	if dir == "" {
		log.Println("No secrets directory configured, skipping file secrets loading")
		return nil
	}

	files, err := os.ReadDir(dir)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("error reading secrets directory %s", dir))
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		name := file.Name()
		content, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return fmt.Errorf("error reading secret file %s: %v", name, err)
		}
		err = os.Setenv(name, strings.TrimSpace(string(content)))
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("error setting environment variable %s", strings.ToUpper(name)))
		}
	}

	return nil
}

// LoadSecretsFromVault loads secrets from a HashiCorp Vault instance and sets them as environment variables.
// It connects to Vault using the provided configuration and retrieves all key-value pairs from the specified path.
// Each key is converted to uppercase and set as an environment variable.
func (c Config) LoadSecretsFromVault() error {
	vaultConfig := c.Vault
	if !vaultConfig.IsValid() {
		log.Println("Vault configuration incomplete, skipping Vault secrets loading")
		return nil
	}

	config := api.DefaultConfig()
	config.Address = vaultConfig.Address

	client, err := api.NewClient(config)
	if err != nil {
		return errors.Wrap(err, "failed to create Vault client")
	}

	client.SetToken(vaultConfig.Token)

	if vaultConfig.Namespace != "" {
		client.SetNamespace(vaultConfig.Namespace)
	}

	secret, err := client.Logical().Read(vaultConfig.Path)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("failed to read secret from path %s", vaultConfig.Path))
	}

	if secret == nil {
		log.Printf("No secret found at path %s", vaultConfig.Path)
		return nil
	}

	// Handle both KV v1 and KV v2 formats
	var data map[string]interface{}
	if secret.Data["data"] != nil {
		// KV v2 format
		if dataMap, ok := secret.Data["data"].(map[string]interface{}); ok {
			data = dataMap
		} else {
			return fmt.Errorf("unexpected data format in KV v2 secret")
		}
	} else {
		// KV v1 format
		data = secret.Data
	}

	for key, value := range data {
		if strValue, ok := value.(string); ok {
			err = os.Setenv(strings.ToUpper(key), strValue)
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("error setting environment variable %s", strings.ToUpper(key)))
			}
		}
	}

	return nil
}

// LoadSecrets loads secrets from both file system and Vault (if configured).
// Vault secrets will override file-based secrets if there are conflicts.
func (c Config) LoadSecrets() error {
	// Load file-based secrets first
	err := LoadSecretsFromDir(c.ServerConfig.SecretsDir)
	if err != nil {
		return errors.Wrap(err, "failed to load secrets from directory")
	}

	// Load Vault secrets (these will override file secrets)
	err = c.LoadSecretsFromVault()
	if err != nil {
		return errors.Wrap(err, "failed to load secrets from Vault")
	}

	return nil
}
