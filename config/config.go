// Package config provides configuration management for the Sargantana Go web framework.
// It handles server configuration including address, session storage, secrets directory,
// debug mode, and session naming.
package config

// Config holds the configuration settings for the Sargantana Go server.
// It encapsulates all necessary configuration parameters including network settings,
// session storage options, security settings, and debugging preferences.
type Config struct {
	address           string                   `yaml:"address"`
	redisSessionStore string                   `yaml:"redis_session_store"`
	secretsDir        string                   `yaml:"secrets_dir"`
	vaultConfig       VaultConfig              `yaml:"vault"`
	debug             bool                     `yaml:"debug"`
	sessionName       string                   `yaml:"session_name"`
	controllerConfig  []map[string]interface{} `yaml:"controller_config"`
}

// VaultConfig holds configuration for connecting to HashiCorp Vault
type VaultConfig struct {
	Address   string `yaml:"address"`
	Token     string `yaml:"token"`
	Path      string `yaml:"path"`
	Namespace string `yaml:"namespace"`
}

// NewConfig creates a new Config instance with the provided parameters.
// It initializes all configuration fields with the given values.
//
// Parameters:
//   - address: The host:port address where the server will listen (e.g., "localhost:8080")
//   - redisSessionStore: Redis server address for session storage (empty string means use cookies)
//   - secretsDir: Directory path containing secret files for environment variables
//   - debug: Whether to enable debug mode for detailed logging
//   - sessionName: Name of the session cookie used for user sessions
//
// Returns a pointer to the newly created Config instance.
func NewConfig(address, redisSessionStore, secretsDir string, debug bool, sessionName string) *Config {
	return &Config{
		address:           address,
		redisSessionStore: redisSessionStore,
		secretsDir:        secretsDir,
		debug:             debug,
		sessionName:       sessionName,
	}
}

// Address returns the server listen address in host:port format.
// This address is used by the HTTP server to bind and listen for incoming connections.
func (c *Config) Address() string {
	return c.address
}

// RedisSessionStore returns the Redis server address for session storage.
// If empty string is returned, the server will use cookie-based session storage instead.
// Format should be "host:port" (e.g., "localhost:6379").
func (c *Config) RedisSessionStore() string {
	return c.redisSessionStore
}

// SecretsDir returns the directory path containing secret files.
// Secret files in this directory are automatically loaded as environment variables
// with uppercase filenames. Returns empty string if no secrets directory is configured.
func (c *Config) SecretsDir() string {
	return c.secretsDir
}

// Debug returns whether debug mode is enabled.
// When true, the server runs in debug mode with detailed logging,
// request/response body logging, and other development-friendly features.
func (c *Config) Debug() bool {
	return c.debug
}

// SessionName returns the name of the session cookie.
// This name is used for both cookie-based and Redis-based session storage
// to identify user sessions consistently across the application.
func (c *Config) SessionName() string {
	return c.sessionName
}

// ControllerConfig returns the controller configuration array.
// Each element contains the type and configuration data for a controller
// that should be initialized by the server.
func (c *Config) ControllerConfig() []map[string]interface{} {
	return c.controllerConfig
}

// VaultConfig returns the Vault configuration settings.
// This includes the Vault server address, authentication token, base path for secrets,
// and optional namespace for Vault Enterprise.
func (c *Config) VaultConfig() VaultConfig {
	return c.vaultConfig
}
