package config

import (
	"testing"
)

func TestNewConfig(t *testing.T) {
	tests := []struct {
		name              string
		address           string
		redisSessionStore string
		secretsDir        string
		debug             bool
		sessionName       string
	}{
		{
			name:              "basic config",
			address:           "localhost:8080",
			redisSessionStore: "",
			secretsDir:        "/secrets",
			debug:             true,
			sessionName:       "app-session",
		},
		{
			name:              "with redis",
			address:           "0.0.0.0:9000",
			redisSessionStore: "localhost:6379",
			secretsDir:        "/var/secrets",
			debug:             false,
			sessionName:       "redis-session",
		},
		{
			name:              "minimal config",
			address:           ":3000",
			redisSessionStore: "",
			secretsDir:        "",
			debug:             false,
			sessionName:       "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := NewConfig(tt.address, tt.redisSessionStore, tt.secretsDir, tt.debug, tt.sessionName)

			if config.Address() != tt.address {
				t.Errorf("Address() = %v, want %v", config.Address(), tt.address)
			}
			if config.RedisSessionStore() != tt.redisSessionStore {
				t.Errorf("RedisSessionStore() = %v, want %v", config.RedisSessionStore(), tt.redisSessionStore)
			}
			if config.SecretsDir() != tt.secretsDir {
				t.Errorf("SecretsDir() = %v, want %v", config.SecretsDir(), tt.secretsDir)
			}
			if config.Debug() != tt.debug {
				t.Errorf("Debug() = %v, want %v", config.Debug(), tt.debug)
			}
			if config.SessionName() != tt.sessionName {
				t.Errorf("SessionName() = %v, want %v", config.SessionName(), tt.sessionName)
			}
		})
	}
}

func TestConfig_Getters(t *testing.T) {
	config := NewConfig("test:1234", "redis:6379", "/test/secrets", true, "test-session")

	// Test all getter methods
	if got := config.Address(); got != "test:1234" {
		t.Errorf("Address() = %v, want %v", got, "test:1234")
	}

	if got := config.RedisSessionStore(); got != "redis:6379" {
		t.Errorf("RedisSessionStore() = %v, want %v", got, "redis:6379")
	}

	if got := config.SecretsDir(); got != "/test/secrets" {
		t.Errorf("SecretsDir() = %v, want %v", got, "/test/secrets")
	}

	if got := config.Debug(); got != true {
		t.Errorf("Debug() = %v, want %v", got, true)
	}

	if got := config.SessionName(); got != "test-session" {
		t.Errorf("SessionName() = %v, want %v", got, "test-session")
	}
}

func TestConfig_EmptyValues(t *testing.T) {
	config := NewConfig("", "", "", false, "")

	if config.Address() != "" {
		t.Errorf("Expected empty address, got %v", config.Address())
	}
	if config.RedisSessionStore() != "" {
		t.Errorf("Expected empty redis store, got %v", config.RedisSessionStore())
	}
	if config.SecretsDir() != "" {
		t.Errorf("Expected empty secrets dir, got %v", config.SecretsDir())
	}
	if config.Debug() != false {
		t.Errorf("Expected debug false, got %v", config.Debug())
	}
	if config.SessionName() != "" {
		t.Errorf("Expected empty session name, got %v", config.SessionName())
	}
}
