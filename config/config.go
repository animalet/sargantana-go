// Package config provides configuration management for the Sargantana Go web framework.
// It handles server configuration including address, session storage, secrets directory,
// debug mode, and session naming.
package config

import (
	"log"
	"os"
	"reflect"

	"gopkg.in/yaml.v3"
)

type (
	// Config holds the configuration settings for the Sargantana Go server.
	// It encapsulates all necessary configuration parameters including network settings,
	// session storage options, security settings, and debugging preferences.
	Config struct {
		ServerConfig       ServerConfig        `yaml:"server"`
		Vault              VaultConfig         `yaml:"vault,omitempty"`
		ControllerBindings []ControllerBinding `yaml:"controllers"`
	}

	// ServerConfig holds the core server configuration parameters.
	ServerConfig struct {
		Address           string `yaml:"address"`
		RedisSessionStore string `yaml:"redis_session_store"`
		SecretsDir        string `yaml:"secrets_dir"`
		Debug             bool   `yaml:"debug"`
		SessionName       string `yaml:"session_name"`
		SessionSecret     string `yaml:"session_secret"`
	}

	// ControllerBinding represents the configuration for a single controller.
	ControllerBinding struct {
		TypeName   string           `yaml:"type"`
		Name       string           `yaml:"name,omitempty"`
		ConfigData ControllerConfig `yaml:"config"`
	}

	ControllerConfig []byte

	// VaultConfig holds configuration for connecting to HashiCorp Vault
	VaultConfig struct {
		Address   string `yaml:"address"`
		Token     string `yaml:"token"`
		Path      string `yaml:"path"`
		Namespace string `yaml:"namespace"`
	}
)

func Load(file string) (*Config, error) {
	var cfg *Config
	err := LoadYaml(file, &cfg)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

// IsValid checks if the VaultConfig has all required fields set.
func (v *VaultConfig) IsValid() bool {
	return v.Address != "" && v.Token != "" && v.Path != ""
}

// LoadYaml reads the YAML configuration file and unmarshalls its content into the provided struct.
// Parameters:
//   - out: A pointer to the struct where the configuration will be unmarshalled
//
// Returns an error if the file cannot be read or unmarshalled.
func LoadYaml(file string, out any) error {
	data, err := os.ReadFile(file)
	if err != nil {
		return err
	}

	return yaml.Unmarshal(data, out)
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
func UnmarshalTo[T any](c ControllerConfig) (*T, error) {
	if c == nil {
		return nil, nil
	}
	var result T
	err := yaml.Unmarshal(c, &result)
	if err != nil {
		return nil, err
	}

	// Always try to expand environment variables for structs
	v := reflect.ValueOf(&result).Elem()
	if v.Kind() == reflect.Struct {
		expandEnv(v)
	}

	return &result, nil
}

// expandEnv recursively traverses the fields of a struct and expands environment variables in string fields.
// It handles nested structs, pointers to structs, slices, and maps.
func expandEnv(val reflect.Value) {
	switch val.Kind() {
	case reflect.String:
		if val.CanSet() {
			val.SetString(os.ExpandEnv(val.String()))
		}
	case reflect.Struct:
		for i := 0; i < val.NumField(); i++ {
			expandEnv(val.Field(i))
		}
	case reflect.Ptr:
		if !val.IsNil() {
			expandEnv(val.Elem())
		}
	case reflect.Slice:
		for j := 0; j < val.Len(); j++ {
			expandEnv(val.Index(j))
		}
	case reflect.Map:
		if val.Type().Elem().Kind() == reflect.String {
			for _, key := range val.MapKeys() {
				mapVal := val.MapIndex(key)
				if mapVal.Kind() == reflect.String {
					expanded := os.ExpandEnv(mapVal.String())
					val.SetMapIndex(key, reflect.ValueOf(expanded))
				}
			}
		} else {
			// Handle maps with non-string values recursively
			for _, key := range val.MapKeys() {
				mapVal := val.MapIndex(key)
				if mapVal.CanAddr() {
					expandEnv(mapVal.Addr().Elem())
				}
			}
		}
	default:
		log.Panicf("expandEnv: unsupported kind %s", val.Kind())
	}
}
