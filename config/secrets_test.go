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
				address: "https://vault.example.com:8200",
				path:    "secret/data/test",
			},
		},
		{
			name: "missing address",
			vaultConfig: VaultConfig{
				token: "test-token",
				path:  "secret/data/test",
			},
		},
		{
			name: "missing path",
			vaultConfig: VaultConfig{
				address: "https://vault.example.com:8200",
				token:   "test-token",
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
	err := os.WriteFile(filepath.Join(tempDir, "FILE_SECRET"), []byte("file_value"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test secret file: %v", err)
	}

	// Create config with both directory and empty Vault config
	config := &Config{
		secretsDir:  tempDir,
		vaultConfig: VaultConfig{}, // Empty Vault config should be skipped
	}

	// Clear any existing environment variable
	_ = os.Unsetenv("FILE_SECRET")

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
	_ = os.Unsetenv("FILE_SECRET")
}

func TestVaultConfig_Getters(t *testing.T) {
	vaultConfig := VaultConfig{
		address:   "https://vault.example.com:8200",
		token:     "test-token",
		path:      "secret/data/myapp",
		namespace: "test-namespace",
	}

	config := &Config{
		vaultConfig: vaultConfig,
	}

	retrievedConfig := config.VaultConfig()

	if retrievedConfig.address != vaultConfig.address {
		t.Errorf("Expected Address %s, got %s", vaultConfig.address, retrievedConfig.address)
	}
	if retrievedConfig.token != vaultConfig.token {
		t.Errorf("Expected Token %s, got %s", vaultConfig.token, retrievedConfig.token)
	}
	if retrievedConfig.path != vaultConfig.path {
		t.Errorf("Expected Path %s, got %s", vaultConfig.path, retrievedConfig.path)
	}
	if retrievedConfig.namespace != vaultConfig.namespace {
		t.Errorf("Expected Namespace %s, got %s", vaultConfig.namespace, retrievedConfig.namespace)
	}
}

// TestVaultIntegration_DockerContainer tests the integration with a real Vault Docker container
// This test requires the Vault container to be running (docker-compose up vault)
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

	tests := []struct {
		name        string
		vaultConfig VaultConfig
		expectError bool
		expectEnvs  map[string]string
	}{
		{
			name: "successful connection to vault container with KV v2",
			vaultConfig: VaultConfig{
				address: "http://localhost:8200",
				token:   "dev-root-token",
				path:    "secret/data/sargantana",
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
			vaultConfig: VaultConfig{
				address: "http://localhost:8200",
				token:   "dev-root-token",
				path:    "secret/sargantana",
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
			vaultConfig: VaultConfig{
				address: "http://localhost:8200",
				token:   "invalid-token",
				path:    "secret/data/sargantana",
			},
			expectError: true,
		},
		{
			name: "non-existent path",
			vaultConfig: VaultConfig{
				address: "http://localhost:8200",
				token:   "dev-root-token",
				path:    "secret/data/nonexistent",
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
			err := LoadSecretsFromVault(tt.vaultConfig)

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
}

// TestLoadSecrets_IntegrationWithVaultContainer tests the complete LoadSecrets function
// with both file-based and Vault-based secrets using the Docker container
func TestLoadSecrets_IntegrationWithVaultContainer(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	vaultAddr := "http://localhost:8200"
	if !isVaultReachable(vaultAddr) {
		t.Skip("Vault container not reachable. Run 'docker-compose up vault' first")
	}

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
	config := &Config{
		secretsDir: tempDir,
		vaultConfig: VaultConfig{
			address: "http://localhost:8200",
			token:   "dev-root-token",
			path:    "secret/data/sargantana",
		},
	}

	// Clear environment variables
	_ = os.Unsetenv("GOOGLE_KEY")
	_ = os.Unsetenv("GOOGLE_SECRET")
	_ = os.Unsetenv("SESSION_SECRET")
	_ = os.Unsetenv("FILE_ONLY_SECRET")

	// Load secrets
	err = LoadSecrets(config)
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
}

// TestVaultHealthCheck tests that we can properly detect if Vault is healthy
func TestVaultHealthCheck(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	vaultAddr := "http://localhost:8200"
	if !isVaultReachable(vaultAddr) {
		t.Skip("Vault container not reachable")
	}

	// Test health endpoint
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
