package config

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

func LoadSecretsFromDir(dir string) error {
	if dir == "" {
		log.Println("No secrets directory configured, skipping secrets loading")
		return nil
	}

	files, err := os.ReadDir(dir)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("error reading secrets directory %s", dir))
	}

	count := 0
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		name := file.Name()
		content, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return fmt.Errorf("error reading secret file %s: %v", name, err)
		}
		err = os.Setenv(strings.ToUpper(name), strings.TrimSpace(string(content)))
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("error setting environment variable %s", strings.ToUpper(name)))
		} else {
			count += 1
		}
	}

	return nil
}
