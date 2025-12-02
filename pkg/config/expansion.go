package config

import (
	"os"
	"reflect"
	"strings"

	"github.com/animalet/sargantana-go/pkg/config/secrets"
	"github.com/pkg/errors"
)

// expand checks for specific prefixes in the string and expands them accordingly.
// Supported prefixes are:
//   - "env:": Expands to the value of the specified environment variable
//   - "vault:": Placeholder for retrieving secrets from Vault
//   - "file:": Reads the content of the specified file in secrets dir (if configured) and returns it as a string
//
// If no known prefix is found, the original string is returned unchanged.
// expand is a custom expansion function that uses the secrets resolution system
// It retrieves the corresponding value based on the prefix and returns it.
// If no known prefix is found, it returns the original string unchanged.
func expand(s string) (string, error) {
	// Use the secrets resolution system to resolve the property
	value, err := secrets.Resolve(s)
	if err != nil {
		return "", errors.Wrap(err, "error resolving property")
	}
	return value, nil
}

// expandVariables recursively traverses the fields of a struct and expands environment variables in string fields.
// It handles nested structs, pointers to structs, slices, and maps.
func expandVariables(val reflect.Value) error {
	switch val.Kind() {
	case reflect.String:
		if val.CanSet() {
			var expandErr error
			expanded := os.Expand(strings.TrimSpace(val.String()), func(s string) string {
				if expandErr != nil {
					return ""
				}
				res, err := expand(s)
				if err != nil {
					expandErr = err
					return ""
				}
				return res
			})
			if expandErr != nil {
				return expandErr
			}
			val.SetString(expanded)
		}
	case reflect.Struct:
		for i := 0; i < val.NumField(); i++ {
			if err := expandVariables(val.Field(i)); err != nil {
				return err
			}
		}
	case reflect.Ptr:
		if !val.IsNil() {
			if err := expandVariables(val.Elem()); err != nil {
				return err
			}
		}
	case reflect.Slice:
		for j := 0; j < val.Len(); j++ {
			if err := expandVariables(val.Index(j)); err != nil {
				return err
			}
		}
	case reflect.Map:
		for _, key := range val.MapKeys() {
			mapVal := val.MapIndex(key)
			// Create a new addressable value of the same type
			newVal := reflect.New(mapVal.Type()).Elem()
			newVal.Set(mapVal)
			// Expand variables in the new value
			if err := expandVariables(newVal); err != nil {
				return err
			}
			// Set the expanded value back into the map
			val.SetMapIndex(key, newVal)
		}
	default:
		return nil
	}
	return nil
}
