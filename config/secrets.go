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
func (c *Config) LoadSecretsFromVault() error {
	vaultConfig := c.Vault
	if !vaultConfig.IsValid() {
		log.Println("Vault configuration incomplete, skipping Vault secrets loading")
		return nil
	}

	// Create Vault client configuration
	config := api.DefaultConfig()
	config.Address = vaultConfig.Address

	// Create Vault client
	client, err := api.NewClient(config)
	if err != nil {
		return errors.Wrap(err, "error creating Vault client")
	}

	// Set the token for authentication
	client.SetToken(os.ExpandEnv(vaultConfig.Token))

	// Set namespace if provided (for Vault Enterprise)
	namespace := vaultConfig.Namespace
	if namespace != "" {
		client.SetNamespace(namespace)
	}

	// Read secrets from the specified path
	path := vaultConfig.Path
	secret, err := client.Logical().Read(path)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("error reading secrets from Vault path %s", path))
	}

	if secret == nil {
		log.Printf("No secret found at Vault path %s", path)
		return nil
	}

	// Extract data from the secret
	var data map[string]any
	if secret.Data != nil {
		// For KV v2 secrets engine, data is nested under "data" key
		if dataField, exists := secret.Data["data"]; exists {
			if dataMap, ok := dataField.(map[string]any); ok {
				data = dataMap
			} else {
				return fmt.Errorf("unexpected data format in Vault secret at path %s", path)
			}
		} else {
			// For KV v1 secrets engine or other engines, data is directly in secret.Data
			data = secret.Data
		}
	}

	if data == nil {
		log.Printf("No data found in Vault secret at path %s", path)
		return nil
	}

	// Set environment variables from Vault secrets
	count := 0
	for key, value := range data {
		if valueStr, ok := value.(string); ok {
			envKey := strings.ToUpper(key)
			err = os.Setenv(envKey, valueStr)
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("error setting environment variable %s from Vault", envKey))
			}
			count++
		} else {
			log.Printf("Warning: Vault secret key %s has non-string value, skipping", key)
		}
	}

	log.Printf("Successfully loaded %d secrets from Vault path %s", count, path)
	return nil
}

// LoadSecrets loads secrets from both directory and Vault sources.
// It first loads secrets from the directory (if configured), then from Vault (if configured).
// Vault secrets will override directory secrets if they have the same key names.
func (c *Config) LoadSecrets() error {
	// Load secrets from directory first
	if err := LoadSecretsFromDir(c.ServerConfig.SecretsDir); err != nil {
		return errors.Wrap(err, "error loading secrets from directory")
	}

	// Load secrets from Vault (will override directory secrets with same names)
	if err := c.LoadSecretsFromVault(); err != nil {
		return errors.Wrap(err, "error loading secrets from Vault")
	}

	return nil
}
