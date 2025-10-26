package resolver

import (
	"net/http"
	"strings"
	"testing"

	"github.com/hashicorp/vault/api"
)

// TestVaultHealthCheck verifies that the Docker Vault container is healthy
func TestVaultHealthCheck(t *testing.T) {
	vaultAddr := "http://localhost:8200"
	resp, err := http.Get(vaultAddr + "/v1/sys/health")
	if err != nil {
		t.Fatalf("Failed to check Vault health: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		t.Errorf("Expected Vault to be healthy (status 200), got status %d", resp.StatusCode)
	}
}

// createTestVaultClient creates a Vault client for testing
func createTestVaultClient(address, token string) (*api.Client, error) {
	config := api.DefaultConfig()
	config.Address = address
	client, err := api.NewClient(config)
	if err != nil {
		return nil, err
	}
	client.SetToken(token)
	return client, nil
}

// TestVaultResolver_Success tests successful secret retrieval from Docker Vault
func TestVaultResolver_Success(t *testing.T) {
	client, err := createTestVaultClient("http://localhost:8200", "dev-root-token")
	if err != nil {
		t.Fatalf("Failed to create Vault client: %v", err)
	}

	resolver := NewVaultResolver(client, "secret/data/sargantana")

	// Test retrieving GOOGLE_KEY
	googleKey, err := resolver.Resolve("GOOGLE_KEY")
	if err != nil {
		t.Fatalf("Resolve failed for GOOGLE_KEY: %v", err)
	}
	if googleKey != "test-google-key" {
		t.Errorf("Expected 'test-google-key', got '%s'", googleKey)
	}

	// Test retrieving SESSION_SECRET
	sessionSecret, err := resolver.Resolve("SESSION_SECRET")
	if err != nil {
		t.Fatalf("Resolve failed for SESSION_SECRET: %v", err)
	}
	if sessionSecret != "test-session-secret-that-is-long-enough" {
		t.Errorf("Expected 'test-session-secret-that-is-long-enough', got '%s'", sessionSecret)
	}
}

// TestVaultResolver_KVv1 tests Vault KV v1 secret engine
func TestVaultResolver_KVv1(t *testing.T) {
	t.Skip("Skipping KV v1 test - requires separate Vault KV v1 setup")

	client, err := createTestVaultClient("http://localhost:8200", "dev-root-token")
	if err != nil {
		t.Fatalf("Failed to create Vault client: %v", err)
	}

	// KV v1 uses a different path structure
	resolver := NewVaultResolver(client, "secret/sargantana_v1")

	googleKey, err := resolver.Resolve("GOOGLE_KEY")
	if err != nil {
		t.Fatalf("Resolve failed for GOOGLE_KEY in KV v1: %v", err)
	}
	if googleKey != "test-google-key-v1" {
		t.Errorf("Expected 'test-google-key-v1', got '%s'", googleKey)
	}
}

// TestVaultResolver_NonexistentPath tests reading from nonexistent Vault path
func TestVaultResolver_NonexistentPath(t *testing.T) {
	client, err := createTestVaultClient("http://localhost:8200", "dev-root-token")
	if err != nil {
		t.Fatalf("Failed to create Vault client: %v", err)
	}

	resolver := NewVaultResolver(client, "secret/data/nonexistent")

	_, err = resolver.Resolve("SOME_KEY")
	if err == nil {
		t.Fatal("Expected error when reading from nonexistent path")
	}
	if !strings.Contains(err.Error(), "no secret found") {
		t.Errorf("Expected 'no secret found' error, got: %v", err)
	}
}

// TestVaultResolver_NonexistentKey tests reading nonexistent key from Vault
func TestVaultResolver_NonexistentKey(t *testing.T) {
	client, err := createTestVaultClient("http://localhost:8200", "dev-root-token")
	if err != nil {
		t.Fatalf("Failed to create Vault client: %v", err)
	}

	resolver := NewVaultResolver(client, "secret/data/sargantana")

	_, err = resolver.Resolve("NONEXISTENT_KEY")
	if err == nil {
		t.Fatal("Expected error when reading nonexistent key")
	}
	if !strings.Contains(err.Error(), "secret \"NONEXISTENT_KEY\" not found") {
		t.Errorf("Expected 'secret not found' error, got: %v", err)
	}
}

// TestVaultResolver_InvalidToken tests Vault resolver with invalid token
func TestVaultResolver_InvalidToken(t *testing.T) {
	client, err := createTestVaultClient("http://localhost:8200", "invalid-token")
	if err != nil {
		t.Fatalf("Failed to create Vault client: %v", err)
	}

	resolver := NewVaultResolver(client, "secret/data/sargantana")

	_, err = resolver.Resolve("GOOGLE_KEY")
	if err == nil {
		t.Fatal("Expected error when using invalid token")
	}
	if !strings.Contains(err.Error(), "failed to read secret from Vault path") {
		t.Errorf("Expected 'failed to read secret' error, got: %v", err)
	}
}

// TestVaultResolver_Name tests the Name method
func TestVaultResolver_Name(t *testing.T) {
	client, err := createTestVaultClient("http://localhost:8200", "dev-root-token")
	if err != nil {
		t.Fatalf("Failed to create Vault client: %v", err)
	}

	resolver := NewVaultResolver(client, "secret/data/test")
	if resolver.Name() != "Vault" {
		t.Errorf("Expected name 'Vault', got '%s'", resolver.Name())
	}
}
