package config

import (
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

// ControllerBinding represents the configuration for a single controller.
type ControllerBinding struct {
	TypeName   string           `yaml:"type"`
	Name       string           `yaml:"name,omitempty"`
	ConfigData ControllerConfig `yaml:"config"`
}

type ControllerBindings []ControllerBinding

func (c ControllerBindings) Validate() error {
	var validationErrors []error
	for i, binding := range c {
		if err := binding.Validate(); err != nil {
			validationErrors = append(validationErrors, errors.Wrapf(err, "controller binding at index %d is invalid", i))
		}
	}

	if len(validationErrors) > 0 {
		return errors.Errorf("configuration validation failed: %v", validationErrors)
	}

	return nil
}

// ControllerConfig is a raw YAML byte slice that can be unmarshaled into specific controller configurations.
type ControllerConfig []byte

// Validate checks if the ControllerBinding has all required fields set.
// Note: Name is optional and will be auto-generated if not provided.
func (c ControllerBinding) Validate() error {
	if c.TypeName == "" {
		return errors.New("controller type must be set and non-empty")
	}
	if c.ConfigData == nil {
		return errors.New("controller config must be provided")
	}
	return nil
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
