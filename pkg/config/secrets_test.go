package config

import (
	"net/http"
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

// TestCreateVaultManager_Success tests successful creation of Vault manager with Docker container
func TestCreateVaultManager_Success(t *testing.T) {

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

	// Verify the vault resolver is registered
	res := resolver.Global.GetResolver("vault")
	if res == nil {
		t.Fatal("Vault resolver should be registered")
	}
}

// TestCreateVaultManager_WithNamespace tests Vault manager creation with namespace
func TestCreateVaultManager_WithNamespace(t *testing.T) {
	vaultCfg := &VaultConfig{
		Address:   "http://localhost:8200",
		Token:     "dev-root-token",
		Path:      "secret/data/sargantana",
		Namespace: "test-namespace",
	}

	// This should succeed even though the namespace doesn't exist in dev mode
	_, err := CreateVaultClient(vaultCfg)
	if err != nil {
		t.Fatalf("CreateVaultClient with namespace failed: %v", err)
	}
}

// TestCreateVaultManager_InvalidConfig tests with invalid Vault configuration
func TestCreateVaultManager_InvalidConfig(t *testing.T) {
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

// TestCreateVaultManager_ConnectionError tests with unreachable Vault server
func TestCreateVaultManager_ConnectionError(t *testing.T) {
	vaultCfg := &VaultConfig{
		Address: "http://nonexistent-vault-server:8200",
		Token:   "test-token",
		Path:    "secret/data/test",
	}

	_, err := CreateVaultClient(vaultCfg)
	// This might not fail at creation time, but will fail when trying to read secrets
	// The actual connection is tested during secret retrieval
	if err != nil && !strings.Contains(err.Error(), "failed to create Vault client") {
		t.Errorf("Unexpected error type: %v", err)
	}
}

// TestCreateVaultManager_InvalidAddress tests with malformed Vault address
func TestCreateVaultManager_InvalidAddress(t *testing.T) {
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

// TestVaultManager_Secret_Success tests successful secret retrieval from Docker Vault
func TestVaultManager_Secret_Success(t *testing.T) {
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

	// Test retrieving the pre-configured secrets using the resolver
	res := resolver.Global.GetResolver("vault")
	if res == nil {
		t.Fatal("Vault resolver should be registered")
	}

	googleKey, err := res.Resolve("GOOGLE_KEY")
	if err != nil {
		t.Fatalf("Failed to retrieve GOOGLE_KEY: %v", err)
	}
	if googleKey != "test-google-key" {
		t.Errorf("Expected GOOGLE_KEY 'test-google-key', got %v", googleKey)
	}

	sessionSecret, err := res.Resolve("SESSION_SECRET")
	if err != nil {
		t.Fatalf("Failed to retrieve SESSION_SECRET: %v", err)
	}
	if sessionSecret != "test-session-secret-that-is-long-enough" {
		t.Errorf("Expected SESSION_SECRET 'test-session-secret-that-is-long-enough', got %v", sessionSecret)
	}
}

// TestVaultManager_Secret_KVv1 tests secret retrieval from KV v1 engine
func TestVaultManager_Secret_KVv1(t *testing.T) {
	config := &Config{
		Vault: &VaultConfig{
			Address: "http://localhost:8200",
			Token:   "dev-root-token",
			Path:    "secret-v1/sargantana", // KV v1 path
		},
	}

	err := setupResolversForTest(config)
	if err != nil {
		t.Fatalf("setupResolversForTest failed: %v", err)
	}

	// Test retrieving secrets from KV v1 engine using resolver
	res := resolver.Global.GetResolver("vault")
	if res == nil {
		t.Fatal("Vault resolver should be registered")
	}

	googleKey, err := res.Resolve("GOOGLE_KEY")
	if err != nil {
		t.Fatalf("Failed to retrieve GOOGLE_KEY from KV v1: %v", err)
	}
	if googleKey != "test-google-key" {
		t.Errorf("Expected GOOGLE_KEY 'test-google-key', got %v", googleKey)
	}
}

// TestVaultManager_Secret_NonexistentPath tests with nonexistent Vault path
func TestVaultManager_Secret_NonexistentPath(t *testing.T) {
	config := &Config{
		Vault: &VaultConfig{
			Address: "http://localhost:8200",
			Token:   "dev-root-token",
			Path:    "secret/data/nonexistent/path",
		},
	}

	err := setupResolversForTest(config)
	if err != nil {
		t.Fatalf("setupResolversForTest failed: %v", err)
	}

	// Test with nonexistent path using resolver
	res := resolver.Global.GetResolver("vault")
	if res == nil {
		t.Fatal("Vault resolver should be registered")
	}

	_, err = res.Resolve("SOME_KEY")
	if err == nil {
		t.Fatal("Expected error when reading from nonexistent path")
	}
	if !strings.Contains(err.Error(), "no secret found") {
		t.Errorf("Expected 'no secret found' error, got: %v", err)
	}
}

// TestVaultManager_Secret_NonexistentKey tests with nonexistent key in existing path
func TestVaultManager_Secret_NonexistentKey(t *testing.T) {
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

	// Test with nonexistent key
	res := resolver.Global.GetResolver("vault")
	if res == nil {
		t.Fatal("Vault resolver should be registered")
	}

	_, err = res.Resolve("NONEXISTENT_KEY")
	if err == nil {
		t.Fatal("Expected error when reading nonexistent key")
	}
	if !strings.Contains(err.Error(), "secret \"NONEXISTENT_KEY\" not found") {
		t.Errorf("Expected 'secret not found' error, got: %v", err)
	}
}

// TestVaultManager_Secret_InvalidToken tests with invalid Vault token
func TestVaultManager_Secret_InvalidToken(t *testing.T) {
	config := &Config{
		Vault: &VaultConfig{
			Address: "http://localhost:8200",
			Token:   "invalid-token",
			Path:    "secret/data/sargantana",
		},
	}

	err := setupResolversForTest(config)
	if err != nil {
		t.Fatalf("setupResolversForTest failed: %v", err)
	}

	// Test with invalid token
	res := resolver.Global.GetResolver("vault")
	if res == nil {
		t.Fatal("Vault resolver should be registered")
	}

	_, err = res.Resolve("GOOGLE_KEY")
	if err == nil {
		t.Fatal("Expected error when using invalid token")
	}
	if !strings.Contains(err.Error(), "failed to read secret from Vault path") {
		t.Errorf("Expected 'failed to read secret' error, got: %v", err)
	}
}

// TestSecretFromFile_Success tests successful file secret reading
func TestSecretFromFile_Success(t *testing.T) {
	tempDir := t.TempDir()

	// Create a test secret file
	secretFile := filepath.Join(tempDir, "test-secret")
	secretContent := "my-secret-value\n"
	err := os.WriteFile(secretFile, []byte(secretContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test secret file: %v", err)
	}

	// Create file resolver and test
	fileResolver := resolver.NewFileResolver(tempDir)
	result, err := fileResolver.Resolve("test-secret")
	if err != nil {
		t.Fatalf("FileResolver.Resolve failed: %v", err)
	}

	expected := "my-secret-value"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

// TestSecretFromFile_NoSecretsDir tests file secret reading without configured secrets directory
func TestSecretFromFile_NoSecretsDir(t *testing.T) {
	// Create file resolver with empty directory
	fileResolver := resolver.NewFileResolver("")

	_, err := fileResolver.Resolve("test-secret")
	if err == nil {
		t.Fatal("Expected error when no secrets directory is configured")
	}
	if !strings.Contains(err.Error(), "no secrets directory configured") {
		t.Errorf("Expected 'no secrets directory configured' error, got: %v", err)
	}
}

// TestSecretFromFile_EmptyFilename tests file secret reading with empty filename
func TestSecretFromFile_EmptyFilename(t *testing.T) {
	tempDir := t.TempDir()

	// Create file resolver and test with empty filename
	fileResolver := resolver.NewFileResolver(tempDir)
	_, err := fileResolver.Resolve("")
	if err == nil {
		t.Fatal("Expected error when filename is empty")
	}
	if !strings.Contains(err.Error(), "no file specified for file secret") {
		t.Errorf("Expected 'no file specified' error, got: %v", err)
	}
}

// TestSecretFromFile_NonexistentFile tests file secret reading with nonexistent file
func TestSecretFromFile_NonexistentFile(t *testing.T) {
	tempDir := t.TempDir()

	// Create file resolver and test with nonexistent file
	fileResolver := resolver.NewFileResolver(tempDir)
	_, err := fileResolver.Resolve("nonexistent-file")
	if err == nil {
		t.Fatal("Expected error when file doesn't exist")
	}
	if !strings.Contains(err.Error(), "error reading secret file") {
		t.Errorf("Expected 'error reading secret file' error, got: %v", err)
	}
}

// TestExpand_VaultPrefix tests the expand function with vault: prefix
func TestExpand_VaultPrefix(t *testing.T) {
	// Set up Vault manager
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

// TestExpand_FilePrefix tests the expand function with file: prefix
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

// TestExpand_EnvPrefix tests the expand function with env: prefix
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
