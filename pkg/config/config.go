// Package config provides configuration management for the Sargantana Go web framework.
// It handles server configuration including address, session storage, secrets directory,
// debug mode, and session naming.
package config

import (
	"encoding/json"
	"encoding/xml"
	"os"
	"reflect"

	"github.com/pelletier/go-toml/v2"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
)

type Config struct {
	modules map[string]ModuleRawConfig
}

type ModuleRawConfig []byte

func Unmarshal[T Validatable](r ModuleRawConfig) (config *T, err error) {
	log.Debug().Msgf("Unmarshalling configuration of type %T", config)
	err = unmarshal(r, &config)
	if err != nil {
		return nil, err
	}
	return doExpand(config)
}

type Validatable interface {
	// Validate checks if the configuration is valid.
	// Returns an error if validation fails, nil otherwise.
	// Ideally, validation shouldn't be executed externally, but rather automatically during the loading/unmarshalling
	// process, so user code should only worry about providing a solid implementation of this method.
	// An exception to this is Validate methods of nested structs, where explicit validation
	// calls might be necessary for child structs.
	Validate() error
}

// ClientFactory is a generic interface for configurations that can create clients.
type ClientFactory[T any] interface {
	Validatable
	// CreateClient creates and configures a client from the Config details.
	// Returns the strongly-typed client T and an error if creation fails.
	CreateClient() (T, error)
}

func NewConfig(path string) (cfg *Config, err error) {
	log.Debug().Str("path", path).Msg("Loading configuration file")
	// #nosec G304 -- Config file path is provided by operator at startup, this is intentional
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read configuration file: %s", path)
	}
	var modules map[string]ModuleRawConfig
	err = unmarshal(data, &modules)
	if err != nil {
		return nil, errors.Wrapf(err, "error unmarshalling to %s", format)
	}
	return &Config{modules: modules}, nil
}

// Get loads a configuration by name and unmarshals it into the specified type T.
// T must implement the Validatable interface.
// Returns a pointer to the configuration T, or nil if the configuration is not present.
// Note: Validation is automatically performed by doExpand before this method returns.
func Get[T Validatable](c *Config, name string) (*T, error) {
	log.Debug().Str("config_name", name).Msg("Getting configuration")
	raw, ok := c.modules[name]
	if !ok {
		// If the module is missing, we might want to return nil, nil or an error.
		// Returning nil, nil allows checking for optional configs.
		log.Debug().Str("config_name", name).Msg("Configuration not found")
		return nil, nil
	}

	var partial *T
	err := unmarshal(raw, &partial)
	if err != nil {
		return nil, err
	}
	return doExpand(partial)
}

// GetClient loads a configuration by name and creates the corresponding client.
// T must be a type that implements ClientFactory[F].
// Returns a pointer to the client F, or nil if the configuration is not present.
// Note: Validation is automatically performed by Get before this method returns.
func GetClient[T ClientFactory[F], F any](c *Config, name string) (*F, error) {
	cfg, err := Get[T](c, name)
	if err != nil {
		return nil, err
	}
	if cfg == nil {
		return nil, nil
	}

	log.Debug().Str("config_name", name).Msg("Creating client from configuration")
	client, err := (*cfg).CreateClient()
	if err != nil {
		return nil, err
	}
	return &client, nil
}

// GetClientAndConfig loads a configuration by name and creates the corresponding client.
// T must be a type that implements ClientFactory[F].
// Returns pointers to both the client F and the config T, or nil for both if the configuration is not present.
// Note: Validation is automatically performed by Get before this method returns.
func GetClientAndConfig[T ClientFactory[F], F any](c *Config, name string) (*F, *T, error) {
	cfg, err := Get[T](c, name)
	if err != nil {
		return nil, nil, err
	}
	if cfg == nil {
		return nil, nil, nil
	}

	log.Debug().Str("config_name", name).Msg("Creating client from configuration")
	client, err := (*cfg).CreateClient()
	if err != nil {
		return nil, nil, err
	}
	return &client, cfg, nil
}

func doExpand[T Validatable](toExpand *T) (*T, error) {
	log.Debug().Msgf("Expanding variables for config type %T", toExpand)
	if err := expandVariables(reflect.ValueOf(toExpand).Elem()); err != nil {
		return nil, err
	}
	log.Debug().Msgf("Validating config type %T", toExpand)
	if err := (*toExpand).Validate(); err != nil {
		return nil, errors.Wrap(err, "configuration is invalid")
	}
	return toExpand, nil
}

var format = YamlFormat

func UseFormat(fId formatId) {
	format = fId
}

type formatId string

const (
	YamlFormat formatId = "yaml"
	JsonFormat formatId = "json"
	TomlFormat formatId = "toml"
	XmlFormat  formatId = "xml"
)

func unmarshal(in []byte, out any) error {
	switch format {
	case YamlFormat:
		return yaml.Unmarshal(in, out)
	case JsonFormat:
		return json.Unmarshal(in, out)
	case TomlFormat:
		return toml.Unmarshal(in, out)
	case XmlFormat:
		return xml.Unmarshal(in, out)
	default:
		return errors.Errorf("unsupported format: %s", format)
	}
}

func (m *ModuleRawConfig) UnmarshalYAML(value *yaml.Node) error {
	// Re-marshal the node to get raw bytes
	data, err := yaml.Marshal(value)
	if err != nil {
		return err
	}
	*m = data
	return nil
}

func (m *ModuleRawConfig) UnmarshalJSON(data []byte) error {
	*m = data
	return nil
}

func (m *ModuleRawConfig) UnmarshalTOML(data interface{}) error {
	bytes, err := toml.Marshal(data)
	if err != nil {
		return err
	}
	*m = bytes
	return nil
}

func (m *ModuleRawConfig) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var node interface{}
	if err := d.DecodeElement(&node, &start); err != nil {
		return err
	}

	// Re-marshal to get raw bytes
	data, err := xml.Marshal(node)
	if err != nil {
		return err
	}
	*m = data
	return nil
}
