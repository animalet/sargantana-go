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

// loadSecretsFromDir loads secrets from a directory and sets them as environment variables.
// Each file in the directory becomes an environment variable with the filename as the key
// and the file content as the value.
func loadSecretsFromDir(dir string) error {
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

type vaultManager struct {
	logical *api.Logical
	path    string
}

var vaultManagerInstance *vaultManager

func (v *vaultManager) GetSecret() {

}

func (c Config) createVaultManager() error {
	if !c.Vault.IsValid() {
		log.Println("Vault configuration incomplete, skipping Vault secrets loading")
		return nil
	}
	config := api.DefaultConfig()
	config.OutputCurlString = true
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

	log.Println("Vault client created successfully")
	return nil
}

// secret retrieves a secret from Vault at the configured path.
func (v *vaultManager) secret(name string) (*string, error) {
	secret, err := v.logical.Read(v.path)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("failed to read secret from path %s", v.path))
	}

	if secret == nil {
		return nil, errors.New("no secret found at the specified path")
	}

	// Handle both KV v1 and KV v2 formats. Retrieve the secreta with "name"
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

	var result *string
	if value, exists := data[name]; exists {
		if strValue, ok := value.(string); ok {
			result = &strValue
		}
	}

	if result == nil {
		log.Printf("Secret %s not found in Vault at path %s", name, v.path)
	}

	return result, nil
}

// LoadSecrets loads secrets from both file system and Vault (if configured).
// Vault secrets will override file-based secrets if there are conflicts.
func (c Config) LoadSecrets() error {
	// Load file-based secrets first
	err := loadSecretsFromDir(c.ServerConfig.SecretsDir)
	if err != nil {
		return errors.Wrap(err, "failed to load secrets from directory")
	}

	// Load Vault secrets (these will override file secrets)
	err = c.createVaultManager()
	if err != nil {
		return errors.Wrap(err, "failed to load secrets from Vault")
	}

	return nil
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
	return strings.TrimSpace(string(b)), nil
}
