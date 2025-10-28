package secrets

import (
	"os"
	"testing"
)

// TestEnvResolver_Success tests successful environment variable resolution
func TestEnvResolver_Success(t *testing.T) {
	// Set up test environment variable
	testKey := "TEST_ENV_RESOLVER_VAR"
	testValue := "test-env-value"
	_ = os.Setenv(testKey, testValue)
	defer func() { _ = os.Unsetenv(testKey) }()

	resolver := NewEnvResolver()
	result, err := resolver.Resolve(testKey)
	if err != nil {
		t.Fatalf("EnvResolver.Resolve failed: %v", err)
	}

	if result != testValue {
		t.Errorf("Expected '%s', got '%s'", testValue, result)
	}
}

// TestEnvResolver_EmptyValue tests resolution of empty environment variable
func TestEnvResolver_EmptyValue(t *testing.T) {
	testKey := "TEST_ENV_RESOLVER_EMPTY"
	_ = os.Setenv(testKey, "")
	defer func() { _ = os.Unsetenv(testKey) }()

	resolver := NewEnvResolver()
	result, err := resolver.Resolve(testKey)
	if err != nil {
		t.Fatalf("EnvResolver.Resolve failed: %v", err)
	}

	if result != "" {
		t.Errorf("Expected empty string, got '%s'", result)
	}
}

// TestEnvResolver_NonexistentVariable tests resolution of nonexistent variable
func TestEnvResolver_NonexistentVariable(t *testing.T) {
	resolver := NewEnvResolver()

	// Make sure the variable doesn't exist
	testKey := "TEST_ENV_RESOLVER_NONEXISTENT_12345"
	_ = os.Unsetenv(testKey)

	result, err := resolver.Resolve(testKey)
	if err != nil {
		t.Fatalf("EnvResolver.Resolve should not error for missing vars: %v", err)
	}

	// EnvResolver returns empty string for missing vars (Go's os.Getenv behavior)
	if result != "" {
		t.Errorf("Expected empty string for nonexistent var, got '%s'", result)
	}
}

// TestEnvResolver_Name tests the Name method
func TestEnvResolver_Name(t *testing.T) {
	resolver := NewEnvResolver()
	if resolver.Name() != "Environment" {
		t.Errorf("Expected name 'Environment', got '%s'", resolver.Name())
	}
}
