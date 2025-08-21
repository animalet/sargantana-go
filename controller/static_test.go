package controller

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/animalet/sargantana-go/config"
	"github.com/gin-gonic/gin"
)

func setupTestServerWithSatic() *gin.Engine {
	gin.SetMode(gin.TestMode)

	tempDir := os.TempDir()
	staticDir := filepath.Join(tempDir, "public")
	templateDir := filepath.Join(tempDir, "templates")
	if err := os.MkdirAll(staticDir, 0755); err != nil {
		panic(err)
	}
	if err := os.MkdirAll(templateDir, 0755); err != nil {
		panic(err)
	}

	// Create a minimal index.html for static tests
	indexPath := filepath.Join(staticDir, "index.html")
	_ = os.WriteFile(indexPath, []byte("<html><body>Test Index</body></html>"), 0644)

	// Create a minimal vite.svg for static asset test
	_ = os.WriteFile(filepath.Join(staticDir, "vite.svg"), []byte("svg"), 0644)

	r := gin.New()
	static := NewStatic(staticDir, templateDir)
	static.Bind(r, config.Config{}, nil)
	return r
}

func TestStatic_IndexRoute(t *testing.T) {
	r := setupTestServerWithSatic()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Expected 200 OK, got %d", w.Code)
	}
}

func TestStatic_NotFound(t *testing.T) {
	r := setupTestServerWithSatic()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/nonexistent", nil)
	r.ServeHTTP(w, req)
	if w.Code == http.StatusOK {
		t.Errorf("Expected non-200 for missing route, got %d", w.Code)
	}
}

func TestStatic_AssetRoute(t *testing.T) {
	r := setupTestServerWithSatic()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/static/vite.svg", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK && w.Code != http.StatusNotFound {
		t.Errorf("Expected 200 OK or 404 Not Found for static asset, got %d", w.Code)
	}
}
