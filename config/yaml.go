package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

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
