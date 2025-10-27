package server

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/animalet/sargantana-go/pkg/config"
	"github.com/animalet/sargantana-go/pkg/controller"
	"github.com/animalet/sargantana-go/pkg/database"
	"github.com/animalet/sargantana-go/pkg/session"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// LogWriter is a helper to capture log output
type LogWriter struct {
	buffer *bytes.Buffer
}

func (lw *LogWriter) Write(p []byte) (n int, err error) {
	return lw.buffer.Write(p)
}

// MockController implements IController for testing purposes
type MockController struct {
	bindCalled  bool
	closeCalled bool
	closeError  error
}

func (m *MockController) Bind(engine *gin.Engine, loginMiddleware gin.HandlerFunc) {
	_ = loginMiddleware // Suppress unused parameter warning
	m.bindCalled = true
	// Add a simple test route
	engine.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "test response"})
	})
}

func (m *MockController) Close() error {
	m.closeCalled = true
	return m.closeError
}

func setupTestEnvironment() {
	// Register a mock controller for testing
	AddControllerType("mock", func(controllerConfig config.ControllerConfig, ctx controller.ControllerContext) (controller.IController, error) {
		return &MockController{}, nil
	})
}

func createTestConfigFile(t *testing.T, content string) string {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "test_config.yaml")

	err := os.WriteFile(configFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	return configFile
}

func TestServerWithRedisSessionsAndDebugMode(t *testing.T) {
	setupTestEnvironment()

	// Create a temporary secrets directory
	tempDir := t.TempDir()
	secretsDir := filepath.Join(tempDir, "secrets")
	err := os.MkdirAll(secretsDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create secrets directory: %v", err)
	}

	// Create session secret file
	secretFile := filepath.Join(secretsDir, "session_secret")
	err = os.WriteFile(secretFile, []byte("test-session-secret-key"), 0644)
	if err != nil {
		t.Fatalf("Failed to create session secret file: %v", err)
	}

	configContent := `
server:
  address: "localhost:0"
  debug: true
  session_name: "test_session"
  session_secret: "test-session-secret-key"
  secrets_dir: "` + secretsDir + `"
redis:
  address: "localhost:6380"
  username: "redisuser"
  password: "redispass"
  max_idle: 10
  idle_timeout: 240s
  tls:
    insecure_skip_verify: true
controllers:
  - type: "mock"
    name: "test_controller"
    config: {}
`

	configFile := createTestConfigFile(t, configContent)

	// Capture log output to verify debug mode
	var logBuffer bytes.Buffer
	logWriter := &LogWriter{buffer: &logBuffer}
	originalWriter := log.Logger
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: logWriter})
	defer func() { log.Logger = originalWriter }()

	// Create server
	previousDebug := debug
	defer func() {
		debug = previousDebug
	}()
	SetDebug(true)
	server, err := NewServerFromConfigFile(configFile)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	if server.config.ServerConfig.SessionName != "test_session" {
		t.Error("Expected session name to be 'test_session'")
	}

	// Manually create and set a Redis session store using SetSessionStore
	// This demonstrates the new pattern where session stores are created externally
	redisConfig, err := config.LoadConfig[database.RedisConfig]("redis", server.config)
	if err != nil {
		t.Fatalf("Failed to load Redis config: %v", err)
	}

	pool, err := redisConfig.CreateClient()
	if err != nil {
		// Redis connection may fail in test environment, try with default cookie store
		t.Logf("Redis connection failed (expected in test without Redis): %v", err)
		// Don't set a custom store, let it use default cookie store
	} else {
		// Create Redis session store and set it
		defer func() { _ = pool.Close() }()
		redisStore, err := session.NewRedisSessionStore(false, []byte("test-session-secret-key"), pool)
		if err != nil {
			t.Fatalf("Failed to create Redis session store: %v", err)
		}
		server.SetSessionStore(&redisStore)
	}

	// Start server
	err = server.Start()

	// Check the logs for debug mode
	logOutput := logBuffer.String()

	// Verify debug mode messages are present
	if !strings.Contains(logOutput, "Debug mode is enabled") {
		t.Error("Expected debug mode message not found in logs")
	}

	// If server started successfully, verify the mock controller was bound
	if err == nil && server.httpServer != nil {
		if len(server.controllers) > 0 {
			if mockController, ok := server.controllers[0].(*MockController); ok {
				if !mockController.bindCalled {
					t.Error("Expected controller Bind method to be called when server starts successfully")
				}
			}
		} else {
			t.Error("Expected controllers to be configured after successful start")
		}

		// Cleanup
		err := server.Shutdown()
		if err != nil {
			t.Errorf("Failed to shutdown server: %v", err)
		}
	} else {
		t.Fatalf("Server failed to start: %v", err)
	}
}

func TestServerBodyLogMiddlewareInDebugMode(t *testing.T) {
	setupTestEnvironment()

	// Create a temporary secrets directory
	tempDir := t.TempDir()
	secretsDir := filepath.Join(tempDir, "secrets")
	err := os.MkdirAll(secretsDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create secrets directory: %v", err)
	}

	// Create session secret file
	secretFile := filepath.Join(secretsDir, "session_secret")
	err = os.WriteFile(secretFile, []byte("test-session-secret-key"), 0644)
	if err != nil {
		t.Fatalf("Failed to create session secret file: %v", err)
	}

	configContent := `
server:
  address: "localhost:0"
  debug: true
  session_name: "test_session"
  session_secret: "test-session-secret-key"
  secrets_dir: "` + secretsDir + `"
controllers:
  - type: "mock"
    name: "test_controller"
    config: {}
`

	configFile := createTestConfigFile(t, configContent)

	// Capture log output to verify bodyLogMiddleware is working
	var logBuffer bytes.Buffer
	logWriter := &LogWriter{buffer: &logBuffer}
	originalWriter := log.Logger
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: logWriter})
	defer func() { log.Logger = originalWriter }()

	// Create and start server
	previousDebug := debug
	defer func() {
		debug = previousDebug
	}()
	SetDebug(true)
	server, err := NewServerFromConfigFile(configFile)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	err = server.Start()
	if err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}

	// Make a test request to verify bodyLogMiddleware is working
	if server.httpServer != nil {
		// Create a test request
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		// Send request through the server's handler
		server.httpServer.Handler.ServeHTTP(w, req)

		// Verify response
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		// Verify bodyLogMiddleware logged the response
		logOutput := logBuffer.String()
		if !strings.Contains(logOutput, "Response body:") {
			t.Error("Expected bodyLogMiddleware to log response body")
		}

		if !strings.Contains(logOutput, "test response") {
			t.Error("Expected response body to contain 'test response'")
		}
	}

	// Cleanup
	if server.httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err := server.httpServer.Shutdown(ctx)
		if err != nil {
			t.Errorf("Failed to shutdown server: %v", err)
		}
	}
}

func TestServerWithCookieSessionsInReleaseMode(t *testing.T) {
	setupTestEnvironment()

	// Create a temporary secrets directory
	tempDir := t.TempDir()
	secretsDir := filepath.Join(tempDir, "secrets")
	err := os.MkdirAll(secretsDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create secrets directory: %v", err)
	}

	// Create session secret file
	secretFile := filepath.Join(secretsDir, "session_secret")
	err = os.WriteFile(secretFile, []byte("test-session-secret-key"), 0644)
	if err != nil {
		t.Fatalf("Failed to create session secret file: %v", err)
	}

	configContent := `
server:
  address: "localhost:0"
  debug: false
  session_name: "test_session"
  session_secret: "test-session-secret-key"
  secrets_dir: "` + secretsDir + `"
controllers:
  - type: "mock"
    name: "test_controller"
    config: {}
`

	configFile := createTestConfigFile(t, configContent)

	// Capture log output
	var logBuffer bytes.Buffer
	logWriter := &LogWriter{buffer: &logBuffer}
	originalWriter := log.Logger
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: logWriter})
	defer func() { log.Logger = originalWriter }()

	// Create and start server
	server, err := NewServerFromConfigFile(configFile)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	err = server.Start()
	if err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}

	// Verify release mode configuration
	logOutput := logBuffer.String()

	if strings.Contains(logOutput, "Debug mode is enabled") {
		t.Error("Debug mode should not be enabled in release mode")
	}

	if !strings.Contains(logOutput, "Running in release mode") {
		t.Error("Expected release mode message not found in logs")
	}

	if !strings.Contains(logOutput, "Using default cookie-based session storage") {
		t.Error("Expected cookie session storage message not found in logs")
	}

	if server.config.ServerConfig.RedisSessionStore != nil {
		t.Error("Expected Redis session store to be nil when not configured")
	}

	// Cleanup
	if server.httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err := server.httpServer.Shutdown(ctx)
		if err != nil {
			t.Errorf("Failed to shutdown server: %v", err)
		}
	}
}

func TestServerShutdown(t *testing.T) {
	setupTestEnvironment()

	// Create a temporary secrets directory
	tempDir := t.TempDir()
	secretsDir := filepath.Join(tempDir, "secrets")
	err := os.MkdirAll(secretsDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create secrets directory: %v", err)
	}

	// Create session secret file
	secretFile := filepath.Join(secretsDir, "session_secret")
	err = os.WriteFile(secretFile, []byte("test-session-secret-key"), 0644)
	if err != nil {
		t.Fatalf("Failed to create session secret file: %v", err)
	}

	configContent := `
server:
  address: "localhost:0"
  debug: true
  session_name: "test_session"
  session_secret: "test-session-secret-key"
  secrets_dir: "` + secretsDir + `"
controllers:
  - type: "mock"
    name: "test_controller"
    config: {}
`

	configFile := createTestConfigFile(t, configContent)

	// Capture log output
	var logBuffer bytes.Buffer
	logWriter := &LogWriter{buffer: &logBuffer}
	originalWriter := log.Logger
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: logWriter})
	defer func() { log.Logger = originalWriter }()

	// Create and start server
	server, err := NewServerFromConfigFile(configFile)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	err = server.Start()
	if err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}

	// Test shutdown
	err = server.Shutdown()
	if err != nil {
		t.Errorf("Failed to shutdown server: %v", err)
	}

	// Verify shutdown hooks were called
	if mockController, ok := server.controllers[0].(*MockController); ok {
		if !mockController.closeCalled {
			t.Error("Expected controller Close method to be called during shutdown")
		}
	} else {
		t.Error("Expected mock controller to be configured")
	}

	// Verify shutdown messages
	logOutput := logBuffer.String()
	if !strings.Contains(logOutput, "Shutting down server...") {
		t.Error("Expected shutdown message not found in logs")
	}

	if !strings.Contains(logOutput, "Executing shutdown hooks...") {
		t.Error("Expected shutdown hooks message not found in logs")
	}

	if !strings.Contains(logOutput, "Server exited gracefully") {
		t.Error("Expected graceful exit message not found in logs")
	}
}

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
					SessionName:   "test-session",
					SessionSecret: "test-secret",
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
  secrets_dir: "` + tt.configData.ServerConfig.SecretsDir + `"
  session_name: "` + tt.configData.ServerConfig.SessionName + `"
  session_secret: "` + tt.configData.ServerConfig.SessionSecret + `"
controllers: []`

			err := os.WriteFile(configFile, []byte(configContent), 0644)
			if err != nil {
				t.Fatalf("Failed to write config file: %v", err)
			}

			server, err := NewServerFromConfigFile(configFile)

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

				// Redis session store should be nil for basic config
				if server.config.ServerConfig.RedisSessionStore != nil {
					t.Error("Expected Redis session store to be nil for basic config")
				}

				if server.config.ServerConfig.SecretsDir != tt.configData.ServerConfig.SecretsDir {
					t.Errorf("SecretsDir = %v, want %v", server.config.ServerConfig.SecretsDir, tt.configData.ServerConfig.SecretsDir)
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
			server, err := NewServerFromConfigFile(tt.configFile)

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

	AddControllerType("static", controller.NewStaticController)
	// Create a temporary config file with controller bindings
	tempDir := t.TempDir()
	staticDir := filepath.Join(tempDir, "static")
	err := os.MkdirAll(staticDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create static dir: %v", err)
	}

	configFile := filepath.Join(tempDir, "config.yaml")

	configContent := fmt.Sprintf(`server:
  address: "localhost:8080"
  debug: true
  session_name: "test-session"
  session_secret: "test-secret"
controllers:
  - type: "static"
    config:
      path: "/static"
      dir: "%s"`, staticDir)

	err = os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	server, err := NewServerFromConfigFile(configFile)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	if server == nil {
		t.Fatal("NewServer returned nil")
	}

	// Note: Controllers are now configured during Start() (in bootstrap()), not in NewServer()
	// This is because they need access to the session store via ControllerContext
	// So we need to start the server to test controller configuration
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

			server, err := NewServerFromConfigFile(configFile)
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

	server, err := NewServerFromConfigFile(configFile)
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

	server, err := NewServerFromConfigFile(configFile)
	if err == nil {
		t.Error("Expected error for invalid config but got none")
	}
	if server != nil {
		t.Error("Expected nil server for invalid config")
	}
}

func TestSetDebugAndGetDebug(t *testing.T) {
	// Store original values to restore later
	originalDebug := GetDebug()
	originalLogLevel := zerolog.GlobalLevel()
	defer func() {
		SetDebug(originalDebug)
		zerolog.SetGlobalLevel(originalLogLevel)
	}()

	// Test setting debug to true
	SetDebug(true)
	if !GetDebug() {
		t.Error("Expected GetDebug() to return true after SetDebug(true)")
	}

	// Verify logger level is set to DEBUG when debug is enabled
	if zerolog.GlobalLevel() != zerolog.DebugLevel {
		t.Errorf("Expected logger level to be DEBUG when debug is enabled, got %v", zerolog.GlobalLevel())
	}

	// Test setting debug to false
	SetDebug(false)
	if GetDebug() {
		t.Error("Expected GetDebug() to return false after SetDebug(false)")
	}

	// Verify logger level is set to INFO when debug is disabled
	if zerolog.GlobalLevel() != zerolog.InfoLevel {
		t.Errorf("Expected logger level to be INFO when debug is disabled, got %v", zerolog.GlobalLevel())
	}
}

func TestSetDebugLoggerFlags(t *testing.T) {
	// Store original values to restore later
	originalDebug := GetDebug()
	defer SetDebug(originalDebug)

	// Test that SetDebug(true) sets appropriate log flags
	SetDebug(true)
	// We can't directly test the flags, but we can verify the function runs without error
	// and that debug mode is properly set
	if !GetDebug() {
		t.Error("Expected debug mode to be enabled")
	}

	// Test that SetDebug(false) sets different log flags
	SetDebug(false)
	if GetDebug() {
		t.Error("Expected debug mode to be disabled")
	}
}

func TestGetDebugInitialState(t *testing.T) {
	// Test that GetDebug returns the current state of the debug variable
	// Since debug is initialized to false, this should be the initial state
	// unless modified by other tests
	currentDebug := GetDebug()

	// Set to a known state and verify
	SetDebug(true)
	if !GetDebug() {
		t.Error("Expected GetDebug() to return true after explicitly setting debug to true")
	}

	SetDebug(false)
	if GetDebug() {
		t.Error("Expected GetDebug() to return false after explicitly setting debug to false")
	}

	// Restore original state
	SetDebug(currentDebug)
}

func TestDebugToggling(t *testing.T) {
	// Store original values to restore later
	originalDebug := GetDebug()
	defer SetDebug(originalDebug)

	// Test multiple toggles
	SetDebug(true)
	firstState := GetDebug()

	SetDebug(false)
	secondState := GetDebug()

	SetDebug(true)
	thirdState := GetDebug()

	if !firstState {
		t.Error("Expected first state (after SetDebug(true)) to be true")
	}
	if secondState {
		t.Error("Expected second state (after SetDebug(false)) to be false")
	}
	if !thirdState {
		t.Error("Expected third state (after SetDebug(true)) to be true")
	}
}

func TestAddControllerType(t *testing.T) {
	// Store original registry state
	originalRegistry := make(map[string]controller.Constructor)
	for k, v := range controllerRegistry {
		originalRegistry[k] = v
	}
	defer func() {
		// Restore original registry
		controllerRegistry = originalRegistry
	}()

	// Test adding a new controller type
	mockConstructor := func(controllerConfig config.ControllerConfig, ctx controller.ControllerContext) (controller.IController, error) {
		return &MockController{}, nil
	}

	AddControllerType("test-controller", mockConstructor)

	if _, exists := controllerRegistry["test-controller"]; !exists {
		t.Error("Expected controller type 'test-controller' to be registered")
	}

	// Test overriding an existing controller type
	mockConstructor2 := func(controllerConfig config.ControllerConfig, ctx controller.ControllerContext) (controller.IController, error) {
		return &MockController{}, nil
	}

	AddControllerType("test-controller", mockConstructor2)

	if _, exists := controllerRegistry["test-controller"]; !exists {
		t.Error("Expected controller type 'test-controller' to still be registered after override")
	}
}

func TestSetSessionStore(t *testing.T) {
	// Create a temporary config file
	configYAML := `
server:
  address: ":18084"
  session_name: "test-session"
  session_secret: "my-test-secret-key-that-is-long-enough"
`
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.yaml")
	err := os.WriteFile(configFile, []byte(configYAML), 0644)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	server, err := NewServerFromConfigFile(configFile)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Test that initially the session store is nil (not yet bootstrapped)
	if server.sessionStore != nil {
		t.Error("Expected session store to be nil before bootstrap")
	}

	// Create a custom session store
	customStore := session.NewCookieStore(false, []byte("test-secret-key-for-sessions-here"))

	// Test SetSessionStore
	server.SetSessionStore(&customStore)

	// Verify the session store was set
	if server.sessionStore == nil {
		t.Error("Expected session store to be set")
	}
	if server.sessionStore != &customStore {
		t.Error("Expected session store to match the one we set")
	}
}
