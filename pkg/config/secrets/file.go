package secrets

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

// FileSecretConfig holds configuration for the file-based resolver
type FileSecretConfig struct {
	SecretsDir string `yaml:"secrets_dir"`
}

// Validate checks if the FileSecretConfig has all required fields set
func (f FileSecretConfig) Validate() error {
	if f.SecretsDir == "" {
		return errors.New("secrets_dir is required for file resolver")
	}

	// Test if the directory exists and is a directory
	info, err := os.Stat(f.SecretsDir)
	if os.IsNotExist(err) {
		return errors.Errorf("secrets_dir %q does not exist", f.SecretsDir)
	}
	if err != nil {
		return errors.Wrapf(err, "error accessing secrets_dir %q", f.SecretsDir)
	}
	if !info.IsDir() {
		return errors.Errorf("secrets_dir %q is not a directory", f.SecretsDir)
	}
	return nil
}

// CreateClient creates a FileSecretLoader from this config.
// Implements the config.ClientFactory[*FileSecretLoader] interface.
// Returns *FileSecretLoader on success, or an error if creation fails.

func (f FileSecretConfig) CreateClient() (*FileSecretLoader, error) {
	return NewFileSecretLoader(f.SecretsDir)
}

// FileSecretLoader reads secrets from files in a configured directory.
// Useful for Docker secrets, Kubernetes secrets, or local development.
//
// Example usage in config:
//
//	password: ${file:db_password}  # Reads from <secretsDir>/db_password
//
// The file contents are trimmed of whitespace.
type FileSecretLoader struct {
	secretsDir string
}

// NewFileSecretLoader creates a new file-based resolver
//
// Parameters:
//   - secretsDir: The directory containing secret files
func NewFileSecretLoader(secretsDir string) (*FileSecretLoader, error) {
	if secretsDir == "" {
		return nil, errors.New("no secrets directory configured")
	}

	absSecretsDir, err := filepath.Abs(secretsDir)
	if err != nil {
		return nil, errors.Wrap(err, "failed to resolve absolute path for secrets directory")
	}

	return &FileSecretLoader{
		secretsDir: absSecretsDir,
	}, nil
}

// Resolve reads a secret from a file
func (f *FileSecretLoader) Resolve(key string) (string, error) {
	if key == "" {
		return "", errors.New("no file specified for file secret")
	}

	// Reject absolute paths
	if filepath.IsAbs(key) {
		return "", errors.New("invalid secret key: absolute paths not allowed")
	}

	// Sanitize the key to prevent path traversal
	cleanKey := filepath.Clean(key)
	if strings.Contains(cleanKey, "..") {
		return "", errors.New("invalid secret key: path traversal detected")
	}

	// Construct the full path using the cached absolute directory
	absFilePath := filepath.Join(f.secretsDir, cleanKey)

	// Verify the resolved path is within the secrets directory
	// This check is still useful to ensure that even if cleanKey doesn't have ".."
	// somehow we don't end up outside (though Join + Clean should prevent it).
	// The main protection is the ".." check above and Join behavior.
	if !strings.HasPrefix(absFilePath, f.secretsDir+string(filepath.Separator)) {
		return "", errors.New("invalid secret key: outside secrets directory")
	}

	// #nosec G304 -- Path traversal is prevented by validation above
	content, err := os.ReadFile(absFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", errors.New("secret not found")
		}
		return "", errors.New("failed to read secret")
	}

	secret := strings.TrimSpace(string(content))
	log.Debug().Str("file", absFilePath).Msg("Retrieved secret from file")
	return secret, nil
}

// Name returns the resolver name
func (f *FileSecretLoader) Name() string {
	return "File"
}
