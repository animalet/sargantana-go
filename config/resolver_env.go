package config

import "os"

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
	// Note: We don't treat empty/missing as an error - Go's os.Expand behavior
	return value, nil
}

// Name returns the resolver name
func (e *EnvResolver) Name() string {
	return "Environment"
}
