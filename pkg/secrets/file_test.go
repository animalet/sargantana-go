package secrets

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestFileResolver_Success tests successful file secret reading
func TestFileResolver_Success(t *testing.T) {
	tempDir := t.TempDir()

	// Create a test secret file
	secretFile := filepath.Join(tempDir, "test-secret")
	secretContent := "my-secret-value\n"
	err := os.WriteFile(secretFile, []byte(secretContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test secret file: %v", err)
	}

	// Create file resolver and test
	fileResolver := NewFileSecretLoader(tempDir)
	result, err := fileResolver.Resolve("test-secret")
	if err != nil {
		t.Fatalf("FileSecretLoader.Resolve failed: %v", err)
	}

	expected := "my-secret-value"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

// TestFileResolver_NoSecretsDir tests file secret reading without configured secrets directory
func TestFileResolver_NoSecretsDir(t *testing.T) {
	// Create file resolver with empty directory
	fileResolver := NewFileSecretLoader("")

	_, err := fileResolver.Resolve("test-secret")
	if err == nil {
		t.Fatal("Expected error when no secrets directory is configured")
	}
	if !strings.Contains(err.Error(), "no secrets directory configured") {
		t.Errorf("Expected 'no secrets directory configured' error, got: %v", err)
	}
}

// TestFileResolver_EmptyFilename tests file secret reading with empty filename
func TestFileResolver_EmptyFilename(t *testing.T) {
	tempDir := t.TempDir()

	// Create file resolver and test with empty filename
	fileResolver := NewFileSecretLoader(tempDir)
	_, err := fileResolver.Resolve("")
	if err == nil {
		t.Fatal("Expected error when filename is empty")
	}
	if !strings.Contains(err.Error(), "no file specified for file secret") {
		t.Errorf("Expected 'no file specified' error, got: %v", err)
	}
}

// TestFileResolver_NonexistentFile tests file secret reading with nonexistent file
func TestFileResolver_NonexistentFile(t *testing.T) {
	tempDir := t.TempDir()

	// Create file resolver and test with nonexistent file
	fileResolver := NewFileSecretLoader(tempDir)
	_, err := fileResolver.Resolve("nonexistent-file")
	if err == nil {
		t.Fatal("Expected error when file doesn't exist")
	}
	if !strings.Contains(err.Error(), "error reading secret file") {
		t.Errorf("Expected 'error reading secret file' error, got: %v", err)
	}
}

// TestFileResolver_WhitespaceTrimming tests that file content is properly trimmed
func TestFileResolver_WhitespaceTrimming(t *testing.T) {
	tempDir := t.TempDir()

	// Create a test secret file with whitespace
	secretFile := filepath.Join(tempDir, "test-secret")
	secretContent := "\n  my-secret-value  \n\t"
	err := os.WriteFile(secretFile, []byte(secretContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test secret file: %v", err)
	}

	fileResolver := NewFileSecretLoader(tempDir)
	result, err := fileResolver.Resolve("test-secret")
	if err != nil {
		t.Fatalf("FileSecretLoader.Resolve failed: %v", err)
	}

	expected := "my-secret-value"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

// TestFileResolver_Name tests the Name method
func TestFileResolver_Name(t *testing.T) {
	resolver := NewFileSecretLoader("/tmp")
	if resolver.Name() != "File" {
		t.Errorf("Expected name 'File', got '%s'", resolver.Name())
	}
}

// TestFileResolverConfig_Validate tests the Validate method
func TestFileResolverConfig_Validate(t *testing.T) {
	// Create a temporary directory for valid tests
	tempDir := t.TempDir()

	// Create a temporary file for testing "not a directory" case
	tempFile := filepath.Join(tempDir, "notadir")
	err := os.WriteFile(tempFile, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	tests := []struct {
		name      string
		config    FileSecretConfig
		wantError bool
		errorMsg  string
	}{
		{
			name: "valid config",
			config: FileSecretConfig{
				SecretsDir: tempDir,
			},
			wantError: false,
		},
		{
			name: "empty secrets dir",
			config: FileSecretConfig{
				SecretsDir: "",
			},
			wantError: true,
			errorMsg:  "secrets_dir is required",
		},
		{
			name: "non-existent directory",
			config: FileSecretConfig{
				SecretsDir: "/path/to/nonexistent/directory",
			},
			wantError: true,
			errorMsg:  "does not exist",
		},
		{
			name: "path is a file not a directory",
			config: FileSecretConfig{
				SecretsDir: tempFile,
			},
			wantError: true,
			errorMsg:  "is not a directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantError {
				if err == nil {
					t.Error("Expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error message to contain '%s', got: %v", tt.errorMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

// TestFileResolverConfig_CreateClient tests the CreateClient method
func TestFileResolverConfig_CreateClient(t *testing.T) {
	// Create a temporary directory for valid tests
	tempDir := t.TempDir()

	tests := []struct {
		name      string
		config    FileSecretConfig
		wantError bool
	}{
		{
			name: "valid config",
			config: FileSecretConfig{
				SecretsDir: tempDir,
			},
			wantError: false,
		},
		{
			name: "invalid config - empty secrets dir",
			config: FileSecretConfig{
				SecretsDir: "",
			},
			wantError: true,
		},
		{
			name: "invalid config - non-existent directory",
			config: FileSecretConfig{
				SecretsDir: "/nonexistent/path/to/secrets",
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver, err := tt.config.CreateClient()
			if tt.wantError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				if resolver != nil {
					t.Error("Expected nil resolver on error")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
				if resolver == nil {
					t.Error("Expected non-nil resolver")
				} else if resolver.secretsDir != tt.config.SecretsDir {
					t.Errorf("Expected secretsDir '%s', got '%s'", tt.config.SecretsDir, resolver.secretsDir)
				}
			}
		})
	}
}
