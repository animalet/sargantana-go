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
	"gopkg.in/yaml.v3"
)

type Config struct {
	modules map[string]ModuleRawConfig
}

type ModuleRawConfig []byte

func Unmarshal[T Validatable](r ModuleRawConfig) (config *T, err error) {
	err = unmarshal(r, &config)
	if err != nil {
		return nil, err
	}
	return doExpand(config)
}

type Validatable interface {
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
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read configuration file: %s", path)
	}
	var modules map[string]ModuleRawConfig
	err = unmarshal(data, &modules)
	if err != nil {
		return nil, errors.Wrapf(err, "error marshalling to %s", format)
	}
	return &Config{modules: modules}, nil
}

func Get[T Validatable](c *Config, name string) (*T, error) {
	raw, ok := c.modules[name]
	if !ok {
		// If the module is missing, we might want to return nil, nil or an error.
		// Returning nil, nil allows checking for optional configs.
		return nil, nil
	}

	var partial *T
	err := unmarshal(raw, &partial)
	if err != nil {
		return nil, err
	}
	return doExpand(partial)
}

func doExpand[T Validatable](toExpand *T) (*T, error) {
	expandVariables(reflect.ValueOf(toExpand).Elem())
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
