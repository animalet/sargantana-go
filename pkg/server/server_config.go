package server

import (
	"fmt"
	"net"

	"github.com/pkg/errors"
)

type WebServerConfig struct {
	Address       string `yaml:"address"`
	SessionName   string `yaml:"session_name"`
	SessionSecret string `yaml:"session_secret"`
}

func (c WebServerConfig) Validate() error {
	if c.SessionSecret == "" {
		return errors.New("session_secret must be set and non-empty")
	}

	if c.SessionName == "" {
		return errors.New("session_name must be set and non-empty")
	}

	if c.Address == "" {
		return errors.New("address must be set and non-empty")
	}

	_, err := net.ResolveTCPAddr("tcp", c.Address)
	if err != nil {
		return fmt.Errorf("invalid address: %w", err)
	}

	return nil
}

// SargantanaConfig holds the configuration settings for the Sargantana Go server.
type SargantanaConfig struct {
	WebServerConfig    WebServerConfig    `yaml:"server"`
	ControllerBindings ControllerBindings `yaml:"controllers"`
}

func (c SargantanaConfig) Validate() error {
	if err := c.WebServerConfig.Validate(); err != nil {
		return errors.Wrap(err, "server configuration is invalid")
	}

	return c.ControllerBindings.Validate()
}
