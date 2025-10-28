package secrets

import (
	"os"

	"github.com/rs/zerolog/log"
)

// EnvResolver resolves properties from environment variables.
// This is the default resolver when no prefix is specified.
//
// Example usage in config:
//
//	address: ${PORT}           # Resolves from env (implicit)
//	address: ${env:PORT}       # Resolves from env (explicit)
type EnvResolver struct{}

// NewEnvResolver creates a new environment variable resolver
func NewEnvResolver() *EnvResolver {
	return &EnvResolver{}
}

// Resolve retrieves an environment variable value
func (e *EnvResolver) Resolve(key string) (string, error) {
	value := os.Getenv(key)

	// Warn about missing environment variables to prevent silent failures
	// Note: We don't treat empty/missing as an error to maintain Go's os.Expand behavior
	if value == "" {
		log.Warn().
			Str("env_var", key).
			Msg("Environment variable not set or empty - using empty string")
	} else {
		log.Debug().
			Str("env_var", key).
			Msg("Retrieved value from environment variable")
	}

	return value, nil
}

// Name returns the resolver name
func (e *EnvResolver) Name() string {
	return "Environment"
}
