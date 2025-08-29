// Package config provides configuration management for the Sargantana Go web framework.
// It handles server configuration including address, session storage, secrets directory,
// debug mode, and session naming.
package config

type (
	// Config holds the configuration settings for the Sargantana Go server.
	// It encapsulates all necessary configuration parameters including network settings,
	// session storage options, security settings, and debugging preferences.
	Config struct {
		ServerConfig       ServerConfig        `yaml:"serverconfig"`
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

	ControllerConfig map[string]any

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
