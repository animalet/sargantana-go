// Package config provides configuration management for the Sargantana Go web framework.
// It handles server configuration including address, session storage, secrets directory,
// debug mode, and session naming.
package config

import (
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

type (
	// Config holds the configuration settings for the Sargantana Go server.
	// It encapsulates all necessary configuration parameters including network settings,
	// session storage options, security settings, and debugging preferences.
	Config struct {
		ServerConfig       ServerConfig        `yaml:"server"`
		Vault              *VaultConfig        `yaml:"vault,omitempty"`
		ControllerBindings []ControllerBinding `yaml:"controllers"`
	}

	// ServerConfig holds the core server configuration parameters.
	ServerConfig struct {
		Address           string `yaml:"address"`
		RedisSessionStore string `yaml:"redis_session_store"`
		SecretsDir        string `yaml:"secrets_dir,omitempty"`
		Debug             bool   `yaml:"debug,omitempty"`
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

	if cfg.ServerConfig.SecretsDir == "" {
		log.Println("No secrets directory configured, file secrets will fail if requested")
	}

	secretDir = cfg.ServerConfig.SecretsDir
	if cfg.Vault.IsValid() && cfg.Vault != nil {
		err = cfg.createVaultManager()
		if err != nil {
			return nil, err
		}
	} else {
		log.Println("Vault configuration incomplete, skipping Vault secrets loading")
	}

	cfg.ServerConfig.SessionSecret = expand(cfg.ServerConfig.SessionSecret)
	if cfg.ServerConfig.SessionSecret == "" {
		return nil, errors.New("session_secret must be set and non-empty")
	}

	// Always try to expand environment variables for structs
	v := reflect.ValueOf(&cfg.ServerConfig).Elem()
	if v.Kind() == reflect.Struct {
		expandEnv(v)
	}
	return cfg, nil
}

// IsValid checks if the VaultConfig has all required fields set.
func (v *VaultConfig) IsValid() bool {
	return v != nil && v.Address != "" && v.Token != "" && v.Path != ""
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

// expand checks for specific prefixes in the string and expands them accordingly.
// Supported prefixes are:
//   - "env:": Expands to the value of the specified environment variable
//   - "vault:": Placeholder for future Vault integration (currently returns a static value)
//
// If no known prefix is found, the original string is returned unchanged.
const envPrefix = "env"
const filePrefix = "file"
const vaultPrefix = "vault"

func expand(s string) string {
	prefix, name, found := strings.Cut(s, envPrefix)
	switch {
	case !found || strings.EqualFold(prefix, envPrefix):
		return os.ExpandEnv(s)
	case strings.EqualFold(prefix, filePrefix):
		file, err := secretFromFile(name)
		if err != nil {
			panic(errors.Wrap(err, "error retrieving secret from Vault"))
		}
		return file
	case strings.EqualFold(prefix, vaultPrefix):
		fromVault, err := vaultManagerInstance.secret(name)
		if err != nil {
			panic(errors.Wrap(err, "error retrieving secret from Vault"))
		}
		if fromVault == nil {
			return ""
		}
		return *fromVault
	default:
		return s
	}
}

var secretDir string

// secretFromFile reads the content of a file and returns it as a trimmed string.
// It returns an error if the file cannot be read.
func secretFromFile(file string) (string, error) {
	if secretDir == "" {
		return "", errors.New("no secrets directory configured")
	}
	file = strings.TrimSpace(file)
	if file == "" {
		return "", errors.New("no file specified for file secret")
	}
	file = filepath.Join(secretDir, file)
	b, err := os.ReadFile(file)
	if err != nil {
		return "", errors.Wrap(err, "error reading secret file")
	}
	return strings.TrimSpace(string(b)), nil
}

// expandEnv recursively traverses the fields of a struct and expands environment variables in string fields.
// It handles nested structs, pointers to structs, slices, and maps.
func expandEnv(val reflect.Value) {
	switch val.Kind() {
	case reflect.String:
		if val.CanSet() {
			val.SetString(os.Expand(val.String(), expand))
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
					expanded := os.Expand(val.String(), expand)
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
		return
	}
}
