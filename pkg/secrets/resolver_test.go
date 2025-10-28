package secrets

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
	defer purgeResolvers()

	mock := &mockResolver{name: "Mock", value: "mock-value"}
	Register("mock", mock)

	result, err := Resolve("mock:test-key")
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
	_, err := Resolve("unknown:value")
	if err == nil {
		t.Fatal("Expected error for unknown prefix")
	}

	if !strings.Contains(err.Error(), "no resolver registered for prefix") {
		t.Errorf("Expected 'no resolver registered' error, got: %v", err)
	}
}

// TestRegistry_Unregister tests unregistering a resolver
func TestRegistry_Unregister(t *testing.T) {
	defer purgeResolvers()

	mock := &mockResolver{name: "Mock", value: "value"}
	Register("test", mock)

	// Should work before unregister
	_, err := Resolve("test:key")
	if err != nil {
		t.Fatalf("Resolve failed before unregister: %v", err)
	}

	// Unregister
	Unregister("test")

	// Should fail after unregister
	_, err = Resolve("test:key")
	if err == nil {
		t.Fatal("Expected error after unregistering resolver")
	}
}

// TestRegistry_GetResolver tests getting a registered resolver
func TestRegistry_GetResolver(t *testing.T) {
	defer purgeResolvers()

	mock := &mockResolver{name: "Mock", value: "value"}
	Register("test", mock)

	resolver := GetResolver("test")
	if resolver == nil {
		t.Fatal("Expected to get registered resolver")
	}

	if resolver.Name() != "Mock" {
		t.Errorf("Expected resolver name 'Mock', got '%s'", resolver.Name())
	}
}

// TestRegistry_GetResolver_NonExistent tests getting a non-existent resolver
func TestRegistry_GetResolver_NonExistent(t *testing.T) {
	resolver := GetResolver("nonexistent")
	if resolver != nil {
		t.Error("Expected nil for non-existent resolver")
	}
}

// TestRegistry_ListPrefixes tests listing registered prefixes
func TestRegistry_ListPrefixes(t *testing.T) {
	defer purgeResolvers()

	Register("env", &mockResolver{name: "Env"})
	Register("file", &mockResolver{name: "File"})
	Register("vault", &mockResolver{name: "Vault"})

	prefixes := ListPrefixes()
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
			prefix, key := parseProperty(tt.input)
			if prefix != tt.expectedPrefix {
				t.Errorf("Expected prefix '%s', got '%s'", tt.expectedPrefix, prefix)
			}
			if key != tt.expectedKey {
				t.Errorf("Expected key '%s', got '%s'", tt.expectedKey, key)
			}
		})
	}
}

// TestRegistry_ConcurrentAccess tests thread-safety (basic smoke test)
func TestRegistry_ConcurrentAccess(t *testing.T) {
	defer purgeResolvers()
	mock := &mockResolver{name: "Concurrent", value: "value"}
	Register("test", mock)

	// Run multiple goroutines accessing the registry
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			_, _ = Resolve("test:key")
			_ = GetResolver("test")
			_ = ListPrefixes()
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

func purgeResolvers() {
	resolvers = make(map[string]PropertyResolver)
}
