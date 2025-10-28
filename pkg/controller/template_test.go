package controller

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/animalet/sargantana-go/pkg/config"
	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"
)

func TestNewTemplateController(t *testing.T) {
	// Create temporary directories for tests that need them
	tempDir := t.TempDir()
	templatesDir := filepath.Join(tempDir, "templates")
	err := os.MkdirAll(templatesDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create templates dir: %v", err)
	}

	// Create a test file (not a template directory)
	testFile := filepath.Join(tempDir, "test.txt")
	err = os.WriteFile(testFile, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tests := []struct {
		name          string
		configData    TemplateControllerConfig
		expectedError bool
		errorNote     string
	}{
		{
			name: "valid templates directory",
			configData: TemplateControllerConfig{
				Path: templatesDir,
			},
			expectedError: false,
		},
		{
			name: "missing path",
			configData: TemplateControllerConfig{
				Path: "",
			},
			expectedError: true,
		},
		{
			name: "non-existent directory",
			configData: TemplateControllerConfig{
				Path: filepath.Join(tempDir, "nonexistent"),
			},
			expectedError: true,
		},
		{
			name: "file instead of directory",
			configData: TemplateControllerConfig{
				Path: testFile,
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.errorNote != "" {
				t.Logf("Note: %s", tt.errorNote)
			}
			configBytes, err := yaml.Marshal(tt.configData)
			if err != nil {
				t.Fatalf("Failed to marshal config: %v", err)
			}

			controller, err := NewTemplateController(configBytes, ControllerContext{ServerConfig: config.ServerConfig{}})

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
					tmpl := controller.(*template)
					if tmpl.path != tt.configData.Path {
						t.Errorf("Path = %v, want %v", tmpl.path, tt.configData.Path)
					}
				}
			}
		})
	}
}

func TestTemplateController_BindWithTemplates(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create temporary directory with template files
	tempDir := t.TempDir()
	templatesDir := filepath.Join(tempDir, "templates")
	err := os.MkdirAll(templatesDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create templates dir: %v", err)
	}

	// Create a simple template file
	templateContent := `<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body>Hello World</body>
</html>`
	err = os.WriteFile(filepath.Join(templatesDir, "index.html"), []byte(templateContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create template file: %v", err)
	}

	configData := TemplateControllerConfig{
		Path: templatesDir,
	}
	configBytes, _ := yaml.Marshal(configData)
	controller, err := NewTemplateController(configBytes, ControllerContext{ServerConfig: config.ServerConfig{}})
	if err != nil {
		t.Fatalf("Failed to create template controller: %v", err)
	}

	tmpl := controller.(*template)
	engine := gin.New()

	// Bind should not panic and should load templates
	tmpl.Bind(engine, nil)

	// Verify templates were loaded by checking if the HTML render is set
	if engine.HTMLRender == nil {
		t.Error("Expected HTML renderer to be set after Bind")
	}
}

func TestTemplateController_BindWithMultipleTemplates(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create temporary directory with multiple template files
	tempDir := t.TempDir()
	templatesDir := filepath.Join(tempDir, "templates")
	err := os.MkdirAll(templatesDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create templates dir: %v", err)
	}

	// Create multiple template files
	err = os.WriteFile(filepath.Join(templatesDir, "index.html"), []byte("<html><body>Index</body></html>"), 0644)
	if err != nil {
		t.Fatalf("Failed to create index template: %v", err)
	}

	err = os.WriteFile(filepath.Join(templatesDir, "about.html"), []byte("<html><body>About</body></html>"), 0644)
	if err != nil {
		t.Fatalf("Failed to create about template: %v", err)
	}

	configData := TemplateControllerConfig{
		Path: templatesDir,
	}
	configBytes, _ := yaml.Marshal(configData)
	controller, err := NewTemplateController(configBytes, ControllerContext{ServerConfig: config.ServerConfig{}})
	if err != nil {
		t.Fatalf("Failed to create template controller: %v", err)
	}

	tmpl := controller.(*template)
	engine := gin.New()

	// Should load all templates
	tmpl.Bind(engine, nil)

	if engine.HTMLRender == nil {
		t.Error("Expected HTML renderer to be set after Bind with multiple templates")
	}
}

func TestTemplateController_BindEmptyDirectory(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create empty templates directory
	tempDir := t.TempDir()
	templatesDir := filepath.Join(tempDir, "templates")
	err := os.MkdirAll(templatesDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create templates dir: %v", err)
	}

	configData := TemplateControllerConfig{
		Path: templatesDir,
	}
	configBytes, _ := yaml.Marshal(configData)
	controller, err := NewTemplateController(configBytes, ControllerContext{ServerConfig: config.ServerConfig{}})
	if err != nil {
		t.Fatalf("Failed to create template controller: %v", err)
	}

	tmpl := controller.(*template)
	engine := gin.New()

	// Bind should not panic and should log warning but not load templates
	tmpl.Bind(engine, nil)

	// HTML renderer should not be set for empty directory
	if engine.HTMLRender != nil {
		t.Error("Expected HTML renderer to not be set for empty directory")
	}
}

func TestTemplateController_BindNonExistentDirectory(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tempDir := t.TempDir()
	nonExistentDir := filepath.Join(tempDir, "nonexistent")

	// Create controller with path but directory doesn't exist yet
	// This should pass validation during NewTemplateController since we validate first
	// but fail during Bind if directory is deleted after creation

	// For this test, we'll create the directory first to pass NewTemplateController
	err := os.MkdirAll(nonExistentDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	configData := TemplateControllerConfig{
		Path: nonExistentDir,
	}
	configBytes, _ := yaml.Marshal(configData)
	controller, err := NewTemplateController(configBytes, ControllerContext{ServerConfig: config.ServerConfig{}})
	if err != nil {
		t.Fatalf("Failed to create template controller: %v", err)
	}

	// Now remove the directory
	err = os.RemoveAll(nonExistentDir)
	if err != nil {
		t.Fatalf("Failed to remove directory: %v", err)
	}

	tmpl := controller.(*template)
	engine := gin.New()

	// Bind should not panic when directory doesn't exist (it checks and skips)
	tmpl.Bind(engine, nil)

	// HTML renderer should not be set
	if engine.HTMLRender != nil {
		t.Error("Expected HTML renderer to not be set for non-existent directory")
	}
}

func TestTemplateController_Close(t *testing.T) {
	tempDir := t.TempDir()
	templatesDir := filepath.Join(tempDir, "templates")

	err := os.MkdirAll(templatesDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create templates dir: %v", err)
	}

	configData := TemplateControllerConfig{
		Path: templatesDir,
	}
	configBytes, _ := yaml.Marshal(configData)
	controller, err := NewTemplateController(configBytes, ControllerContext{ServerConfig: config.ServerConfig{}})
	if err != nil {
		t.Fatalf("Failed to create template controller: %v", err)
	}

	err = controller.Close()
	if err != nil {
		t.Errorf("Close() returned error: %v", err)
	}
}

func TestTemplateControllerConfig_Validate(t *testing.T) {
	tempDir := t.TempDir()
	validDir := filepath.Join(tempDir, "valid")
	err := os.MkdirAll(validDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create valid dir: %v", err)
	}

	testFile := filepath.Join(tempDir, "file.txt")
	err = os.WriteFile(testFile, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tests := []struct {
		name          string
		config        TemplateControllerConfig
		expectedError bool
		errorNote     string
	}{
		{
			name: "valid directory",
			config: TemplateControllerConfig{
				Path: validDir,
			},
			expectedError: false,
		},
		{
			name: "empty path",
			config: TemplateControllerConfig{
				Path: "",
			},
			expectedError: true,
		},
		{
			name: "non-existent path",
			config: TemplateControllerConfig{
				Path: filepath.Join(tempDir, "nonexistent"),
			},
			expectedError: true,
		},
		{
			name: "file instead of directory",
			config: TemplateControllerConfig{
				Path: testFile,
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.errorNote != "" {
				t.Logf("Note: %s", tt.errorNote)
			}
			err := tt.config.Validate()
			if tt.expectedError {
				if err == nil {
					t.Error("Expected validation error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected validation error: %v", err)
				}
			}
		})
	}
}
