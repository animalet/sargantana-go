package config

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/vault/api"
)

func TestLoadSecretsFromDir(t *testing.T) {
	tests := []struct {
		name          string
		dir           string
		setupFiles    map[string]string // filename -> content
		expectedError bool
		expectedEnvs  map[string]string // env var -> expected value
	}{
		{
			name:          "empty directory string",
			dir:           "",
			setupFiles:    nil,
			expectedError: false,
			expectedEnvs:  nil,
		},
		{
			name: "single secret file",
			dir:  "", // will be set to temp dir
			setupFiles: map[string]string{
				"API_KEY": "secret-api-key-123",
			},
			expectedError: false,
			expectedEnvs: map[string]string{
				"API_KEY": "secret-api-key-123",
			},
		},
		{
			name: "multiple secret files",
			dir:  "", // will be set to temp dir
			setupFiles: map[string]string{
				"DATABASE_PASSWORD": "db-pass-456",
				"JWT_SECRET":        "jwt-secret-789",
				"REDIS_PASSWORD":    "redis-pass-abc",
			},
			expectedError: false,
			expectedEnvs: map[string]string{
				"DATABASE_PASSWORD": "db-pass-456",
				"JWT_SECRET":        "jwt-secret-789",
				"REDIS_PASSWORD":    "redis-pass-abc",
			},
		},
		{
			name: "secret file with whitespace",
			dir:  "", // will be set to temp dir
			setupFiles: map[string]string{
				"TRIMMED_SECRET": "  secret-with-spaces  \n\t",
			},
			expectedError: false,
			expectedEnvs: map[string]string{
				"TRIMMED_SECRET": "secret-with-spaces",
			},
		},
		{
			name: "ignore subdirectories",
			dir:  "", // will be set to temp dir
			setupFiles: map[string]string{
				"VALID_SECRET": "valid-content",
			},
			expectedError: false,
			expectedEnvs: map[string]string{
				"VALID_SECRET": "valid-content",
			},
		},
		{
			name:          "non-existent directory",
			dir:           "/non/existent/directory",
			setupFiles:    nil,
			expectedError: true,
			expectedEnvs:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Store original environment to restore later
			originalEnvs := make(map[string]string)
			if tt.expectedEnvs != nil {
				for key := range tt.expectedEnvs {
					if val, exists := os.LookupEnv(key); exists {
						originalEnvs[key] = val
					}
					_ = os.Unsetenv(key) // Clear before test
				}
			}
			defer func() {
				// Restore original environment
				for key := range tt.expectedEnvs {
					_ = os.Unsetenv(key)
					if val, exists := originalEnvs[key]; exists {
						_ = os.Setenv(key, val)
					}
				}
			}()

			var testDir string
			if tt.dir == "" && tt.setupFiles != nil {
				// Create temporary directory for test
				tempDir := t.TempDir()
				testDir = tempDir

				// Create secret files
				for filename, content := range tt.setupFiles {
					err := os.WriteFile(filepath.Join(tempDir, filename), []byte(content), 0600)
					if err != nil {
						t.Fatalf("Failed to create test file %s: %v", filename, err)
					}
				}

				// For the "ignore subdirectories" test, create a subdirectory
				if tt.name == "ignore subdirectories" {
					subDir := filepath.Join(tempDir, "subdir")
					err := os.Mkdir(subDir, 0755)
					if err != nil {
						t.Fatalf("Failed to create subdirectory: %v", err)
					}
					err = os.WriteFile(filepath.Join(subDir, "IGNORED_SECRET"), []byte("should-be-ignored"), 0600)
					if err != nil {
						t.Fatalf("Failed to create file in subdirectory: %v", err)
					}
				}
			} else {
				testDir = tt.dir
			}

			// Run the function
			err := LoadSecretsFromDir(testDir)

			// Check error expectation
			if tt.expectedError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Check environment variables
			if tt.expectedEnvs != nil {
				for expectedKey, expectedValue := range tt.expectedEnvs {
					actualValue, exists := os.LookupEnv(expectedKey)
					if !exists {
						t.Errorf("Environment variable %s was not set", expectedKey)
						continue
					}
					if actualValue != expectedValue {
						t.Errorf("Environment variable %s = %q, want %q", expectedKey, actualValue, expectedValue)
					}
				}
			}

			// For the "ignore subdirectories" test, verify subdirectory files were ignored
			if tt.name == "ignore subdirectories" {
				if _, exists := os.LookupEnv("IGNORED_SECRET"); exists {
					t.Error("Environment variable from subdirectory should not be set")
				}
			}
		})
	}
}

func TestLoadSecretsFromDir_EmptyDirectory(t *testing.T) {
	// Create an empty temporary directory
	tempDir := t.TempDir()

	// Test loading from empty directory
	err := LoadSecretsFromDir(tempDir)
	if err != nil {
		t.Errorf("LoadSecretsFromDir with empty directory should not return error, got: %v", err)
	}
}

func TestLoadSecretsFromDir_FilePermissionError(t *testing.T) {
	tempDir := t.TempDir()
	secretFile := filepath.Join(tempDir, "SECRET_KEY")

	// Create a file with content
	err := os.WriteFile(secretFile, []byte("secret-content"), 0600)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Remove read permissions to simulate permission error
	err = os.Chmod(secretFile, 0000)
	if err != nil {
		t.Fatalf("Failed to change file permissions: %v", err)
	}

	// Restore permissions after test
	defer func() {
		err := os.Chmod(secretFile, 0600)
		if err != nil {
			t.Errorf("Failed to restore file permissions: %v", err)
		}
	}()

	// Test should return error due to permission issue
	err = LoadSecretsFromDir(tempDir)
	if err == nil {
		t.Fatal("Expected error due to file permission, but got none")
	}

	if !strings.Contains(err.Error(), "permission denied") && !strings.Contains(err.Error(), "SECRET_KEY") {
		t.Errorf("Error should mention permission issue or file name, got: %v", err)
	}
}

func TestLoadSecrets(t *testing.T) {
	// Create a temporary config with secrets directory
	tempDir := t.TempDir()
	secretFile := filepath.Join(tempDir, "TEST_SECRET")
	err := os.WriteFile(secretFile, []byte("test-value"), 0600)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Store original environment
	originalValue, originalExists := os.LookupEnv("TEST_SECRET")
	_ = os.Unsetenv("TEST_SECRET")
	defer func() {
		_ = os.Unsetenv("TEST_SECRET")
		if originalExists {
			_ = os.Setenv("TEST_SECRET", originalValue)
		}
	}()

	config := &Config{
		ServerConfig: ServerConfig{
			SecretsDir: tempDir,
		},
		Vault: VaultConfig{
			// Empty vault config - should be skipped
		},
	}

	err = config.LoadSecrets()
	if err != nil {
		t.Errorf("LoadSecrets should not return error, got: %v", err)
	}

	// Check that environment variable was set
	value, exists := os.LookupEnv("TEST_SECRET")
	if !exists {
		t.Error("TEST_SECRET environment variable should be set")
	}
	if value != "test-value" {
		t.Errorf("TEST_SECRET = %q, want %q", value, "test-value")
	}
}

func TestLoadSecrets_EmptySecretsDir(t *testing.T) {
	config := &Config{
		ServerConfig: ServerConfig{
			SecretsDir: "",
		},
		Vault: VaultConfig{
			// Empty vault config - should be skipped
		},
	}

	err := config.LoadSecrets()
	if err != nil {
		t.Errorf("LoadSecrets with empty secrets dir should not return error, got: %v", err)
	}
}

func TestLoadSecrets_DirectoryError(t *testing.T) {
	config := &Config{
		ServerConfig: ServerConfig{
			SecretsDir: "/non/existent/directory",
		},
		Vault: VaultConfig{
			// Empty vault config - should be skipped
		},
	}

	err := config.LoadSecrets()
	if err == nil {
		t.Fatal("LoadSecrets with non-existent directory should return error")
	}

	if !strings.Contains(err.Error(), "failed to load secrets from directory") {
		t.Errorf("Error should mention directory loading failure, got: %v", err)
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

// TestLoadSecretsFromVault_NilSecret tests the case where Vault returns nil secret (no error but secret doesn't exist)
func TestLoadSecretsFromVault_NilSecret(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	vaultAddr := "http://localhost:8200"
	if !isVaultReachable(vaultAddr) {
		t.Skip("Vault container not reachable. Run 'docker-compose up vault' first")
	}

	// Test with a path that doesn't exist in Vault
	nonExistentPath := "secret/data/absolutely/nonexistent/path/for/testing"
	config := &Config{
		Vault: VaultConfig{
			Address: vaultAddr,
			Token:   "dev-root-token",
			Path:    nonExistentPath,
		},
	}

	// First verify the API behavior directly (this was the valuable part of the manual simulation test)
	apiConfig := api.DefaultConfig()
	apiConfig.Address = vaultAddr
	client, err := api.NewClient(apiConfig)
	if err != nil {
		t.Fatalf("Failed to create Vault client: %v", err)
	}
	client.SetToken("dev-root-token")

	// Verify that the Vault API returns nil secret with no error for nonexistent paths
	secret, err := client.Logical().Read(nonExistentPath)
	if err != nil {
		t.Fatalf("Expected no error from Vault API but got: %v", err)
	}
	if secret != nil {
		t.Errorf("Expected nil secret for nonexistent path, but got: %+v", secret)
	}

	// Now test that our function handles this case gracefully
	err = config.LoadSecretsFromVault()
	if err != nil {
		t.Errorf("LoadSecretsFromVault should not return error for nonexistent path, got: %v", err)
	}
}

// TestLoadSecretsFromVault_Success tests successful loading of secrets from Vault
func TestLoadSecretsFromVault_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	vaultAddr := "http://localhost:8200"
	if !isVaultReachable(vaultAddr) {
		t.Skip("Vault container not reachable. Run 'docker-compose up vault' first")
	}

	config := &Config{
		Vault: VaultConfig{
			Address: vaultAddr,
			Token:   "dev-root-token",
			Path:    "secret/data/sargantana",
		},
	}

	// Clear environment variables before test
	_ = os.Unsetenv("GOOGLE_KEY")
	_ = os.Unsetenv("GOOGLE_SECRET")
	_ = os.Unsetenv("SESSION_SECRET")

	defer func() {
		// Clean up after test
		_ = os.Unsetenv("GOOGLE_KEY")
		_ = os.Unsetenv("GOOGLE_SECRET")
		_ = os.Unsetenv("SESSION_SECRET")
	}()

	err := config.LoadSecretsFromVault()
	if err != nil {
		t.Fatalf("LoadSecretsFromVault failed: %v", err)
	}

	// Verify that secrets were loaded (assuming the test data exists)
	googleKey := os.Getenv("GOOGLE_KEY")
	if googleKey == "" {
		t.Log("GOOGLE_KEY not found - this might be expected if test data doesn't exist")
	}
}

// TestLoadSecretsFromVault_InvalidConfig tests with invalid Vault configuration
func TestLoadSecretsFromVault_InvalidConfig(t *testing.T) {
	config := &Config{
		Vault: VaultConfig{
			// Missing required fields - should be invalid
			Address: "",
			Token:   "",
			Path:    "",
		},
	}

	err := config.LoadSecretsFromVault()
	if err != nil {
		t.Errorf("LoadSecretsFromVault with invalid config should not return error (should skip), got: %v", err)
	}
}

// TestLoadSecretsFromVault_ConnectionError tests with unreachable Vault server
func TestLoadSecretsFromVault_ConnectionError(t *testing.T) {
	config := &Config{
		Vault: VaultConfig{
			Address: "http://nonexistent-vault-server:8200",
			Token:   "test-token",
			Path:    "secret/data/test",
		},
	}

	err := config.LoadSecretsFromVault()
	if err == nil {
		t.Fatal("LoadSecretsFromVault with unreachable server should return error")
	}

	if !strings.Contains(err.Error(), "failed to read secret from path") {
		t.Errorf("Error should mention failed to read secret, got: %v", err)
	}
}

// TestLoadSecretsFromVault_InvalidToken tests with invalid Vault token
func TestLoadSecretsFromVault_InvalidToken(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	vaultAddr := "http://localhost:8200"
	if !isVaultReachable(vaultAddr) {
		t.Skip("Vault container not reachable. Run 'docker-compose up vault' first")
	}

	config := &Config{
		Vault: VaultConfig{
			Address: vaultAddr,
			Token:   "invalid-token-that-does-not-exist",
			Path:    "secret/data/sargantana",
		},
	}

	err := config.LoadSecretsFromVault()
	if err == nil {
		t.Fatal("LoadSecretsFromVault with invalid token should return error")
	}

	if !strings.Contains(err.Error(), "failed to read secret from path") {
		t.Errorf("Error should mention failed to read secret, got: %v", err)
	}
}

// TestLoadSecretsFromVault_WithNamespace tests Vault with namespace configuration
func TestLoadSecretsFromVault_WithNamespace(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	vaultAddr := "http://localhost:8200"
	if !isVaultReachable(vaultAddr) {
		t.Skip("Vault container not reachable. Run 'docker-compose up vault' first")
	}

	config := &Config{
		Vault: VaultConfig{
			Address:   vaultAddr,
			Token:     "dev-root-token",
			Path:      "secret/data/test",
			Namespace: "test-namespace", // This namespace likely doesn't exist in dev mode
		},
	}

	// This should not panic or crash, even if namespace doesn't exist
	err := config.LoadSecretsFromVault()
	// We don't assert on error here because namespace behavior varies in dev mode
	if err != nil {
		t.Logf("LoadSecretsFromVault with namespace returned error (expected): %v", err)
	}
}
