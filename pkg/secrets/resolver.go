// package secrets provides an extensible property resolution system that allows
// developers to register custom property resolvers for different prefixes.
package secrets

import (
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

// PropertyResolver defines the interface that all property resolvers must implement.
// A resolver is responsible for retrieving a property value based on a key.
//
// Example implementations:
//   - EnvResolver: Resolves environment variables
//   - FileResolver: Reads secrets from files
//   - VaultResolver: Retrieves secrets from HashiCorp Vault
//   - Custom: Database lookups, remote APIs, encrypted stores, etc.
type PropertyResolver interface {
	// Resolve retrieves the property value for the given key.
	// Returns the resolved value or an error if resolution fails.
	//
	// Parameters:
	//   - key: The property key to resolve (without the prefix)
	//
	// Returns:
	//   - string: The resolved property value
	//   - error: An error if the property cannot be resolved
	Resolve(key string) (string, error)

	// Name returns a human-readable name for this resolver (for logging/debugging)
	Name() string
}

// registry manages the registration and lookup of property resolvers.
// It provides a thread-safe registry for associating prefixes with their resolvers.
var resolvers = make(map[string]PropertyResolver)

func init() {
	// Register default resolvers in the global registry
	Register("env", NewEnvResolver())
}

// Register registers a property resolver for a specific prefix.
// The prefix should not include the trailing colon (e.g., "vault" not "vault:").
//
// If a resolver is already registered for the prefix, it will be replaced and
// a warning will be logged.
//
// Example:
//
//	registry.Register("vault", NewVaultResolver(vaultConfig))
//	registry.Register("custom", NewCustomResolver(customConfig))
//
// Thread-safe: This method can be called concurrently.
func Register(prefix string, resolver PropertyResolver) {
	if _, exists := resolvers[prefix]; exists {
		// Log warning about override (but don't fail)
		log.Warn().Msgf("Overriding existing resolver for prefix %q", prefix)
	}

	resolvers[prefix] = resolver
}

// Unregister removes a resolver for a specific prefix.
// This is useful for testing or dynamic reconfiguration.
//
// Thread-safe: This method can be called concurrently.
func Unregister(prefix string) {
	delete(resolvers, prefix)
}

// Resolve attempts to resolve a property using the appropriate resolver.
// The input should be in the format "prefix:key" or just "key" (defaults to env).
//
// Examples:
//   - "vault:DATABASE_PASSWORD" -> Uses VaultResolver
//   - "file:api_key" -> Uses FileResolver
//   - "env:PORT" -> Uses EnvResolver (explicit)
//   - "PORT" -> Uses EnvResolver (implicit, no prefix)
//
// Returns:
//   - string: The resolved property value
//   - error: An error if no resolver is found or resolution fails
//
// Thread-safe: This method can be called concurrently.
func Resolve(property string) (string, error) {
	// Parse prefix and key
	prefix, key := parseProperty(property)

	// Look up the resolver
	resolver, exists := resolvers[prefix]
	if !exists {
		return "", errors.Errorf("no resolver registered for prefix %q", prefix)
	}

	// Resolve the property
	value, err := resolver.Resolve(key)
	if err != nil {
		return "", errors.Wrapf(err, "failed to resolve property %q using %s resolver", property, resolver.Name())
	}

	return value, nil
}

// GetResolver returns the resolver registered for a specific prefix.
// Returns nil if no resolver is registered for the prefix.
func GetResolver(prefix string) PropertyResolver {
	return resolvers[prefix]
}

// ListPrefixes returns a list of all registered prefixes.
// Useful for debugging and documentation.
//
// Thread-safe: This method can be called concurrently.
func ListPrefixes() []string {
	prefixes := make([]string, 0, len(resolvers))
	for prefix := range resolvers {
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
