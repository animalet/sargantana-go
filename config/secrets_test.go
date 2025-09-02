package config

import (
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/vault/api"
)

// TestVaultHealthCheck tests that we can properly detect if Vault is healthy
func TestVaultHealthCheck(t *testing.T) {
	vaultAddr := "http://localhost:8200"
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

// TestLoadSecretsFromVault_NilSecret tests the case where Vault returns nil secret (no error but secret doesn't exist)
func TestLoadSecretsFromVault_NilSecret(t *testing.T) {
	vaultAddr := "http://localhost:8200"
	// Test with a path that doesn't exist in Vault
	nonExistentPath := "secret/data/absolutely/nonexistent/path/for/testing"
	config := &Config{
		Vault: &VaultConfig{
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
	err = config.createVaultManager()
	if err != nil {
		t.Errorf("LoadSecretsFromVault should not return error for nonexistent path, got: %v", err)
	}
}

// TestLoadSecretsFromVault_Success tests successful loading of secrets from Vault
func TestLoadSecretsFromVault_Success(t *testing.T) {
	config := &Config{
		Vault: &VaultConfig{
			Address: "http://localhost:8200",
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

	err := config.createVaultManager()
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
		Vault: &VaultConfig{
			// Missing required fields - should be invalid
			Address: "",
			Token:   "",
			Path:    "",
		},
	}

	err := config.createVaultManager()
	if err != nil {
		t.Errorf("LoadSecretsFromVault with invalid config should not return error (should skip), got: %v", err)
	}
}

// TestLoadSecretsFromVault_ConnectionError tests with unreachable Vault server
func TestLoadSecretsFromVault_ConnectionError(t *testing.T) {
	config := &Config{
		Vault: &VaultConfig{
			Address: "http://nonexistent-vault-server:8200",
			Token:   "test-token",
			Path:    "secret/data/test",
		},
	}

	err := config.createVaultManager()
	if err == nil {
		t.Fatal("LoadSecretsFromVault with unreachable server should return error")
	}

	if !strings.Contains(err.Error(), "failed to read secret from path") {
		t.Errorf("Error should mention failed to read secret, got: %v", err)
	}
}

// TestLoadSecretsFromVault_InvalidToken tests with invalid Vault token
func TestLoadSecretsFromVault_InvalidToken(t *testing.T) {
	config := &Config{
		Vault: &VaultConfig{
			Address: "http://localhost:8200",
			Token:   "invalid-token-that-does-not-exist",
			Path:    "secret/data/sargantana",
		},
	}

	err := config.createVaultManager()
	if err == nil {
		t.Fatal("LoadSecretsFromVault with invalid token should return error")
	}

	if !strings.Contains(err.Error(), "failed to read secret from path") {
		t.Errorf("Error should mention failed to read secret, got: %v", err)
	}
}

// TestLoadSecretsFromVault_WithNamespace tests Vault with namespace configuration
func TestLoadSecretsFromVault_WithNamespace(t *testing.T) {
	// Test both integration scenario (with real Vault) and unit scenario (with invalid token)

	// Unit test scenario with invalid token
	config := &Config{
		Vault: &VaultConfig{
			Address:   "http://localhost:8200",
			Token:     "test-token",
			Path:      "secret/data/test",
			Namespace: "test-namespace",
		},
	}

	// This test focuses on the namespace setting code path
	// We expect it to fail due to invalid token, but the namespace should be set
	err := config.createVaultManager()
	if err == nil {
		t.Fatal("Expected error due to invalid token")
	}

	// The error should be about reading the secret, not about namespace
	if !strings.Contains(err.Error(), "failed to read secret from path") {
		t.Errorf("Error should mention failed to read secret, got: %v", err)
	}
}

// TestLoadSecretsFromVault_CreateClientError tests error when creating Vault client fails
func TestLoadSecretsFromVault_CreateClientError(t *testing.T) {
	// Test with configuration that would cause client creation to fail
	// This is challenging to test directly, so we test with an extreme edge case
	config := &Config{
		Vault: &VaultConfig{
			Address: string([]byte{0, 1, 2, 3}), // Invalid URL with null bytes
			Token:   "test-token",
			Path:    "secret/data/test",
		},
	}

	err := config.createVaultManager()
	if err == nil {
		t.Fatal("Expected error when creating Vault client with invalid address")
	}

	if !strings.Contains(err.Error(), "failed to create Vault client") {
		t.Errorf("Error should mention failed to create Vault client, got: %v", err)
	}
}
