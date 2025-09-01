package server

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/animalet/sargantana-go/config"
	"github.com/gin-gonic/gin"
)

func TestNewServer(t *testing.T) {
	tests := []struct {
		name        string
		configData  config.Config
		expectError bool
	}{
		{
			name: "basic server config",
			configData: config.Config{
				ServerConfig: config.ServerConfig{
					Address:       "localhost:8080",
					Debug:         false,
					SessionName:   "test-session",
					SessionSecret: "test-secret",
				},
				ControllerBindings: []config.ControllerBinding{},
			},
			expectError: false,
		},
		{
			name: "debug server config",
			configData: config.Config{
				ServerConfig: config.ServerConfig{
					Address:           "0.0.0.0:9000",
					RedisSessionStore: "localhost:6379",
					SecretsDir:        "/secrets",
					Debug:             true,
					SessionName:       "redis-session",
					SessionSecret:     "test-secret",
				},
				ControllerBindings: []config.ControllerBinding{},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary config file
			tempDir := t.TempDir()
			configFile := filepath.Join(tempDir, "config.yaml")

			configContent := `server:
  address: "` + tt.configData.ServerConfig.Address + `"
  redis_session_store: "` + tt.configData.ServerConfig.RedisSessionStore + `"
  secrets_dir: "` + tt.configData.ServerConfig.SecretsDir + `"
  debug: ` + boolToString(tt.configData.ServerConfig.Debug) + `
  session_name: "` + tt.configData.ServerConfig.SessionName + `"
  session_secret: "` + tt.configData.ServerConfig.SessionSecret + `"
controllers: []`

			err := os.WriteFile(configFile, []byte(configContent), 0644)
			if err != nil {
				t.Fatalf("Failed to write config file: %v", err)
			}

			server, err := NewServer(configFile)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if server == nil {
					t.Fatal("NewServer returned nil")
				}
				if server.config == nil {
					t.Fatal("Server config is nil")
				}

				if server.config.ServerConfig.Address != tt.configData.ServerConfig.Address {
					t.Errorf("Address = %v, want %v", server.config.ServerConfig.Address, tt.configData.ServerConfig.Address)
				}
				if server.config.ServerConfig.RedisSessionStore != tt.configData.ServerConfig.RedisSessionStore {
					t.Errorf("Redis = %v, want %v", server.config.ServerConfig.RedisSessionStore, tt.configData.ServerConfig.RedisSessionStore)
				}
				if server.config.ServerConfig.SecretsDir != tt.configData.ServerConfig.SecretsDir {
					t.Errorf("SecretsDir = %v, want %v", server.config.ServerConfig.SecretsDir, tt.configData.ServerConfig.SecretsDir)
				}
				if server.config.ServerConfig.Debug != tt.configData.ServerConfig.Debug {
					t.Errorf("Debug = %v, want %v", server.config.ServerConfig.Debug, tt.configData.ServerConfig.Debug)
				}
				if server.config.ServerConfig.SessionName != tt.configData.ServerConfig.SessionName {
					t.Errorf("SessionName = %v, want %v", server.config.ServerConfig.SessionName, tt.configData.ServerConfig.SessionName)
				}
			}
		})
	}
}

func boolToString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func TestNewServer_InvalidConfigFile(t *testing.T) {
	tests := []struct {
		name       string
		configFile string
	}{
		{
			name:       "non-existent file",
			configFile: "/non/existent/file.yaml",
		},
		{
			name:       "empty path",
			configFile: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, err := NewServer(tt.configFile)

			if err == nil {
				t.Error("Expected error but got none")
			}
			if server != nil {
				t.Error("Expected nil server but got a value")
			}
		})
	}
}

func TestNewServer_WithControllers(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create a temporary config file with controller bindings
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.yaml")

	configContent := `server:
  address: "localhost:8080"
  debug: true
  session_name: "test-session"
  session_secret: "test-secret"
controllers:
  - type: "static"
    config:
      statics_dir: "./static"
      templates_dir: "./templates"`

	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	server, err := NewServer(configFile)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	if server == nil {
		t.Fatal("NewServer returned nil")
	}
	if len(server.controllers) == 0 {
		t.Error("Expected at least one controller to be configured")
	}
}

func TestServer_Start(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Set up environment
	err := os.Setenv("SESSION_SECRET", "test-secret")
	if err != nil {
		t.Fatalf("Failed to set SESSION_SECRET: %v", err)
	}
	defer func() {
		_ = os.Unsetenv("SESSION_SECRET")
	}()

	tests := []struct {
		name  string
		debug bool
	}{
		{
			name:  "debug mode",
			debug: true,
		},
		{
			name:  "release mode",
			debug: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary config file
			tempDir := t.TempDir()
			configFile := filepath.Join(tempDir, "config.yaml")

			configContent := `server:
  address: "localhost:0"
  debug: ` + boolToString(tt.debug) + `
  session_name: "test-session"
  session_secret: "test-secret"
controllers: []`

			err := os.WriteFile(configFile, []byte(configContent), 0644)
			if err != nil {
				t.Fatalf("Failed to write config file: %v", err)
			}

			server, err := NewServer(configFile)
			if err != nil {
				t.Fatalf("Failed to create server: %v", err)
			}

			err = server.Start()
			if err != nil {
				t.Fatalf("Start() returned error: %v", err)
			}

			if server.httpServer == nil {
				t.Fatalf("httpServer not initialized")
			}

			// Cleanup
			err = server.Shutdown()
			if err != nil {
				t.Fatalf("Shutdown() returned error: %v", err)
			}
		})
	}
}

func TestServer_StartAndWaitForSignal(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Set up environment
	err := os.Setenv("SESSION_SECRET", "test-secret")
	if err != nil {
		t.Fatalf("Failed to set SESSION_SECRET: %v", err)
	}
	defer func() {
		_ = os.Unsetenv("SESSION_SECRET")
	}()

	// Create a temporary config file
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.yaml")

	configContent := `server:
  address: "localhost:0"
  debug: true
  session_name: "test-session"
  session_secret: "test-secret"
controllers: []`

	err = os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	server, err := NewServer(configFile)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Start server in goroutine
	errorChan := make(chan error, 1)
	go func() {
		err := server.StartAndWaitForSignal()
		errorChan <- err
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Send shutdown signal
	if server.shutdownChannel != nil {
		server.shutdownChannel <- os.Interrupt
	} else {
		// Force shutdown if channel not initialized
		err := server.Shutdown()
		if err != nil {
			t.Errorf("Shutdown() returned error: %v", err)
		}
	}

	// Wait for completion with timeout
	select {
	case err := <-errorChan:
		if err != nil {
			t.Errorf("StartAndWaitForSignal() returned error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Error("StartAndWaitForSignal() timed out")
		err := server.Shutdown()
		if err != nil {
			t.Errorf("Shutdown() returned error during timeout: %v", err)
		}
	}
}

func TestServer_Shutdown(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Set up environment
	err := os.Setenv("SESSION_SECRET", "test-secret")
	if err != nil {
		t.Fatalf("Failed to set SESSION_SECRET: %v", err)
	}
	defer func() {
		_ = os.Unsetenv("SESSION_SECRET")
	}()

	// Create a temporary config file
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.yaml")

	configContent := `server:
  address: "localhost:0"
  debug: false
  session_name: "test-session"
  session_secret: "test-secret"
controllers: []`

	err = os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	server, err := NewServer(configFile)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Start server
	err = server.Start()
	if err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}

	// Test shutdown
	err = server.Shutdown()
	if err != nil {
		t.Errorf("Shutdown() returned error: %v", err)
	}
}

func TestServer_InvalidConfig(t *testing.T) {
	// Create a temporary config file with invalid YAML
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.yaml")

	invalidConfigContent := `server:
  address: "localhost:8080"
  debug: not-a-boolean
controllers: []`

	err := os.WriteFile(configFile, []byte(invalidConfigContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	server, err := NewServer(configFile)
	if err == nil {
		t.Error("Expected error for invalid config but got none")
	}
	if server != nil {
		t.Error("Expected nil server for invalid config")
	}
}
