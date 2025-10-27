package resolver

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

// FileResolverConfig holds configuration for the file-based resolver
type FileResolverConfig struct {
	SecretsDir string `yaml:"secrets_dir"`
}

// Validate checks if the FileResolverConfig has all required fields set
func (f FileResolverConfig) Validate() error {
	if f.SecretsDir == "" {
		return errors.New("secrets_dir is required for file resolver")
	}
	return nil
}

// CreateClient creates a FileResolver from this config.
// Implements a factory pattern similar to other configs.
// Returns *FileResolver on success, or an error if creation fails.
func (f FileResolverConfig) CreateClient() (*FileResolver, error) {
	if err := f.Validate(); err != nil {
		return nil, err
	}
	return newFileResolver(f.SecretsDir), nil
}

// FileResolver reads secrets from files in a configured directory.
// Useful for Docker secrets, Kubernetes secrets, or local development.
//
// Example usage in config:
//
//	password: ${file:db_password}  # Reads from <secretsDir>/db_password
//
// The file contents are trimmed of whitespace.
type FileResolver struct {
	secretsDir string
}

// newFileResolver creates a new file-based resolver
//
// Parameters:
//   - secretsDir: The directory containing secret files
func newFileResolver(secretsDir string) *FileResolver {
	return &FileResolver{
		secretsDir: secretsDir,
	}
}

// Resolve reads a secret from a file
func (f *FileResolver) Resolve(key string) (string, error) {
	if f.secretsDir == "" {
		return "", errors.New("no secrets directory configured")
	}

	key = strings.TrimSpace(key)
	if key == "" {
		return "", errors.New("no file specified for file secret")
	}

	filePath := filepath.Join(f.secretsDir, key)
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", errors.Wrapf(err, "error reading secret file %q", filePath)
	}

	secret := strings.TrimSpace(string(content))
	log.Debug().Str("file", filePath).Msg("Retrieved secret from file")
	return secret, nil
}

// Name returns the resolver name
func (f *FileResolver) Name() string {
	return "File"
}
