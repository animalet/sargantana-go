package config

import "github.com/pkg/errors"

// ServerConfig holds the core server configuration parameters.
type ServerConfig struct {
	Address       string `yaml:"address"`
	SessionName   string `yaml:"session_name"`
	SessionSecret string `yaml:"session_secret"`
}

// Validate checks if the ServerConfig has all required fields set.
func (c ServerConfig) Validate() error {
	if c.SessionSecret == "" {
		return errors.New("session_secret must be set and non-empty")
	}
	return nil
}
