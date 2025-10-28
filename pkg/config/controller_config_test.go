package config

import (
	"strings"
	"testing"
)

// TestControllerBindings_Validate tests validation of multiple controller bindings
func TestControllerBindings_Validate(t *testing.T) {
	tests := []struct {
		name        string
		bindings    ControllerBindings
		expectError bool
		errorMsg    string
	}{
		{
			name: "all valid bindings",
			bindings: ControllerBindings{
				{
					TypeName:   "auth",
					Name:       "oauth",
					ConfigData: []byte("key: value"),
				},
				{
					TypeName:   "static",
					Name:       "public",
					ConfigData: []byte("path: /public"),
				},
			},
			expectError: false,
		},
		{
			name:        "empty bindings list",
			bindings:    ControllerBindings{},
			expectError: false,
		},
		{
			name: "one invalid binding",
			bindings: ControllerBindings{
				{
					TypeName:   "auth",
					Name:       "oauth",
					ConfigData: []byte("key: value"),
				},
				{
					TypeName:   "", // Invalid - missing type
					Name:       "invalid",
					ConfigData: []byte("key: value"),
				},
			},
			expectError: true,
			errorMsg:    "controller binding at index 1 is invalid",
		},
		{
			name: "multiple invalid bindings",
			bindings: ControllerBindings{
				{
					TypeName:   "", // Invalid
					ConfigData: []byte("key: value"),
				},
				{
					TypeName:   "static",
					ConfigData: nil, // Invalid
				},
			},
			expectError: true,
			errorMsg:    "configuration validation failed",
		},
		{
			name: "valid binding without name",
			bindings: ControllerBindings{
				{
					TypeName:   "static",
					ConfigData: []byte("path: /public"),
					// Name omitted - should be valid
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.bindings.Validate()
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
