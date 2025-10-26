package controller

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/animalet/sargantana-go/config"
	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"
)

func TestNewStaticController(t *testing.T) {
	// Create temporary directories for tests that need them
	tempDir := t.TempDir()
	staticDir := filepath.Join(tempDir, "static")
	err := os.MkdirAll(staticDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create static dir: %v", err)
	}

	// Create a test file
	faviconPath := filepath.Join(staticDir, "favicon.ico")
	err = os.WriteFile(faviconPath, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create favicon.ico: %v", err)
	}

	tests := []struct {
		name          string
		configData    StaticControllerConfig
		expectedError bool
	}{
		{
			name: "valid directory config",
			configData: StaticControllerConfig{
				Path: "/static",
				Dir:  staticDir,
			},
			expectedError: false,
		},
		{
			name: "valid file config",
			configData: StaticControllerConfig{
				Path: "/favicon.ico",
				File: faviconPath,
			},
			expectedError: false,
		},
		{
			name: "missing path",
			configData: StaticControllerConfig{
				Path: "",
				Dir:  staticDir,
			},
			expectedError: true,
		},
		{
			name: "missing dir and file",
			configData: StaticControllerConfig{
				Path: "/static",
			},
			expectedError: true,
		},
		{
			name: "both dir and file set",
			configData: StaticControllerConfig{
				Path: "/static",
				Dir:  staticDir,
				File: faviconPath,
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configBytes, err := yaml.Marshal(tt.configData)
			if err != nil {
				t.Fatalf("Failed to marshal config: %v", err)
			}

			controller, err := NewStaticController(configBytes, config.ServerConfig{})

			if tt.expectedError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if controller == nil {
					t.Error("Expected controller but got nil")
				}

				if controller != nil {
					static := controller.(*static)
					if static.config.Path != tt.configData.Path {
						t.Errorf("Path = %v, want %v", static.config.Path, tt.configData.Path)
					}
				}
			}
		})
	}
}

func TestStatic_BindDirectory(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create temporary directories for testing
	tempDir := t.TempDir()
	staticDir := filepath.Join(tempDir, "static")

	// Create directories
	err := os.MkdirAll(staticDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create static dir: %v", err)
	}

	// Create a static asset
	err = os.WriteFile(filepath.Join(staticDir, "test.css"), []byte("body { color: red; }"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test.css: %v", err)
	}

	configData := StaticControllerConfig{
		Path: "/static",
		Dir:  staticDir,
	}
	configBytes, _ := yaml.Marshal(configData)
	controller, err := NewStaticController(configBytes, config.ServerConfig{})
	if err != nil {
		t.Fatalf("Failed to create static controller: %v", err)
	}

	static := controller.(*static)
	engine := gin.New()
	static.Bind(engine, nil)

	tests := []struct {
		name           string
		path           string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "static asset",
			path:           "/static/test.css",
			expectedStatus: http.StatusOK,
			expectedBody:   "body { color: red; }",
		},
		{
			name:           "non-existent static file",
			path:           "/static/nonexistent.js",
			expectedStatus: http.StatusNotFound,
			expectedBody:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", tt.path, nil)
			engine.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Status = %v, want %v", w.Code, tt.expectedStatus)
			}

			if tt.expectedBody != "" && w.Body.String() != tt.expectedBody {
				t.Errorf("Body = %v, want %v", w.Body.String(), tt.expectedBody)
			}
		})
	}
}

func TestStatic_BindFile(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create temporary directory
	tempDir := t.TempDir()
	staticDir := filepath.Join(tempDir, "static")

	err := os.MkdirAll(staticDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create static dir: %v", err)
	}

	// Create favicon.ico
	faviconContent := []byte("fake favicon")
	faviconPath := filepath.Join(staticDir, "favicon.ico")
	err = os.WriteFile(faviconPath, faviconContent, 0644)
	if err != nil {
		t.Fatalf("Failed to create favicon.ico: %v", err)
	}

	configData := StaticControllerConfig{
		Path: "/favicon.ico",
		File: faviconPath,
	}
	configBytes, _ := yaml.Marshal(configData)
	staticController, err := NewStaticController(configBytes, config.ServerConfig{})
	if err != nil {
		t.Fatalf("Failed to create static controller: %v", err)
	}

	engine := gin.New()
	staticController.Bind(engine, nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/favicon.ico", nil)
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	if w.Body.String() != string(faviconContent) {
		t.Errorf("Body = %v, want %v", w.Body.String(), string(faviconContent))
	}
}

func TestStatic_Close(t *testing.T) {
	tempDir := t.TempDir()
	staticDir := filepath.Join(tempDir, "static")

	err := os.MkdirAll(staticDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create static dir: %v", err)
	}

	configData := StaticControllerConfig{
		Path: "/static",
		Dir:  staticDir,
	}
	configBytes, _ := yaml.Marshal(configData)
	staticController, err := NewStaticController(configBytes, config.ServerConfig{})
	if err != nil {
		t.Fatalf("Failed to create static controller: %v", err)
	}

	err = staticController.Close()
	if err != nil {
		t.Errorf("Close() returned error: %v", err)
	}
}

func TestStatic_MultipleInstances(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create temporary directories
	tempDir := t.TempDir()
	publicDir := filepath.Join(tempDir, "public")
	adminDir := filepath.Join(tempDir, "admin")

	err := os.MkdirAll(publicDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create public dir: %v", err)
	}
	err = os.MkdirAll(adminDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create admin dir: %v", err)
	}

	// Create test files
	err = os.WriteFile(filepath.Join(publicDir, "public.css"), []byte("public styles"), 0644)
	if err != nil {
		t.Fatalf("Failed to create public.css: %v", err)
	}
	err = os.WriteFile(filepath.Join(adminDir, "admin.css"), []byte("admin styles"), 0644)
	if err != nil {
		t.Fatalf("Failed to create admin.css: %v", err)
	}

	// Create two static controller instances
	publicConfig := StaticControllerConfig{
		Path: "/public",
		Dir:  publicDir,
	}
	adminConfig := StaticControllerConfig{
		Path: "/admin",
		Dir:  adminDir,
	}

	publicBytes, _ := yaml.Marshal(publicConfig)
	adminBytes, _ := yaml.Marshal(adminConfig)

	publicController, err := NewStaticController(publicBytes, config.ServerConfig{})
	if err != nil {
		t.Fatalf("Failed to create public controller: %v", err)
	}

	adminController, err := NewStaticController(adminBytes, config.ServerConfig{})
	if err != nil {
		t.Fatalf("Failed to create admin controller: %v", err)
	}

	// Bind both to the same engine
	engine := gin.New()
	publicController.Bind(engine, nil)
	adminController.Bind(engine, nil)

	// Test public path
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/public/public.css", nil)
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Public path status = %v, want %v", w.Code, http.StatusOK)
	}
	if w.Body.String() != "public styles" {
		t.Errorf("Public path body = %v, want 'public styles'", w.Body.String())
	}

	// Test admin path
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/admin/admin.css", nil)
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Admin path status = %v, want %v", w.Code, http.StatusOK)
	}
	if w.Body.String() != "admin styles" {
		t.Errorf("Admin path body = %v, want 'admin styles'", w.Body.String())
	}
}
