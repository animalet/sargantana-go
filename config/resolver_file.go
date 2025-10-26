package config

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

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

// NewFileResolver creates a new file-based resolver
//
// Parameters:
//   - secretsDir: The directory containing secret files
func NewFileResolver(secretsDir string) *FileResolver {
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
	log.Info().Str("file", filePath).Msg("Retrieved secret from file")
	return secret, nil
}

// Name returns the resolver name
func (f *FileResolver) Name() string {
	return "File"
}
