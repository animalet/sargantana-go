// Package config provides configuration management for the Sargantana Go web framework.
// It handles server configuration including address, session storage, secrets directory,
// debug mode, and session naming.
package config

import (
	"os"
	"reflect"

	"github.com/animalet/sargantana-go/pkg/database"
	"github.com/animalet/sargantana-go/pkg/resolver"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

type (
	// Config holds the configuration settings for the Sargantana Go server.
	// It encapsulates all necessary configuration parameters including network settings,
	// session storage options, security settings, and debugging preferences.
	Config struct {
		ServerConfig       ServerConfig          `yaml:"server"`
		Vault              *resolver.VaultConfig `yaml:"vault,omitempty"`
		ControllerBindings []ControllerBinding   `yaml:"controllers"`
		Other              map[string]any        `yaml:",inline"`
	}

	// ServerConfig holds the core server configuration parameters.
	ServerConfig struct {
		Address           string                `yaml:"address"`
		RedisSessionStore *database.RedisConfig `yaml:"redis_session_store"`
		SecretsDir        string                `yaml:"secrets_dir,omitempty"`
		SessionName       string                `yaml:"session_name"`
		SessionSecret     string                `yaml:"session_secret"`
	}

	// ControllerBinding represents the configuration for a single controller.
	ControllerBinding struct {
		TypeName   string           `yaml:"type"`
		Name       string           `yaml:"name,omitempty"`
		ConfigData ControllerConfig `yaml:"config"`
	}

	ControllerConfig []byte
)

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

// Validate checks if the ServerConfig has all required fields set.
func (c ServerConfig) Validate() error {
	if c.SessionSecret == "" {
		return errors.New("session_secret must be set and non-empty")
	}
	return nil
}

func (cfg *Config) Load() (err error) {
	expandVariables(reflect.ValueOf(&cfg.ServerConfig).Elem())
	if err = cfg.ServerConfig.Validate(); err != nil {
		return errors.Wrap(err, "server configuration is invalid")
	}

	// Expand Vault configuration if present
	if cfg.Vault != nil {
		expandVariables(reflect.ValueOf(cfg.Vault).Elem())
	}

	return nil
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
		return nil, errors.Wrap(err, "partial configuration is invalid")
	}
	return partial, nil
}

// UnmarshalYAML implements the yaml.Unmarshaler interface.
// It marshals the provided yaml.Node back into a YAML byte slice.
func (c *ControllerConfig) UnmarshalYAML(value *yaml.Node) error {
	out, err := yaml.Marshal(value)
	if err != nil {
		return err
	}
	*c = out
	return nil
}

// UnmarshalTo unmarshal the raw YAML data from ControllerConfig into a new instance of type T.
// This method creates a new instance and returns it, avoiding addressability issues.
func UnmarshalTo[T Validatable](c ControllerConfig) (*T, error) {
	if c == nil {
		return nil, nil
	}
	var result T
	err := yaml.Unmarshal(c, &result)
	if err != nil {
		return nil, err
	}

	if err = result.Validate(); err != nil {
		return nil, errors.Wrap(err, "controller config is invalid")
	}

	// Always try to expand environment variables for structs
	v := reflect.ValueOf(&result).Elem()
	expandVariables(v)

	return &result, nil
}

// expand checks for specific prefixes in the string and expands them accordingly.
// Supported prefixes are:
//   - "env:": Expands to the value of the specified environment variable
//   - "vault:": Placeholder for retrieving secrets from Vault
//   - "file:": Reads the content of the specified file in secrets dir (if configured) and returns it as a string
//
// If no known prefix is found, the original string is returned unchanged.
// expand is a custom expansion function that uses the PropertyResolver registry
// It retrieves the corresponding value based on the prefix and returns it.
// If no known prefix is found, it returns the original string unchanged.
func expand(s string) string {
	// Use the global resolver registry to resolve the property
	value, err := resolver.Global.Resolve(s)
	if err != nil {
		panic(errors.Wrapf(err, "error resolving property %q", s))
	}
	return value
}

// expandVariables recursively traverses the fields of a struct and expands environment variables in string fields.
// It handles nested structs, pointers to structs, slices, and maps.
func expandVariables(val reflect.Value) {
	switch val.Kind() {
	case reflect.String:
		if val.CanSet() {
			val.SetString(os.Expand(val.String(), expand))
		}
	case reflect.Struct:
		for i := 0; i < val.NumField(); i++ {
			expandVariables(val.Field(i))
		}
	case reflect.Ptr:
		if !val.IsNil() {
			expandVariables(val.Elem())
		}
	case reflect.Slice:
		for j := 0; j < val.Len(); j++ {
			expandVariables(val.Index(j))
		}
	case reflect.Map:
		for _, key := range val.MapKeys() {
			mapVal := val.MapIndex(key)
			// Create a new addressable value of the same type
			newVal := reflect.New(mapVal.Type()).Elem()
			newVal.Set(mapVal)
			// Expand variables in the new value
			expandVariables(newVal)
			// Set the expanded value back into the map
			val.SetMapIndex(key, newVal)
		}
	default:
		return
	}
}
