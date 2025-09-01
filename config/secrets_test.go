package config

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadSecrets(t *testing.T) {
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
		{
			name: "file secrets with directories to skip",
			setupFunc: func(dir string) error {
				err := os.WriteFile(filepath.Join(dir, "FILE_SECRET"), []byte("file_value"), 0644)
				if err != nil {
					return err
				}
				// Create a directory to test skipping behavior
				return os.Mkdir(filepath.Join(dir, "secret_as_dir"), 0755)
			},
			expectError: false,
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

			err := LoadSecretsFromDir(secretsDir)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Check that file secret was loaded for the directory skipping test
			if tt.name == "file secrets with directories to skip" && err == nil {
				value := os.Getenv("FILE_SECRET")
				if value != "file_value" {
					t.Errorf("Expected FILE_SECRET to be 'file_value', got '%s'", value)
				}
				// Clean up
				_ = os.Unsetenv("FILE_SECRET")
			}
		})
	}
}

func TestLoadSecretsFromVault_EmptyConfig(t *testing.T) {
	// Test with empty Vault configuration
	c := Config{}
	err := c.LoadSecretsFromVault()
	if err != nil {
		t.Errorf("Expected no error with empty config, got: %v", err)
	}
}

func TestLoadSecretsFromVault_PartialConfig(t *testing.T) {
	tests := []struct {
		name   string
		config Config
	}{
		{
			name: "missing token",
			config: Config{
				Vault: VaultConfig{
					Address: "https://vault.example.com:8200",
					Path:    "secret/data/test",
				},
			},
		},
		{
			name: "missing address",
			config: Config{
				Vault: VaultConfig{
					Token: "test-token",
					Path:  "secret/data/test",
				},
			},
		},
		{
			name: "missing path",
			config: Config{
				Vault: VaultConfig{
					Address: "https://vault.example.com:8200",
					Token:   "test-token",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.LoadSecretsFromVault()
			if err != nil {
				t.Errorf("Expected no error with partial config, got: %v", err)
			}
		})
	}
}

// TestVaultIntegration_DockerContainer tests the integration with a real Vault Docker container
// This comprehensive test covers Vault operations, health checks, and complete LoadSecrets integration
func TestVaultIntegration_DockerContainer(t *testing.T) {
	// Skip this test if not running integration tests
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Check if Vault container is running by attempting to connect
	vaultAddr := "http://localhost:8200"
	if !isVaultReachable(vaultAddr) {
		t.Skip("Vault container not reachable at http://localhost:8200. Run 'docker-compose up vault' first")
	}

	// Test Vault health check first
	t.Run("vault health check", func(t *testing.T) {
		resp, err := http.Get(vaultAddr + "/v1/sys/health")
		if err != nil {
			t.Fatalf("Failed to check Vault health: %v", err)
		}
		defer func() {
			err := resp.Body.Close()
			if err != nil {
				t.Errorf("Failed to close response body: %v", err)
			}
		}()

		// Vault dev mode returns 200 for unsealed, initialized state
		if resp.StatusCode != 200 {
			t.Errorf("Expected Vault to be healthy (status 200), got status %d", resp.StatusCode)
		}
	})

	tests := []struct {
		name        string
		config      Config
		expectError bool
		expectEnvs  map[string]string
	}{
		{
			name: "successful connection to vault container with KV v2",
			config: Config{
				Vault: VaultConfig{
					Address: "http://localhost:8200",
					Token:   "dev-root-token",
					Path:    "secret/data/sargantana",
				},
			},
			expectError: false,
			expectEnvs: map[string]string{
				"GOOGLE_KEY":     "test-google-key",
				"GOOGLE_SECRET":  "test-google-secret",
				"SESSION_SECRET": "test-session-secret-that-is-long-enough",
			},
		},
		{
			name: "successful connection to vault container with KV v1",
			config: Config{
				Vault: VaultConfig{
					Address: "http://localhost:8200",
					Token:   "dev-root-token",
					Path:    "secret-v1/sargantana",
				},
			},
			expectError: false,
			expectEnvs: map[string]string{
				"GOOGLE_KEY":     "test-google-key",
				"GOOGLE_SECRET":  "test-google-secret",
				"SESSION_SECRET": "test-session-secret-that-is-long-enough",
			},
		},
		{
			name: "invalid token",
			config: Config{
				Vault: VaultConfig{
					Address: "http://localhost:8200",
					Token:   "invalid-token",
					Path:    "secret/data/sargantana",
				},
			},
			expectError: true,
		},
		{
			name: "non-existent path",
			config: Config{
				Vault: VaultConfig{
					Address: "http://localhost:8200",
					Token:   "dev-root-token",
					Path:    "secret/data/nonexistent",
				},
			},
			expectError: false, // Should not error, just no secrets loaded
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment variables before test
			for envKey := range tt.expectEnvs {
				_ = os.Unsetenv(envKey)
			}

			// Run the test
			err := tt.config.LoadSecretsFromVault()

			// Check error expectation
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
				return
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// If no error expected, check environment variables
			if !tt.expectError && err == nil {
				for envKey, expectedValue := range tt.expectEnvs {
					actualValue := os.Getenv(envKey)
					if actualValue != expectedValue {
						t.Errorf("Expected %s to be '%s', got '%s'", envKey, expectedValue, actualValue)
					}
				}
			}

			// Clean up environment variables after test
			for envKey := range tt.expectEnvs {
				_ = os.Unsetenv(envKey)
			}
		})
	}

	// Test complete LoadSecrets integration with both file and Vault secrets
	t.Run("complete LoadSecrets integration with file and vault secrets", func(t *testing.T) {
		// Create temporary directory for file-based secrets
		tempDir := t.TempDir()

		// Create a test secret file that will be overridden by Vault
		err := os.WriteFile(filepath.Join(tempDir, "GOOGLE_KEY"), []byte("file-google-key"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test secret file: %v", err)
		}

		// Create a file-only secret (not in Vault)
		err = os.WriteFile(filepath.Join(tempDir, "FILE_ONLY_SECRET"), []byte("file-only-value"), 0644)
		if err != nil {
			t.Fatalf("Failed to create file-only secret: %v", err)
		}

		// Create config with both directory and Vault config
		c := &Config{
			ServerConfig: ServerConfig{
				SecretsDir: tempDir,
			},
			Vault: VaultConfig{
				Address: "http://localhost:8200",
				Token:   "dev-root-token",
				Path:    "secret/data/sargantana",
			},
		}

		// Clear environment variables
		_ = os.Unsetenv("GOOGLE_KEY")
		_ = os.Unsetenv("GOOGLE_SECRET")
		_ = os.Unsetenv("SESSION_SECRET")
		_ = os.Unsetenv("FILE_ONLY_SECRET")

		// Load secrets
		err = c.LoadSecrets()
		if err != nil {
			t.Fatalf("LoadSecrets failed: %v", err)
		}

		// Verify that Vault secrets override file secrets
		googleKey := os.Getenv("GOOGLE_KEY")
		if googleKey != "test-google-key" {
			t.Errorf("Expected GOOGLE_KEY to be 'test-google-key' (from Vault), got '%s'", googleKey)
		}

		// Verify Vault-only secrets are loaded
		googleSecret := os.Getenv("GOOGLE_SECRET")
		if googleSecret != "test-google-secret" {
			t.Errorf("Expected GOOGLE_SECRET to be 'test-google-secret', got '%s'", googleSecret)
		}

		sessionSecret := os.Getenv("SESSION_SECRET")
		if sessionSecret != "test-session-secret-that-is-long-enough" {
			t.Errorf("Expected SESSION_SECRET to be 'test-session-secret-that-is-long-enough', got '%s'", sessionSecret)
		}

		// Verify file-only secrets are still loaded
		fileOnlySecret := os.Getenv("FILE_ONLY_SECRET")
		if fileOnlySecret != "file-only-value" {
			t.Errorf("Expected FILE_ONLY_SECRET to be 'file-only-value', got '%s'", fileOnlySecret)
		}

		// Clean up
		_ = os.Unsetenv("GOOGLE_KEY")
		_ = os.Unsetenv("GOOGLE_SECRET")
		_ = os.Unsetenv("SESSION_SECRET")
		_ = os.Unsetenv("FILE_ONLY_SECRET")
	})
}

// isVaultReachable checks if Vault is reachable at the given address
func isVaultReachable(addr string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", addr+"/v1/sys/health", nil)
	if err != nil {
		return false
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	err = resp.Body.Close()
	if err != nil {
		return false
	}

	// Accept any response (2xx, 4xx, 5xx) as "reachable"
	// Vault might return different status codes based on its state
	return true
}
