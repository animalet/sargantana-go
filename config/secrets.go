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

type vaultManager struct {
	logical *api.Logical
	path    string
}

var vaultManagerInstance *vaultManager

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
		log.Info().Str("secret_name", name).Str("vault_path", v.path).Msg("Retrieved secret from Vault")
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
	log.Info().Str("file", file).Msg("Retrieved secret from file")
	return secret, nil
}
