package controller

import (
	"flag"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/animalet/sargantana-go/config"
	"github.com/gin-gonic/gin"
)

func TestNewStatic(t *testing.T) {
	tests := []struct {
		name             string
		staticsDir       string
		htmlTemplatesDir string
	}{
		{
			name:             "basic directories",
			staticsDir:       "./static",
			htmlTemplatesDir: "./templates",
		},
		{
			name:             "absolute paths",
			staticsDir:       "/var/www/static",
			htmlTemplatesDir: "/var/www/templates",
		},
		{
			name:             "empty paths",
			staticsDir:       "",
			htmlTemplatesDir: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			static := NewStatic(tt.staticsDir, tt.htmlTemplatesDir)

			if static == nil {
				t.Fatal("NewStatic returned nil")
			}
			if static.staticsDir != tt.staticsDir {
				t.Errorf("staticsDir = %v, want %v", static.staticsDir, tt.staticsDir)
			}
			if static.htmlTemplatesDir != tt.htmlTemplatesDir {
				t.Errorf("htmlTemplatesDir = %v, want %v", static.htmlTemplatesDir, tt.htmlTemplatesDir)
			}
		})
	}
}

func TestNewStaticFromFlags(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		wantFrontend  string
		wantTemplates string
	}{
		{
			name:          "default values",
			args:          []string{},
			wantFrontend:  "./frontend",
			wantTemplates: "./templates",
		},
		{
			name:          "custom values",
			args:          []string{"-frontend=/custom/static", "-templates=/custom/templates"},
			wantFrontend:  "/custom/static",
			wantTemplates: "/custom/templates",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
			factory := NewStaticFromFlags(flagSet)

			err := flagSet.Parse(tt.args)
			if err != nil {
				t.Fatalf("Failed to parse flags: %v", err)
			}

			controller := factory().(*static)

			if controller.staticsDir != tt.wantFrontend {
				t.Errorf("staticsDir = %v, want %v", controller.staticsDir, tt.wantFrontend)
			}
			if controller.htmlTemplatesDir != tt.wantTemplates {
				t.Errorf("htmlTemplatesDir = %v, want %v", controller.htmlTemplatesDir, tt.wantTemplates)
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

	static := NewStatic(staticDir, templatesDir)
	engine := gin.New()
	cfg := config.NewConfig("localhost:8080", "", "", false, "test-session", "")

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

	static := NewStatic(staticDir, templatesDir)
	engine := gin.New()
	cfg := config.NewConfig("localhost:8080", "", "", false, "test-session", "")

	// This should load templates without error
	static.Bind(engine, nil)
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

	static := NewStatic(staticDir, templatesDir)
	engine := gin.New()
	cfg := config.NewConfig("localhost:8080", "", "", false, "test-session", "")

	// This should handle empty templates directory gracefully
	static.Bind(engine, nil)
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

	static := NewStatic(staticDir, templatesDir)
	engine := gin.New()
	cfg := config.NewConfig("localhost:8080", "", "", false, "test-session", "")

	// This should handle non-existent templates directory gracefully
	static.Bind(engine, nil)
}

func TestStatic_Close(t *testing.T) {
	static := NewStatic("./static", "./templates")
	err := static.Close()
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

	static := NewStatic(staticDir, "")
	engine := gin.New()
	cfg := config.NewConfig("localhost:8080", "", "", false, "test-session", "")

	static.Bind(engine, nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	engine.ServeHTTP(w, req)

	contentType := w.Header().Get("Content-Type")
	if contentType != "text/html" {
		t.Errorf("Content-Type = %v, want text/html", contentType)
	}
}
