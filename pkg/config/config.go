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

type Config map[string]ModuleRawConfig
type ModuleRawConfig []byte

func Unmarshal[T Validatable](r ModuleRawConfig) (config *T, err error) {
	err = unmarshal(r, &config)
	if err != nil {
		return nil, err
	}
	return doExpand(config)
}

// Validatable interface defines types that can be validated.
type Validatable interface {
	Validate() error
}

// ClientFactory is a generic interface for configurations that can create clients
// for data sources like Vault, Redis, databases, etc.
// The type parameter T specifies the concrete client type that will be returned.
// Implementations should create and configure the appropriate client type
// from their configuration details.
//
// Example implementations:
//   - VaultConfig implements ClientFactory[*api.Client]
//   - RedisConfig implements ClientFactory[*redis.Pool]
//
// Usage:
//
//	client, err := cfg.Vault.CreateClient()  // Returns (*api.Client, error) directly
type ClientFactory[T any] interface {
	Validatable
	// CreateClient creates and configures a client from the Config details.
	// Returns the strongly-typed client T and an error if creation fails.
	CreateClient() (T, error)
}

func ReadModular(path string) (cfg Config, err error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read configuration file: %s", path)
	}
	err = unmarshal(data, &cfg)
	if err != nil {
		return nil, errors.Wrapf(err, "error marshalling to %s", format)
	}
	return cfg, nil
}

func ReadFull[T Validatable](path string) (full *T, err error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read configuration file: %s", path)
	}
	err = unmarshal(data, &full)
	if err != nil {
		return nil, err
	}

	return doExpand(full)
}

func Load[T Validatable](cfg ModuleRawConfig) (partial *T, err error) {
	err = unmarshal(cfg, &partial)
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

// UnmarshalYAML implements custom YAML unmarshaling for ModuleRawConfig
func (m *ModuleRawConfig) UnmarshalYAML(value *yaml.Node) error {
	// Re-marshal the node to get raw bytes
	data, err := yaml.Marshal(value)
	if err != nil {
		return err
	}
	*m = data
	return nil
}

// UnmarshalJSON implements custom JSON unmarshaling
func (m *ModuleRawConfig) UnmarshalJSON(data []byte) error {
	*m = data
	return nil
}

// UnmarshalTOML implements custom TOML unmarshaling
func (m *ModuleRawConfig) UnmarshalTOML(data interface{}) error {
	bytes, err := toml.Marshal(data)
	if err != nil {
		return err
	}
	*m = bytes
	return nil
}

// UnmarshalXML implements custom XML unmarshaling
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
