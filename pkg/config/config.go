// Package config provides configuration management for the Sargantana Go web framework.
// It handles server configuration including address, session storage, secrets directory,
// debug mode, and session naming.
package config

import (
	"os"
	"reflect"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

// Config holds the configuration settings for the Sargantana Go server.
// It encapsulates all necessary configuration parameters including network settings,
// session storage options, security settings, and debugging preferences.
type Config struct {
	ServerConfig       ServerConfig       `yaml:"server"`
	ControllerBindings ControllerBindings `yaml:"controllers"`
	Other              map[string]any     `yaml:",inline"`
}

// Validatable interface defines types that can be validated.
type Validatable interface {
	Validate() error
}

// ClientFactory is a generic interface for configurations that can create clients
// for data sources like Vault, Redis, databases, etc.
// The type parameter T specifies the concrete client type that will be returned.
// Implementations should create and configure the appropriate client type
// from their configuration details.
//
// Example implementations:
//   - VaultConfig implements ClientFactory[*api.Client]
//   - RedisConfig implements ClientFactory[*redis.Pool]
//
// Usage:
//
//	client, err := cfg.Vault.CreateClient()  // Returns (*api.Client, error) directly
type ClientFactory[T any] interface {
	Validatable
	// CreateClient creates and configures a client from the config details.
	// Returns the strongly-typed client T and an error if creation fails.
	CreateClient() (T, error)
}

// Validate checks the Config struct for required fields and valid values.
// It validates the ServerConfig and each ControllerBinding.
// Returns an error if any validation fails.
func (c *Config) Validate() error {
	if err := c.ServerConfig.Validate(); err != nil {
		return errors.Wrap(err, "server configuration is invalid")
	}

	return c.ControllerBindings.Validate()
}

// ReadConfig reads the YAML configuration file and unmarshalls its content into the provided struct.
//
// Parameters:
//   - file: Path to the YAML configuration file
//
// Returns:
//   - *T: Pointer to the struct of type T containing the unmarshalled configuration
//   - error: Error if reading or unmarshalling
func ReadConfig(file string) (*Config, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	var out *Config
	err = yaml.Unmarshal(data, &out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Load validates and processes the configuration.
func (cfg *Config) Load() (err error) {
	expandVariables(reflect.ValueOf(&cfg.ServerConfig).Elem())
	if err = cfg.ServerConfig.Validate(); err != nil {
		return errors.Wrap(err, "server configuration is invalid")
	}

	// Validate all controller bindings
	for i, binding := range cfg.ControllerBindings {
		if err = binding.Validate(); err != nil {
			return errors.Wrapf(err, "controller binding at index %d is invalid", i)
		}
	}

	return nil
}

// LoadConfig loads a partial configuration from the Config.Other map by key.
// It unmarshals, validates, and expands variables for the configuration.
func LoadConfig[T Validatable](key string, cfg *Config) (partial *T, err error) {
	c, exist := cfg.Other[key]
	if !exist {
		return nil, errors.Errorf("no configuration found for %q", key)
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return nil, errors.Wrap(err, "error marshalling to YAML")
	}

	partial, err = UnmarshalTo[T](data)
	if err != nil {
		return nil, err
	}

	expandVariables(reflect.ValueOf(partial).Elem())
	if err = (*partial).Validate(); err != nil {
		return nil, errors.Wrap(err, "configuration is invalid")
	}
	return partial, nil
}

// UnmarshalTo unmarshals raw YAML data into a new instance of type T.
// This function creates a new instance and returns it, avoiding addressability issues.
// It is used for both controller configurations and other partial configurations.
func UnmarshalTo[T Validatable](data []byte) (*T, error) {
	if data == nil {
		return nil, nil
	}

	var result T
	err := yaml.Unmarshal(data, &result)
	if err != nil {
		return nil, err
	}

	if err = result.Validate(); err != nil {
		return nil, errors.Wrap(err, "configuration is invalid")
	}

	// Always try to expand environment variables for structs
	v := reflect.ValueOf(&result).Elem()
	expandVariables(v)

	return &result, nil
}
