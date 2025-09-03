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
	tests := []struct {
		name          string
		configData    StaticControllerConfig
		expectedError bool
	}{
		{
			name: "basic directories",
			configData: StaticControllerConfig{
				StaticsDir:       "./static",
				HtmlTemplatesDir: "./templates",
			},
			expectedError: false,
		},
		{
			name: "absolute paths",
			configData: StaticControllerConfig{
				StaticsDir:       "/var/www/static",
				HtmlTemplatesDir: "/var/www/templates",
			},
			expectedError: false,
		},
		{
			name: "empty paths",
			configData: StaticControllerConfig{
				StaticsDir:       "",
				HtmlTemplatesDir: "",
			},
			expectedError: false,
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

				static := controller.(*static)
				if static.staticsDir != tt.configData.StaticsDir {
					t.Errorf("staticsDir = %v, want %v", static.staticsDir, tt.configData.StaticsDir)
				}
				if static.htmlTemplatesDir != tt.configData.HtmlTemplatesDir {
					t.Errorf("htmlTemplatesDir = %v, want %v", static.htmlTemplatesDir, tt.configData.HtmlTemplatesDir)
				}
			}
		})
	}
}

func TestStatic_Bind(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create temporary directories for testing
	tempDir := t.TempDir()
	staticDir := filepath.Join(tempDir, "static")
	templatesDir := filepath.Join(tempDir, "templates")

	// Create directories
	err := os.MkdirAll(staticDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create static dir: %v", err)
	}

	// Create index.html
	indexContent := "<html><body>Test Page</body></html>"
	err = os.WriteFile(filepath.Join(staticDir, "index.html"), []byte(indexContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create index.html: %v", err)
	}

	// Create a static asset
	err = os.WriteFile(filepath.Join(staticDir, "test.css"), []byte("body { color: red; }"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test.css: %v", err)
	}

	configData := StaticControllerConfig{
		StaticsDir:       staticDir,
		HtmlTemplatesDir: templatesDir,
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
			name:           "index page",
			path:           "/",
			expectedStatus: http.StatusOK,
			expectedBody:   indexContent,
		},
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

func TestStatic_BindWithTemplates(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create temporary directories with templates
	tempDir := t.TempDir()
	staticDir := filepath.Join(tempDir, "static")
	templatesDir := filepath.Join(tempDir, "templates")

	err := os.MkdirAll(staticDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create static dir: %v", err)
	}

	err = os.MkdirAll(templatesDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create templates dir: %v", err)
	}

	// Create index.html
	err = os.WriteFile(filepath.Join(staticDir, "index.html"), []byte("<html><body>Test</body></html>"), 0644)
	if err != nil {
		t.Fatalf("Failed to create index.html: %v", err)
	}

	// Create a template file
	templateContent := `<html><body>Hello {{.name}}</body></html>`
	err = os.WriteFile(filepath.Join(templatesDir, "test.html"), []byte(templateContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create template: %v", err)
	}

	configData := StaticControllerConfig{
		StaticsDir:       staticDir,
		HtmlTemplatesDir: templatesDir,
	}
	configBytes, _ := yaml.Marshal(configData)
	staticController, err := NewStaticController(configBytes, config.ServerConfig{})
	if err != nil {
		t.Fatalf("Failed to create static controller: %v", err)
	}

	engine := gin.New()

	// This should load templates without error
	staticController.Bind(engine, nil)
}

func TestStatic_BindWithEmptyTemplatesDir(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create temporary directories with empty templates
	tempDir := t.TempDir()
	staticDir := filepath.Join(tempDir, "static")
	templatesDir := filepath.Join(tempDir, "templates")

	err := os.MkdirAll(staticDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create static dir: %v", err)
	}

	err = os.MkdirAll(templatesDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create templates dir: %v", err)
	}

	// Create index.html
	err = os.WriteFile(filepath.Join(staticDir, "index.html"), []byte("<html><body>Test</body></html>"), 0644)
	if err != nil {
		t.Fatalf("Failed to create index.html: %v", err)
	}

	configData := StaticControllerConfig{
		StaticsDir:       staticDir,
		HtmlTemplatesDir: templatesDir,
	}
	configBytes, _ := yaml.Marshal(configData)
	staticController, err := NewStaticController(configBytes, config.ServerConfig{})
	if err != nil {
		t.Fatalf("Failed to create static controller: %v", err)
	}

	engine := gin.New()

	// This should handle empty templates directory gracefully
	staticController.Bind(engine, nil)
}

func TestStatic_BindWithNonExistentTemplatesDir(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create temporary directories
	tempDir := t.TempDir()
	staticDir := filepath.Join(tempDir, "static")
	templatesDir := filepath.Join(tempDir, "nonexistent")

	err := os.MkdirAll(staticDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create static dir: %v", err)
	}

	// Create index.html
	err = os.WriteFile(filepath.Join(staticDir, "index.html"), []byte("<html><body>Test</body></html>"), 0644)
	if err != nil {
		t.Fatalf("Failed to create index.html: %v", err)
	}

	configData := StaticControllerConfig{
		StaticsDir:       staticDir,
		HtmlTemplatesDir: templatesDir,
	}
	configBytes, _ := yaml.Marshal(configData)
	staticController, err := NewStaticController(configBytes, config.ServerConfig{})
	if err != nil {
		t.Fatalf("Failed to create static controller: %v", err)
	}

	engine := gin.New()

	// This should handle non-existent templates directory gracefully
	staticController.Bind(engine, nil)
}

func TestStatic_Close(t *testing.T) {
	configData := StaticControllerConfig{
		StaticsDir:       "./static",
		HtmlTemplatesDir: "./templates",
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

func TestStatic_ContentTypeHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tempDir := t.TempDir()
	staticDir := filepath.Join(tempDir, "static")

	err := os.MkdirAll(staticDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create static dir: %v", err)
	}

	// Create index.html
	err = os.WriteFile(filepath.Join(staticDir, "index.html"), []byte("<html><body>Test</body></html>"), 0644)
	if err != nil {
		t.Fatalf("Failed to create index.html: %v", err)
	}

	configData := StaticControllerConfig{
		StaticsDir:       staticDir,
		HtmlTemplatesDir: "",
	}
	configBytes, _ := yaml.Marshal(configData)
	staticController, err := NewStaticController(configBytes, config.ServerConfig{})
	if err != nil {
		t.Fatalf("Failed to create static controller: %v", err)
	}

	engine := gin.New()
	staticController.Bind(engine, nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	engine.ServeHTTP(w, req)

	contentType := w.Header().Get("Content-Type")
	if contentType != "text/html" {
		t.Errorf("Content-Type = %v, want text/html", contentType)
	}
}
