package server

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSecrets_SetsEnvVars(t *testing.T) {
	secrets := map[string]string{
		"MY_SECRET1": "supersecret",
		"MY_SECRET2": "123123123123",
		"MY_SECRET3": "asd123asd123",
	}

	dir := t.TempDir()
	for name, value := range secrets {
		secretFile := filepath.Join(dir, name)
		os.WriteFile(secretFile, []byte("   "+value+" \n  \t"), 0644)
	}

	s := &Server{secretsDir: dir, debug: false}
	s.loadSecrets()

	// assert that the environment variables are set correctly
	for name, expected := range secrets {
		if got := os.Getenv(name); got != expected {
			t.Errorf("Expected %s=%q, got %s=%q", name, expected, name, got)
		}
	}
}
