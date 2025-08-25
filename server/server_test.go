package server

import (
	"flag"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/animalet/sargantana-go/config"
	"github.com/animalet/sargantana-go/controller"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

func TestNewServer(t *testing.T) {
	tests := []struct {
		name        string
		host        string
		port        int
		redis       string
		secretsDir  string
		debug       bool
		sessionName string
	}{
		{
			name:        "basic server",
			host:        "localhost",
			port:        8080,
			redis:       "",
			secretsDir:  "",
			debug:       false,
			sessionName: "test-session",
		},
		{
			name:        "debug server with redis",
			host:        "0.0.0.0",
			port:        9000,
			redis:       "localhost:6379",
			secretsDir:  "/secrets",
			debug:       true,
			sessionName: "redis-session",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := NewServer(tt.host, tt.port, tt.redis, tt.secretsDir, tt.debug, tt.sessionName)

			if server == nil {
				t.Fatal("NewServer returned nil")
			}
			if server.config == nil {
				t.Fatal("Server config is nil")
			}

			expectedAddress := tt.host + ":" + string(rune(tt.port/1000+48)) + string(rune((tt.port%1000)/100+48)) + string(rune((tt.port%100)/10+48)) + string(rune(tt.port%10+48))
			switch tt.port {
			case 8080:
				expectedAddress = tt.host + ":8080"
			case 9000:
				expectedAddress = tt.host + ":9000"
			}

			if server.config.Address() != expectedAddress {
				t.Errorf("Address = %v, want %v", server.config.Address(), expectedAddress)
			}
			if server.config.RedisSessionStore() != tt.redis {
				t.Errorf("Redis = %v, want %v", server.config.RedisSessionStore(), tt.redis)
			}
			if server.config.SecretsDir() != tt.secretsDir {
				t.Errorf("SecretsDir = %v, want %v", server.config.SecretsDir(), tt.secretsDir)
			}
			if server.config.Debug() != tt.debug {
				t.Errorf("Debug = %v, want %v", server.config.Debug(), tt.debug)
			}
			if server.config.SessionName() != tt.sessionName {
				t.Errorf("SessionName = %v, want %v", server.config.SessionName(), tt.sessionName)
			}
		})
	}
}

func TestNewServerFromFlags(t *testing.T) {
	// Save original args
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	tests := []struct {
		name          string
		args          []string
		expectedHost  string
		expectedPort  string
		expectedDebug bool
	}{
		{
			name:          "default values",
			args:          []string{"program"},
			expectedHost:  "localhost",
			expectedPort:  "8080",
			expectedDebug: false,
		},
		{
			name:          "custom values",
			args:          []string{"program", "-host=0.0.0.0", "-port=9090", "-debug=true"},
			expectedHost:  "0.0.0.0",
			expectedPort:  "9090",
			expectedDebug: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Args = tt.args

			// Create a dummy controller initializer
			dummyInit := func(fs *flag.FlagSet) func() controller.IController {
				return func() controller.IController {
					return &mockController{name: "test"}
				}
			}

			server, controllers := NewServerFromFlags(dummyInit)

			if server == nil {
				t.Fatal("NewServerFromFlags returned nil server")
			}
			if len(controllers) != 1 {
				t.Errorf("Expected 1 controller, got %d", len(controllers))
			}

			expectedAddress := tt.expectedHost + ":" + tt.expectedPort
			if server.config.Address() != expectedAddress {
				t.Errorf("Address = %v, want %v", server.config.Address(), expectedAddress)
			}
			if server.config.Debug() != tt.expectedDebug {
				t.Errorf("Debug = %v, want %v", server.config.Debug(), tt.expectedDebug)
			}
		})
	}
}

func TestServer_LoadSecrets(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func(string) error
		secretsDir  string
		expectError bool
	}{
		{
			name: "valid secrets",
			setupFunc: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, "TEST_SECRET"), []byte("secret_value"), 0644)
			},
			expectError: false,
		},
		{
			name: "empty directory",
			setupFunc: func(dir string) error {
				return nil // Create empty directory
			},
			expectError: false,
		},
		{
			name:        "non-existent directory",
			setupFunc:   nil,
			secretsDir:  "/non/existent/path",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var secretsDir string
			if tt.secretsDir != "" {
				secretsDir = tt.secretsDir
			} else {
				secretsDir = t.TempDir()
				if tt.setupFunc != nil {
					err := tt.setupFunc(secretsDir)
					if err != nil {
						t.Fatalf("Setup failed: %v", err)
					}
				}
			}

			server := &Server{
				config: config.NewConfig("localhost:8080", "", secretsDir, false, "test"),
			}

			err := server.loadSecrets()

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
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
		name        string
		debug       bool
		controllers []controller.IController
	}{
		{
			name:        "debug mode",
			debug:       true,
			controllers: []controller.IController{&mockController{name: "test1"}},
		},
		{
			name:        "release mode",
			debug:       false,
			controllers: []controller.IController{&mockController{name: "test2"}},
		},
		{
			name:  "multiple controllers",
			debug: true,
			controllers: []controller.IController{
				&mockController{name: "test3"},
				&mockController{name: "test4"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := NewServer("localhost", 0, "", "", tt.debug, "test-session")

			err := server.Start(tt.controllers...)
			if err != nil {
				t.Errorf("Start() returned error: %v", err)
			}

			if server.httpServer == nil {
				t.Error("httpServer not initialized")
			}

			// Cleanup
			err = server.Shutdown() // Force cleanup
			if err != nil {
				t.Errorf("Shutdown() returned error during timeout: %v", err)
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

	server := NewServer("localhost", 0, "", "", true, "test-session")
	mockController := &mockController{name: "test"}

	// Start server in goroutine
	errorChan := make(chan error, 1)
	go func() {
		err := server.StartAndWaitForSignal(mockController)
		errorChan <- err
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Send shutdown signal
	if server.shutdownChannel != nil {
		server.shutdownChannel <- os.Interrupt
	} else {
		// Force shutdown if channel not initialized
		err := server.Shutdown() // Force cleanup
		if err != nil {
			t.Errorf("Shutdown() returned error during timeout: %v", err)
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
		err := server.Shutdown() // Force cleanup
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

	server := NewServer("localhost", 0, "", "", false, "test-session")
	mockController := &mockController{name: "test"}

	// Start server
	err = server.Start(mockController)
	if err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}

	// Test shutdown
	err = server.Shutdown()
	if err != nil {
		t.Errorf("Shutdown() returned error: %v", err)
	}

	// Verify controller was closed
	if !mockController.closed {
		t.Error("Controller Close() was not called during shutdown")
	}
}

func TestServer_Bootstrap(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Set up environment
	err := os.Setenv("SESSION_SECRET", "test-secret")
	if err != nil {
		t.Fatalf("Failed to set SESSION_SECRET: %v", err)
	}
	defer func() {
		_ = os.Unsetenv("SESSION_SECRET")
	}()

	tempDir := t.TempDir()
	server := NewServer("localhost", 0, "", tempDir, true, "test-session")

	controllers := []controller.IController{
		&mockController{name: "controller1"},
		&mockController{name: "controller2"},
	}

	err = server.bootstrap(controllers...)
	if err != nil {
		t.Errorf("bootstrap() returned error: %v", err)
	}

	// Verify controllers were bound
	for _, ctrl := range controllers {
		mockCtrl := ctrl.(*mockController)
		if !mockCtrl.bound {
			t.Errorf("Controller %s was not bound", mockCtrl.name)
		}
	}

	// Cleanup
	err = server.Shutdown() // Force cleanup
	if err != nil {
		t.Errorf("Shutdown() returned error during timeout: %v", err)
	}
}

func TestServer_SessionStore(t *testing.T) {
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
		redis string
	}{
		{
			name:  "cookie store",
			redis: "",
		},
		{
			name:  "redis store",
			redis: "localhost:6379",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := NewServer("localhost", 0, tt.redis, "", false, "test-session")
			controller := &sessionTestController{}

			err := server.Start(controller)
			if err != nil {
				t.Errorf("Start() with %s returned error: %v", tt.name, err)
			}

			// Test session functionality
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/session-test", nil)
			server.httpServer.Handler.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Session test failed with status %d", w.Code)
			}

			// Cleanup
			err = server.Shutdown() // Force cleanup
			if err != nil {
				t.Errorf("Shutdown() returned error during timeout: %v", err)
			}
		})
	}
}

func TestAddress(t *testing.T) {
	tests := []struct {
		host     string
		port     int
		expected string
	}{
		{
			host:     "localhost",
			port:     8080,
			expected: "localhost:8080",
		},
		{
			host:     "0.0.0.0",
			port:     3000,
			expected: "0.0.0.0:3000",
		},
		{
			host:     "",
			port:     80,
			expected: ":80",
		},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := address(tt.host, tt.port)
			if result != tt.expected {
				t.Errorf("address(%s, %d) = %s, want %s", tt.host, tt.port, result, tt.expected)
			}
		})
	}
}

// Mock controller for testing
type mockController struct {
	name   string
	bound  bool
	closed bool
}

func (m *mockController) Bind(engine *gin.Engine, cfg config.Config, loginMiddleware gin.HandlerFunc) {
	m.bound = true
	engine.GET("/"+m.name, func(c *gin.Context) {
		c.String(http.StatusOK, "Hello from "+m.name)
	})
}

func (m *mockController) Close() error {
	m.closed = true
	return nil
}

// Session test controller
type sessionTestController struct{}

func (s *sessionTestController) Bind(engine *gin.Engine, cfg config.Config, loginMiddleware gin.HandlerFunc) {
	engine.GET("/session-test", func(c *gin.Context) {
		session := sessions.Default(c)
		session.Set("test_key", "test_value")
		err := session.Save()
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to save session: %v", err)
			return
		}
		c.String(http.StatusOK, "Session set")
	})
}

func (s *sessionTestController) Close() error {
	return nil
}
