package secrets

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

	resolver := NewVaultSecretLoader(client, "secret/data/sargantana")

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
	client, err := createTestVaultClient("http://localhost:8200", "dev-root-token")
	if err != nil {
		t.Fatalf("Failed to create Vault client: %v", err)
	}

	// KV v1 uses a different path structure (no /data/ in the path)
	resolver := NewVaultSecretLoader(client, "secret-v1/sargantana")

	// Test retrieving GOOGLE_KEY from KV v1
	googleKey, err := resolver.Resolve("GOOGLE_KEY")
	if err != nil {
		t.Fatalf("Resolve failed for GOOGLE_KEY in KV v1: %v", err)
	}
	if googleKey != "test-google-key" {
		t.Errorf("Expected 'test-google-key', got '%s'", googleKey)
	}

	// Test retrieving SESSION_SECRET from KV v1
	sessionSecret, err := resolver.Resolve("SESSION_SECRET")
	if err != nil {
		t.Fatalf("Resolve failed for SESSION_SECRET in KV v1: %v", err)
	}
	if sessionSecret != "test-session-secret-that-is-long-enough" {
		t.Errorf("Expected 'test-session-secret-that-is-long-enough', got '%s'", sessionSecret)
	}
}

// TestVaultResolver_NonexistentPath tests reading from nonexistent Vault path
func TestVaultResolver_NonexistentPath(t *testing.T) {
	client, err := createTestVaultClient("http://localhost:8200", "dev-root-token")
	if err != nil {
		t.Fatalf("Failed to create Vault client: %v", err)
	}

	resolver := NewVaultSecretLoader(client, "secret/data/nonexistent")

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

	resolver := NewVaultSecretLoader(client, "secret/data/sargantana")

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

	resolver := NewVaultSecretLoader(client, "secret/data/sargantana")

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

	resolver := NewVaultSecretLoader(client, "secret/data/test")
	if resolver.Name() != "Vault" {
		t.Errorf("Expected name 'Vault', got '%s'", resolver.Name())
	}
}

// TestVaultConfig_Validate tests VaultConfig validation
func TestVaultConfig_Validate(t *testing.T) {
	tests := []struct {
		name          string
		config        *VaultConfig
		errorExpected bool
		errorContains string
	}{
		{
			name: "valid config",
			config: &VaultConfig{
				Address: "http://localhost:8200",
				Token:   "test-token",
				Path:    "secret/data/test",
			},
			errorExpected: false,
		},
		{
			name: "valid config with namespace",
			config: &VaultConfig{
				Address:   "http://localhost:8200",
				Token:     "test-token",
				Path:      "secret/data/test",
				Namespace: "test-namespace",
			},
			errorExpected: false,
		},
		{
			name: "missing address",
			config: &VaultConfig{
				Address: "",
				Token:   "test-token",
				Path:    "secret/data/test",
			},
			errorExpected: true,
			errorContains: "Vault address is required",
		},
		{
			name: "missing token",
			config: &VaultConfig{
				Address: "http://localhost:8200",
				Token:   "",
				Path:    "secret/data/test",
			},
			errorExpected: true,
			errorContains: "Vault token is required",
		},
		{
			name: "missing path",
			config: &VaultConfig{
				Address: "http://localhost:8200",
				Token:   "test-token",
				Path:    "",
			},
			errorExpected: true,
			errorContains: "Vault path is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.errorExpected {
				if err == nil {
					t.Error("Expected error but got nil")
				} else if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error containing %q, got: %v", tt.errorContains, err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

// TestCreateVaultClient_Success tests successful Vault client creation
func TestCreateVaultClient_Success(t *testing.T) {
	vaultCfg := &VaultConfig{
		Address: "http://localhost:8200",
		Token:   "dev-root-token",
		Path:    "secret/data/sargantana",
	}

	client, err := vaultCfg.CreateClient()
	if err != nil {
		t.Fatalf("CreateVaultClient failed: %v", err)
	}

	if client == nil {
		t.Fatal("Vault client should not be nil")
	}

	// Verify client is configured correctly
	if client.Address() != "http://localhost:8200" {
		t.Errorf("Expected address 'http://localhost:8200', got '%s'", client.Address())
	}
}

// TestCreateVaultClient_WithNamespace tests Vault client creation with namespace
func TestCreateVaultClient_WithNamespace(t *testing.T) {
	vaultCfg := &VaultConfig{
		Address:   "http://localhost:8200",
		Token:     "dev-root-token",
		Path:      "secret/data/sargantana",
		Namespace: "test-namespace",
	}

	client, err := vaultCfg.CreateClient()
	if err != nil {
		t.Fatalf("CreateVaultClient with namespace failed: %v", err)
	}

	if client == nil {
		t.Fatal("Vault client should not be nil")
	}
}

// TestCreateVaultClient_InvalidConfig tests with invalid Vault configuration
func TestCreateVaultClient_InvalidConfig(t *testing.T) {
	vaultCfg := &VaultConfig{
		Address: "",
		Token:   "",
		Path:    "",
	}

	_, err := vaultCfg.CreateClient()
	if err == nil {
		t.Error("Expected error with invalid Vault configuration, got nil")
	}
	if err != nil && !strings.Contains(err.Error(), "Vault address is required") {
		t.Errorf("Expected address validation error, got: %v", err)
	}
}

// TestCreateVaultClient_InvalidAddress tests with malformed Vault address
func TestCreateVaultClient_InvalidAddress(t *testing.T) {
	vaultCfg := &VaultConfig{
		Address: string([]byte{0, 1, 2, 3}), // Invalid URL with null bytes
		Token:   "test-token",
		Path:    "secret/data/test",
	}

	_, err := vaultCfg.CreateClient()
	if err == nil {
		t.Fatal("Expected error when creating Vault client with invalid address")
	}

	if !strings.Contains(err.Error(), "invalid control character in URL") {
		t.Errorf("Error should mention invalid control character, got: %v", err)
	}
}

// TestVaultConfig_CreateClient tests the ClientFactory pattern
func TestVaultConfig_CreateClient(t *testing.T) {
	vaultCfg := &VaultConfig{
		Address: "http://localhost:8200",
		Token:   "dev-root-token",
		Path:    "secret/data/sargantana",
	}

	// Test using ClientFactory interface
	client, err := vaultCfg.CreateClient()
	if err != nil {
		t.Fatalf("CreateClient failed: %v", err)
	}

	if client == nil {
		t.Fatal("Client should not be nil")
	}

	if client.Address() != "http://localhost:8200" {
		t.Errorf("Expected address 'http://localhost:8200', got '%s'", client.Address())
	}
}

// TestVaultPropertyResolution_Success tests property resolution using Vault resolver with Docker
func TestVaultPropertyResolution_Success(t *testing.T) {
	// Set up Vault resolver using docker compose Vault instance
	vaultCfg := &VaultConfig{
		Address: "http://localhost:8200",
		Token:   "dev-root-token",
		Path:    "secret/data/sargantana",
	}

	client, err := vaultCfg.CreateClient()
	if err != nil {
		t.Fatalf("Failed to create Vault client: %v", err)
	}

	vaultResolver := NewVaultSecretLoader(client, vaultCfg.Path)

	// Register the resolver
	Register("vault", vaultResolver)

	// Test resolving a property using vault: prefix
	result, err := Resolve("vault:GOOGLE_KEY")
	if err != nil {
		t.Fatalf("Failed to resolve vault:GOOGLE_KEY: %v", err)
	}

	expected := "test-google-key"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}

	// Test resolving another property
	sessionSecret, err := Resolve("vault:SESSION_SECRET")
	if err != nil {
		t.Fatalf("Failed to resolve vault:SESSION_SECRET: %v", err)
	}

	expectedSecret := "test-session-secret-that-is-long-enough"
	if sessionSecret != expectedSecret {
		t.Errorf("Expected '%s', got '%s'", expectedSecret, sessionSecret)
	}
}

// TestVaultPropertyResolution_NonexistentKey tests property resolution with nonexistent key
func TestVaultPropertyResolution_NonexistentKey(t *testing.T) {
	vaultCfg := &VaultConfig{
		Address: "http://localhost:8200",
		Token:   "dev-root-token",
		Path:    "secret/data/sargantana",
	}

	client, err := vaultCfg.CreateClient()
	if err != nil {
		t.Fatalf("Failed to create Vault client: %v", err)
	}

	vaultResolver := NewVaultSecretLoader(client, vaultCfg.Path)
	Register("vault", vaultResolver)

	_, err = Resolve("vault:NONEXISTENT_KEY")
	if err == nil {
		t.Fatal("Expected error when resolving nonexistent Vault key")
	}

	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Expected 'not found' error, got: %v", err)
	}
}

// TestVaultPropertyResolution_InvalidToken tests property resolution with invalid token
func TestVaultPropertyResolution_InvalidToken(t *testing.T) {
	vaultCfg := &VaultConfig{
		Address: "http://localhost:8200",
		Token:   "invalid-token",
		Path:    "secret/data/sargantana",
	}

	client, err := vaultCfg.CreateClient()
	if err != nil {
		t.Fatalf("Failed to create Vault client: %v", err)
	}

	vaultResolver := NewVaultSecretLoader(client, vaultCfg.Path)
	Register("vault", vaultResolver)

	_, err = Resolve("vault:GOOGLE_KEY")
	if err == nil {
		t.Fatal("Expected error when using invalid token")
	}

	if !strings.Contains(err.Error(), "failed to read secret") && !strings.Contains(err.Error(), "permission denied") {
		t.Errorf("Expected authentication error, got: %v", err)
	}
}

// TestVaultPropertyResolution_KVv1 tests property resolution using KV v1 secret engine
func TestVaultPropertyResolution_KVv1(t *testing.T) {
	// Set up Vault resolver for KV v1
	vaultCfg := &VaultConfig{
		Address: "http://localhost:8200",
		Token:   "dev-root-token",
		Path:    "secret-v1/sargantana",
	}

	client, err := vaultCfg.CreateClient()
	if err != nil {
		t.Fatalf("Failed to create Vault client: %v", err)
	}

	vaultResolver := NewVaultSecretLoader(client, vaultCfg.Path)
	Register("vault", vaultResolver)

	// Test resolving from KV v1
	result, err := Resolve("vault:GOOGLE_KEY")
	if err != nil {
		t.Fatalf("Failed to resolve vault:GOOGLE_KEY from KV v1: %v", err)
	}

	expected := "test-google-key"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}

	// Test another property from KV v1
	sessionSecret, err := Resolve("vault:SESSION_SECRET")
	if err != nil {
		t.Fatalf("Failed to resolve vault:SESSION_SECRET from KV v1: %v", err)
	}

	expectedSecret := "test-session-secret-that-is-long-enough"
	if sessionSecret != expectedSecret {
		t.Errorf("Expected '%s', got '%s'", expectedSecret, sessionSecret)
	}
}
