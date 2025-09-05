package config

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
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

// TestCreateVaultManager_Success tests successful creation of Vault manager with Docker container
func TestCreateVaultManager_Success(t *testing.T) {
	config := &Config{
		Vault: &VaultConfig{
			Address: "http://localhost:8200",
			Token:   "dev-root-token",
			Path:    "secret/data/sargantana",
		},
	}

	err := config.createSecretSourcesIfNotPresent()
	if err != nil {
		t.Fatalf("createVaultManager failed: %v", err)
	}

	if vaultManagerInstance == nil {
		t.Fatal("vaultManagerInstance should be initialized")
	}
	if vaultManagerInstance.path != "secret/data/sargantana" {
		t.Errorf("Expected path 'secret/data/sargantana', got '%s'", vaultManagerInstance.path)
	}
}

// TestCreateVaultManager_WithNamespace tests Vault manager creation with namespace
func TestCreateVaultManager_WithNamespace(t *testing.T) {
	config := &Config{
		Vault: &VaultConfig{
			Address:   "http://localhost:8200",
			Token:     "dev-root-token",
			Path:      "secret/data/sargantana",
			Namespace: "test-namespace",
		},
	}

	// This should succeed even though the namespace doesn't exist in dev mode
	err := config.createSecretSourcesIfNotPresent()
	if err != nil {
		t.Fatalf("createVaultManager with namespace failed: %v", err)
	}
}

// TestCreateVaultManager_InvalidConfig tests with invalid Vault configuration
func TestCreateVaultManager_InvalidConfig(t *testing.T) {
	config := &Config{
		Vault: &VaultConfig{
			Address: "",
			Token:   "",
			Path:    "",
		},
	}

	err := config.createSecretSourcesIfNotPresent()
	if err != nil {
		t.Errorf("createVaultManager with invalid config should not return error (should skip), got: %v", err)
	}
}

// TestCreateVaultManager_ConnectionError tests with unreachable Vault server
func TestCreateVaultManager_ConnectionError(t *testing.T) {
	config := &Config{
		Vault: &VaultConfig{
			Address: "http://nonexistent-vault-server:8200",
			Token:   "test-token",
			Path:    "secret/data/test",
		},
	}

	err := config.createSecretSourcesIfNotPresent()
	// This might not fail at creation time, but will fail when trying to read secrets
	// The actual connection is tested during secret retrieval
	if err != nil && !strings.Contains(err.Error(), "failed to create Vault client") {
		t.Errorf("Unexpected error type: %v", err)
	}
}

// TestCreateVaultManager_InvalidAddress tests with malformed Vault address
func TestCreateVaultManager_InvalidAddress(t *testing.T) {
	config := &Config{
		Vault: &VaultConfig{
			Address: string([]byte{0, 1, 2, 3}), // Invalid URL with null bytes
			Token:   "test-token",
			Path:    "secret/data/test",
		},
	}

	err := config.createSecretSourcesIfNotPresent()
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

	err := config.createSecretSourcesIfNotPresent()
	if err != nil {
		t.Fatalf("createVaultManager failed: %v", err)
	}

	// Test retrieving the pre-configured secrets
	googleKey, err := vaultManagerInstance.secret("GOOGLE_KEY")
	if err != nil {
		t.Fatalf("Failed to retrieve GOOGLE_KEY: %v", err)
	}
	if googleKey == nil || *googleKey != "test-google-key" {
		t.Errorf("Expected GOOGLE_KEY 'test-google-key', got %v", googleKey)
	}

	sessionSecret, err := vaultManagerInstance.secret("SESSION_SECRET")
	if err != nil {
		t.Fatalf("Failed to retrieve SESSION_SECRET: %v", err)
	}
	if sessionSecret == nil || *sessionSecret != "test-session-secret-that-is-long-enough" {
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

	err := config.createSecretSourcesIfNotPresent()
	if err != nil {
		t.Fatalf("createVaultManager failed: %v", err)
	}

	// Test retrieving secrets from KV v1 engine
	googleKey, err := vaultManagerInstance.secret("GOOGLE_KEY")
	if err != nil {
		t.Fatalf("Failed to retrieve GOOGLE_KEY from KV v1: %v", err)
	}
	if googleKey == nil || *googleKey != "test-google-key" {
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

	err := config.createSecretSourcesIfNotPresent()
	if err != nil {
		t.Fatalf("createVaultManager failed: %v", err)
	}

	// Test with nonexistent path
	_, err = vaultManagerInstance.secret("SOME_KEY")
	if err == nil {
		t.Fatal("Expected error when reading from nonexistent path")
	}
	if !strings.Contains(err.Error(), "no secret found at the specified path") {
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

	err := config.createSecretSourcesIfNotPresent()
	if err != nil {
		t.Fatalf("createVaultManager failed: %v", err)
	}

	// Test with nonexistent key
	_, err = vaultManagerInstance.secret("NONEXISTENT_KEY")
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

	err := config.createSecretSourcesIfNotPresent()
	if err != nil {
		t.Fatalf("createVaultManager failed: %v", err)
	}

	// Test with invalid token
	_, err = vaultManagerInstance.secret("GOOGLE_KEY")
	if err == nil {
		t.Fatal("Expected error when using invalid token")
	}
	if !strings.Contains(err.Error(), "failed to read secret from path") {
		t.Errorf("Expected 'failed to read secret' error, got: %v", err)
	}
}

// TestSecretFromFile_Success tests successful file secret reading
func TestSecretFromFile_Success(t *testing.T) {
	tempDir := t.TempDir()
	secretDir = tempDir

	// Create a test secret file
	secretFile := filepath.Join(tempDir, "test-secret")
	secretContent := "my-secret-value\n"
	err := os.WriteFile(secretFile, []byte(secretContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test secret file: %v", err)
	}

	result, err := secretFromFile("test-secret")
	if err != nil {
		t.Fatalf("secretFromFile failed: %v", err)
	}

	expected := "my-secret-value"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

// TestSecretFromFile_NoSecretsDir tests file secret reading without configured secrets directory
func TestSecretFromFile_NoSecretsDir(t *testing.T) {
	originalSecretDir := secretDir
	secretDir = ""
	defer func() { secretDir = originalSecretDir }()

	_, err := secretFromFile("test-secret")
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
	secretDir = tempDir

	_, err := secretFromFile("")
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
	secretDir = tempDir

	_, err := secretFromFile("nonexistent-file")
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

	err := config.createSecretSourcesIfNotPresent()
	if err != nil {
		t.Fatalf("createVaultManager failed: %v", err)
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

	err := config.createSecretSourcesIfNotPresent()
	if err != nil {
		t.Fatalf("createVaultManager failed: %v", err)
	}

	defer func() {
		if r := recover(); r != nil {
			if !strings.Contains(r.(error).Error(), "error retrieving secret from Vault") {
				t.Errorf("Expected 'error retrieving secret from Vault' panic, got: %v", r)
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
	secretDir = tempDir

	// Create a test secret file
	secretFile := filepath.Join(tempDir, "test-secret")
	secretContent := "file-secret-value"
	err := os.WriteFile(secretFile, []byte(secretContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test secret file: %v", err)
	}

	result := expand("file:test-secret")
	if result != secretContent {
		t.Errorf("Expected '%s', got '%s'", secretContent, result)
	}
}

// TestExpand_EnvPrefix tests the expand function with env: prefix
func TestExpand_EnvPrefix(t *testing.T) {
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
			if !strings.Contains(r.(string), "unknown prefix in expansion string") {
				t.Errorf("Expected 'unknown prefix' panic, got: %v", r)
			}
		} else {
			t.Fatal("Expected panic for unknown prefix")
		}
	}()

	expand("unknown:value")
}
