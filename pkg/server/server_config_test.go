package server

import (
	"strings"
	"testing"
)

// TestServerConfig_Validate tests server configuration validation
func TestServerConfig_Validate(t *testing.T) {
	tests := []struct {
		name        string
		config      WebServerConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid config",
			config: WebServerConfig{
				Address:       ":8080",
				SessionName:   "test-session",
				SessionSecret: "test-secret",
			},
			expectError: false,
		},
		{
			name: "valid config with host",
			config: WebServerConfig{
				Address:       "localhost:8080",
				SessionName:   "test-session",
				SessionSecret: "test-secret",
			},
			expectError: false,
		},
		{
			name: "missing session secret",
			config: WebServerConfig{
				Address:     ":8080",
				SessionName: "test-session",
			},
			expectError: true,
			errorMsg:    "session_secret must be set and non-empty",
		},
		{
			name: "missing session name",
			config: WebServerConfig{
				Address:       ":8080",
				SessionSecret: "test-secret",
			},
			expectError: true,
			errorMsg:    "session_name must be set and non-empty",
		},
		{
			name: "missing address",
			config: WebServerConfig{
				SessionName:   "test-session",
				SessionSecret: "test-secret",
			},
			expectError: true,
			errorMsg:    "address must be set and non-empty",
		},
		{
			name: "invalid address format",
			config: WebServerConfig{
				Address:       "not-a-valid-address",
				SessionName:   "test-session",
				SessionSecret: "test-secret",
			},
			expectError: true,
			errorMsg:    "invalid address",
		},
		{
			name: "invalid port",
			config: WebServerConfig{
				Address:       ":99999",
				SessionName:   "test-session",
				SessionSecret: "test-secret",
			},
			expectError: true,
			errorMsg:    "invalid address",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}
