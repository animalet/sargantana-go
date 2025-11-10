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

	if !strings.Contains(err.Error(), "no secret provider registered for prefix") {
		t.Errorf("Expected 'no secret provider registered' error, got: %v", err)
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

func purgeResolvers() {
	providers = make(map[string]SecretLoader)
}
