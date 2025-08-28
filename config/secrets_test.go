package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSecrets_LoadSecrets(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func(string) error
		secretsDir  string
		expectError bool
	}{
		{
			name: "valid secrets",
			setupFunc: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, "TEST_SECRET"), []byte("secret_value"), 0644)
			},
			expectError: false,
		},
		{
			name: "empty directory",
			setupFunc: func(dir string) error {
				return nil // Create empty directory
			},
			expectError: false,
		},
		{
			name:        "non-existent directory",
			setupFunc:   nil,
			secretsDir:  "/non/existent/path",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var secretsDir string
			if tt.secretsDir != "" {
				secretsDir = tt.secretsDir
			} else {
				secretsDir = t.TempDir()
				if tt.setupFunc != nil {
					err := tt.setupFunc(secretsDir)
					if err != nil {
						t.Fatalf("Setup failed: %v", err)
					}
				}
			}
			c := NewConfig("localhost:8080", "", secretsDir, false, "test")

			err := LoadSecretsFromDir(c.SecretsDir())

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestSecrets_LoadSecretsWithInvalidFile(t *testing.T) {
	tempDir := t.TempDir()

	// Create a directory instead of a file to test error handling
	secretDir := filepath.Join(tempDir, "secret_as_dir")
	err := os.Mkdir(secretDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	c := NewConfig("localhost:8080", "", tempDir, false, "test")

	err = LoadSecretsFromDir(c.SecretsDir())
	// Should not error on directories, they are skipped
	if err != nil {
		t.Errorf("loadSecrets should skip directories without error: %v", err)
	}
}
