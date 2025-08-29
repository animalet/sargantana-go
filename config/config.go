// Package config provides configuration management for the Sargantana Go web framework.
// It handles server configuration including address, session storage, secrets directory,
// debug mode, and session naming.
package config

import (
	"os"

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

func UnmarshalYAML[T any](data []byte) (*T, error) {
	var s T
	if err := yaml.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

func UnmarshalYAMLFromMap[T any](data map[string]any) (*T, error) {
	yamlBytes, err := yaml.Marshal(data)
	if err != nil {
		return nil, err
	}

	var s T
	if err := yaml.Unmarshal(yamlBytes, &s); err != nil {
		return nil, err
	}
	return &s, nil
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

// To unmarshals the raw YAML data from ControllerConfig into a target struct.
// The 'out' parameter must be a pointer to a struct that can be unmarshalled from YAML.
func (c *ControllerConfig) To(out any) error {
	bytes := []byte(*c)
	return yaml.Unmarshal(bytes, out)
}

// UnmarshalToNew unmarshals the raw YAML data from ControllerConfig into a new instance of type T.
// This method creates a new instance and returns it, avoiding addressability issues.
func UnmarshalToNew[T any](c ControllerConfig) (*T, error) {
	var result T
	data := make([]byte, len(c))
	copy(data, c)
	err := yaml.Unmarshal(data, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}
