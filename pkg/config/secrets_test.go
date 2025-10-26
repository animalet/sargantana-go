package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/animalet/sargantana-go/pkg/resolver"
)

// setupResolversForTest registers resolvers for testing based on the provided config
func setupResolversForTest(cfg *Config) error {
	// Always register env resolver
	resolver.Register("env", resolver.NewEnvResolver())

	// Register file resolver if secrets directory is configured
	if cfg.ServerConfig.SecretsDir != "" {
		resolver.Register("file", resolver.NewFileResolver(cfg.ServerConfig.SecretsDir))
	}

	// Register Vault resolver if Vault is configured
	if cfg.Vault != nil {
		client, err := CreateVaultClient(cfg.Vault)
		if err != nil {
			return err
		}
		resolver.Register("vault", resolver.NewVaultResolver(client, cfg.Vault.Path))
	}

	return nil
}

// TestCreateVaultClient_Success tests successful Vault client creation
func TestCreateVaultClient_Success(t *testing.T) {
	vaultCfg := &VaultConfig{
		Address: "http://localhost:8200",
		Token:   "dev-root-token",
		Path:    "secret/data/sargantana",
	}

	client, err := CreateVaultClient(vaultCfg)
	if err != nil {
		t.Fatalf("CreateVaultClient failed: %v", err)
	}

	if client == nil {
		t.Fatal("Vault client should not be nil")
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

	_, err := CreateVaultClient(vaultCfg)
	if err != nil {
		t.Fatalf("CreateVaultClient with namespace failed: %v", err)
	}
}

// TestCreateVaultClient_InvalidConfig tests with invalid Vault configuration
func TestCreateVaultClient_InvalidConfig(t *testing.T) {
	vaultCfg := &VaultConfig{
		Address: "",
		Token:   "",
		Path:    "",
	}

	_, err := CreateVaultClient(vaultCfg)
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

	_, err := CreateVaultClient(vaultCfg)
	if err == nil {
		t.Fatal("Expected error when creating Vault client with invalid address")
	}

	if !strings.Contains(err.Error(), "invalid control character in URL") {
		t.Errorf("Error should mention invalid control character, got: %v", err)
	}
}

// TestExpand_VaultPrefix tests the expand function with vault: prefix integration
func TestExpand_VaultPrefix(t *testing.T) {
	config := &Config{
		Vault: &VaultConfig{
			Address: "http://localhost:8200",
			Token:   "dev-root-token",
			Path:    "secret/data/sargantana",
		},
	}

	err := setupResolversForTest(config)
	if err != nil {
		t.Fatalf("setupResolversForTest failed: %v", err)
	}

	result := expand("vault:GOOGLE_KEY")
	expected := "test-google-key"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

// TestExpand_VaultPrefix_NonexistentKey tests expand with nonexistent Vault key
func TestExpand_VaultPrefix_NonexistentKey(t *testing.T) {
	config := &Config{
		Vault: &VaultConfig{
			Address: "http://localhost:8200",
			Token:   "dev-root-token",
			Path:    "secret/data/sargantana",
		},
	}

	err := setupResolversForTest(config)
	if err != nil {
		t.Fatalf("setupResolversForTest failed: %v", err)
	}

	defer func() {
		if r := recover(); r != nil {
			if !strings.Contains(r.(error).Error(), "error resolving property") {
				t.Errorf("Expected 'error resolving property' panic, got: %v", r)
			}
		} else {
			t.Fatal("Expected panic when expanding nonexistent Vault key")
		}
	}()

	expand("vault:NONEXISTENT_KEY")
}

// TestExpand_FilePrefix tests the expand function with file: prefix integration
func TestExpand_FilePrefix(t *testing.T) {
	tempDir := t.TempDir()

	// Create a test secret file
	secretFile := filepath.Join(tempDir, "test-secret")
	secretContent := "file-secret-value"
	err := os.WriteFile(secretFile, []byte(secretContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test secret file: %v", err)
	}

	// Register file resolver for this test
	resolver.Register("file", resolver.NewFileResolver(tempDir))
	defer resolver.Unregister("file")

	result := expand("file:test-secret")
	if result != secretContent {
		t.Errorf("Expected '%s', got '%s'", secretContent, result)
	}
}

// TestExpand_EnvPrefix tests the expand function with env: prefix integration
func TestExpand_EnvPrefix(t *testing.T) {
	// Register env resolver
	resolver.Register("env", resolver.NewEnvResolver())
	defer resolver.Unregister("env")

	_ = os.Setenv("TEST_EXPAND_VAR", "env-value")
	defer func() { _ = os.Unsetenv("TEST_EXPAND_VAR") }()

	result := expand("env:TEST_EXPAND_VAR")
	expected := "env-value"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

// TestExpand_PlainEnvVar tests the expand function with plain environment variable name
func TestExpand_PlainEnvVar(t *testing.T) {
	// Register env resolver (used as default when no prefix)
	resolver.Register("env", resolver.NewEnvResolver())
	defer resolver.Unregister("env")

	_ = os.Setenv("TEST_PLAIN_VAR", "plain-value")
	defer func() { _ = os.Unsetenv("TEST_PLAIN_VAR") }()

	result := expand("TEST_PLAIN_VAR")
	expected := "plain-value"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

// TestExpand_UnknownPrefix tests the expand function with unknown prefix
func TestExpand_UnknownPrefix(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			if !strings.Contains(r.(error).Error(), "no resolver registered") {
				t.Errorf("Expected 'no resolver registered' panic, got: %v", r)
			}
		} else {
			t.Fatal("Expected panic for unknown prefix")
		}
	}()

	expand("unknown:value")
}
