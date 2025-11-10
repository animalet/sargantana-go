package server

import (
	"github.com/animalet/sargantana-go/pkg/config"
	"github.com/pkg/errors"
)

// ControllerBinding represents the configuration for a single controller.
type ControllerBinding struct {
	Config   config.ModuleRawConfig `yaml:"config"`
	TypeName string                 `yaml:"type"`
	Name     string                 `yaml:"name,omitempty"`
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

// Validate checks if the ControllerBinding has all required fields set.
// Note: Name is optional and will be auto-generated if not provided.
func (c ControllerBinding) Validate() error {
	if c.TypeName == "" {
		return errors.New("controller type must be set and non-empty")
	}
	if c.Config == nil {
		return errors.New("controller config must be provided")
	}
	return nil
}
