package resolver

import (
	"strings"
	"testing"
)

// mockResolver is a simple mock resolver for testing
type mockResolver struct {
	name  string
	value string
}

func (m *mockResolver) Resolve(key string) (string, error) {
	return m.value + ":" + key, nil
}

func (m *mockResolver) Name() string {
	return m.name
}

// TestRegistry_RegisterAndResolve tests basic registration and resolution
func TestRegistry_RegisterAndResolve(t *testing.T) {
	registry := NewRegistry()

	mock := &mockResolver{name: "Mock", value: "mock-value"}
	registry.Register("mock", mock)

	result, err := registry.Resolve("mock:test-key")
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	expected := "mock-value:test-key"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

// TestRegistry_UnknownPrefix tests resolution with unknown prefix
func TestRegistry_UnknownPrefix(t *testing.T) {
	registry := NewRegistry()

	_, err := registry.Resolve("unknown:value")
	if err == nil {
		t.Fatal("Expected error for unknown prefix")
	}

	if !strings.Contains(err.Error(), "no resolver registered for prefix") {
		t.Errorf("Expected 'no resolver registered' error, got: %v", err)
	}
}

// TestRegistry_Unregister tests unregistering a resolver
func TestRegistry_Unregister(t *testing.T) {
	registry := NewRegistry()

	mock := &mockResolver{name: "Mock", value: "value"}
	registry.Register("test", mock)

	// Should work before unregister
	_, err := registry.Resolve("test:key")
	if err != nil {
		t.Fatalf("Resolve failed before unregister: %v", err)
	}

	// Unregister
	registry.Unregister("test")

	// Should fail after unregister
	_, err = registry.Resolve("test:key")
	if err == nil {
		t.Fatal("Expected error after unregistering resolver")
	}
}

// TestRegistry_GetResolver tests getting a registered resolver
func TestRegistry_GetResolver(t *testing.T) {
	registry := NewRegistry()

	mock := &mockResolver{name: "Mock", value: "value"}
	registry.Register("test", mock)

	resolver := registry.GetResolver("test")
	if resolver == nil {
		t.Fatal("Expected to get registered resolver")
	}

	if resolver.Name() != "Mock" {
		t.Errorf("Expected resolver name 'Mock', got '%s'", resolver.Name())
	}
}

// TestRegistry_GetResolver_NonExistent tests getting a non-existent resolver
func TestRegistry_GetResolver_NonExistent(t *testing.T) {
	registry := NewRegistry()

	resolver := registry.GetResolver("nonexistent")
	if resolver != nil {
		t.Error("Expected nil for non-existent resolver")
	}
}

// TestRegistry_ListPrefixes tests listing registered prefixes
func TestRegistry_ListPrefixes(t *testing.T) {
	registry := NewRegistry()

	registry.Register("env", &mockResolver{name: "Env"})
	registry.Register("file", &mockResolver{name: "File"})
	registry.Register("vault", &mockResolver{name: "Vault"})

	prefixes := registry.ListPrefixes()
	if len(prefixes) != 3 {
		t.Errorf("Expected 3 prefixes, got %d", len(prefixes))
	}

	// Check that all expected prefixes are present
	prefixMap := make(map[string]bool)
	for _, p := range prefixes {
		prefixMap[p] = true
	}

	for _, expected := range []string{"env", "file", "vault"} {
		if !prefixMap[expected] {
			t.Errorf("Expected prefix '%s' not found in list", expected)
		}
	}
}

// TestParseProperty tests the property parsing function
func TestParseProperty(t *testing.T) {
	tests := []struct {
		input          string
		expectedPrefix string
		expectedKey    string
	}{
		{"vault:SECRET_KEY", "vault", "SECRET_KEY"},
		{"env:PORT", "env", "PORT"},
		{"file:api_key", "file", "api_key"},
		{"PORT", "env", "PORT"},                         // No prefix defaults to env
		{"custom:db:password", "custom", "db:password"}, // Only first : is separator
		{"", "env", ""},                                 // Empty string defaults to env with empty key
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			prefix, key := ParseProperty(tt.input)
			if prefix != tt.expectedPrefix {
				t.Errorf("Expected prefix '%s', got '%s'", tt.expectedPrefix, prefix)
			}
			if key != tt.expectedKey {
				t.Errorf("Expected key '%s', got '%s'", tt.expectedKey, key)
			}
		})
	}
}

// TestGlobalRegistry tests that the global registry is accessible
func TestGlobalRegistry(t *testing.T) {
	if Global() == nil {
		t.Fatal("Global registry should not be nil")
	}

	// Test using the convenience functions
	mock := &mockResolver{name: "Test", value: "test"}
	Register("testglobal", mock)
	defer Unregister("testglobal")

	result, err := Global().Resolve("testglobal:key")
	if err != nil {
		t.Fatalf("Global resolve failed: %v", err)
	}

	if !strings.Contains(result, "test") {
		t.Errorf("Expected result to contain 'test', got '%s'", result)
	}
}

// TestRegistry_ConcurrentAccess tests thread-safety (basic smoke test)
func TestRegistry_ConcurrentAccess(t *testing.T) {
	registry := NewRegistry()
	mock := &mockResolver{name: "Concurrent", value: "value"}
	registry.Register("test", mock)

	// Run multiple goroutines accessing the registry
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			_, _ = registry.Resolve("test:key")
			_ = registry.GetResolver("test")
			_ = registry.ListPrefixes()
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}
