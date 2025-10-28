// package secrets provides an extensible secret resolution system that allows
// developers to register custom secret providers for different prefixes.
package secrets

import (
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

// PropertyResolver defines the interface that all secret providers must implement.
// A provider is responsible for retrieving a secret value based on a key.
//
// Example implementations:
//   - EnvResolver: Resolves environment variables
//   - FileResolver: Reads secrets from files
//   - VaultResolver: Retrieves secrets from HashiCorp Vault
//   - Custom: Database lookups, remote APIs, encrypted stores, etc.
type PropertyResolver interface {
	// Resolve retrieves the secret value for the given key.
	// Returns the resolved value or an error if resolution fails.
	//
	// Parameters:
	//   - key: The secret key to resolve (without the prefix)
	//
	// Returns:
	//   - string: The resolved secret value
	//   - error: An error if the secret cannot be resolved
	Resolve(key string) (string, error)

	// Name returns a human-readable name for this provider (for logging/debugging)
	Name() string
}

// providers manages the registration and lookup of secret providers.
// It provides a thread-safe registry for associating prefixes with their providers.
var providers = make(map[string]PropertyResolver)

func init() {
	// Register default provider in the global registry
	Register("env", NewEnvResolver())
}

// Register registers a secret provider for a specific prefix.
// The prefix should not include the trailing colon (e.g., "vault" not "vault:").
//
// If a provider is already registered for the prefix, it will be replaced and
// a warning will be logged.
//
// Example:
//
//	secrets.Register("vault", NewVaultResolver(vaultConfig))
//	secrets.Register("custom", NewCustomProvider(customConfig))
//
// Thread-safe: This method can be called concurrently.
func Register(prefix string, provider PropertyResolver) {
	if _, exists := providers[prefix]; exists {
		// Log warning about override (but don't fail)
		log.Warn().Msgf("Overriding existing secret provider for prefix %q", prefix)
	}

	providers[prefix] = provider
}

// Unregister removes a provider for a specific prefix.
// This is useful for testing or dynamic reconfiguration.
//
// Thread-safe: This method can be called concurrently.
func Unregister(prefix string) {
	delete(providers, prefix)
}

// Resolve attempts to resolve a secret using the appropriate provider.
// The input should be in the format "prefix:key" or just "key" (defaults to env).
//
// Examples:
//   - "vault:DATABASE_PASSWORD" -> Uses Vault provider
//   - "file:api_key" -> Uses File provider
//   - "env:PORT" -> Uses Environment provider (explicit)
//   - "PORT" -> Uses Environment provider (implicit, no prefix)
//
// Returns:
//   - string: The resolved secret value
//   - error: An error if no provider is found or resolution fails
//
// Thread-safe: This method can be called concurrently.
func Resolve(property string) (string, error) {
	// Parse prefix and key
	prefix, key := parseProperty(property)

	// Look up the provider
	provider, exists := providers[prefix]
	if !exists {
		return "", errors.Errorf("no secret provider registered for prefix %q", prefix)
	}

	// Resolve the secret
	value, err := provider.Resolve(key)
	if err != nil {
		return "", errors.Wrapf(err, "failed to resolve secret %q using %s provider", property, provider.Name())
	}

	return value, nil
}

// GetResolver returns the provider registered for a specific prefix.
// Returns nil if no provider is registered for the prefix.
func GetResolver(prefix string) PropertyResolver {
	return providers[prefix]
}

// ListPrefixes returns a list of all registered prefixes.
// Useful for debugging and documentation.
//
// Thread-safe: This method can be called concurrently.
func ListPrefixes() []string {
	prefixes := make([]string, 0, len(providers))
	for prefix := range providers {
		prefixes = append(prefixes, prefix)
	}
	return prefixes
}

// parseProperty splits a property string into prefix and key.
// If no prefix is present, defaults to "env".
//
// Examples:
//   - "vault:SECRET_KEY" -> ("vault", "SECRET_KEY")
//   - "env:PORT" -> ("env", "PORT")
//   - "PORT" -> ("env", "PORT")  // Default to env
//   - "custom:db:password" -> ("custom", "db:password")  // Only first : is separator
func parseProperty(property string) (prefix string, key string) {
	// Find the first colon
	colonIndex := -1
	for i, ch := range property {
		if ch == ':' {
			colonIndex = i
			break
		}
	}

	// No colon found - default to env resolver
	if colonIndex == -1 {
		return "env", property
	}

	// Split at first colon
	prefix = property[:colonIndex]
	key = property[colonIndex+1:]

	return prefix, key
}
