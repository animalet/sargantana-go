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

func TestLoadSecretsFromVault_EmptyConfig(t *testing.T) {
	// Test with empty Vault configuration
	vaultConfig := VaultConfig{}
	err := LoadSecretsFromVault(vaultConfig)
	if err != nil {
		t.Errorf("Expected no error with empty config, got: %v", err)
	}
}

func TestLoadSecretsFromVault_PartialConfig(t *testing.T) {
	tests := []struct {
		name        string
		vaultConfig VaultConfig
	}{
		{
			name: "missing token",
			vaultConfig: VaultConfig{
				Address: "https://vault.example.com:8200",
				Path:    "secret/data/test",
			},
		},
		{
			name: "missing address",
			vaultConfig: VaultConfig{
				Token: "test-token",
				Path:  "secret/data/test",
			},
		},
		{
			name: "missing path",
			vaultConfig: VaultConfig{
				Address: "https://vault.example.com:8200",
				Token:   "test-token",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := LoadSecretsFromVault(tt.vaultConfig)
			if err != nil {
				t.Errorf("Expected no error with partial config, got: %v", err)
			}
		})
	}
}

func TestLoadSecrets_Integration(t *testing.T) {
	// Create temporary directory for file-based secrets
	tempDir := t.TempDir()

	// Create a test secret file
	err := os.WriteFile(filepath.Join(tempDir, "file_secret"), []byte("file_value"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test secret file: %v", err)
	}

	// Create config with both directory and empty Vault config
	config := &Config{
		secretsDir:  tempDir,
		vaultConfig: VaultConfig{}, // Empty Vault config should be skipped
	}

	// Clear any existing environment variable
	os.Unsetenv("FILE_SECRET")

	err = LoadSecrets(config)
	if err != nil {
		t.Errorf("LoadSecrets failed: %v", err)
	}

	// Check that file secret was loaded
	value := os.Getenv("FILE_SECRET")
	if value != "file_value" {
		t.Errorf("Expected FILE_SECRET to be 'file_value', got '%s'", value)
	}

	// Clean up
	os.Unsetenv("FILE_SECRET")
}

func TestVaultConfig_Getters(t *testing.T) {
	vaultConfig := VaultConfig{
		Address:   "https://vault.example.com:8200",
		Token:     "test-token",
		Path:      "secret/data/myapp",
		Namespace: "test-namespace",
	}

	config := &Config{
		vaultConfig: vaultConfig,
	}

	retrievedConfig := config.VaultConfig()

	if retrievedConfig.Address != vaultConfig.Address {
		t.Errorf("Expected Address %s, got %s", vaultConfig.Address, retrievedConfig.Address)
	}
	if retrievedConfig.Token != vaultConfig.Token {
		t.Errorf("Expected Token %s, got %s", vaultConfig.Token, retrievedConfig.Token)
	}
	if retrievedConfig.Path != vaultConfig.Path {
		t.Errorf("Expected Path %s, got %s", vaultConfig.Path, retrievedConfig.Path)
	}
	if retrievedConfig.Namespace != vaultConfig.Namespace {
		t.Errorf("Expected Namespace %s, got %s", vaultConfig.Namespace, retrievedConfig.Namespace)
	}
}
