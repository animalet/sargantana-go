// Package expansion provides internal variable expansion functionality for configuration values.
// It handles expansion of environment variables and secret references in configuration strings.
package expansion

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
func expand(s string) (string, error) {
	value, err := secrets.Resolve(s)
	if err != nil {
		return "", errors.Wrap(err, "error resolving property")
	}
	return value, nil
}

// ExpandVariables recursively traverses the fields of a struct and expands environment variables in string fields.
// It handles nested structs, pointers to structs, slices, and maps.
// The toExpand parameter must be a pointer to the value to expand.
func ExpandVariables(toExpand any) error {
	if toExpand == nil {
		return nil
	}

	v := reflect.ValueOf(toExpand)

	// Handle pointer: dereference to get the actual value
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil
		}
		return expandValue(v.Elem())
	}

	return expandValue(v)
}

// expandValue is the internal recursive function that operates on reflect.Value
func expandValue(val reflect.Value) error {
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
			if err := expandValue(val.Field(i)); err != nil {
				return err
			}
		}

	case reflect.Ptr:
		if !val.IsNil() {
			if err := expandValue(val.Elem()); err != nil {
				return err
			}
		}

	case reflect.Slice:
		for j := 0; j < val.Len(); j++ {
			if err := expandValue(val.Index(j)); err != nil {
				return err
			}
		}

	case reflect.Map:
		for _, key := range val.MapKeys() {
			mapVal := val.MapIndex(key)
			// Create a new addressable value of the same type
			newVal := reflect.New(mapVal.Type()).Elem()
			newVal.Set(mapVal)
			if err := expandValue(newVal); err != nil {
				return err
			}
			val.SetMapIndex(key, newVal)
		}
	default:
		// No action needed for other kinds
	}

	return nil
}
