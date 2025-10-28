package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/animalet/sargantana-go/pkg/secrets"
)

// RedisExpandTestConfig is a test config type for variable expansion tests
type RedisExpandTestConfig struct {
	Address     string `yaml:"address"`
	MaxIdle     int    `yaml:"max_idle"`
	IdleTimeout string `yaml:"idle_timeout"`
}

func (r RedisExpandTestConfig) Validate() error {
	return nil
}

// TestExpandVariables tests environment variable expansion
func TestExpandVariables(t *testing.T) {
	// Register env resolver for expansion
	secrets.Register("env", secrets.NewEnvLoader())
	defer secrets.Unregister("env")

	// Set up test environment variables
	_ = os.Setenv("TEST_VAR", "localhost")
	_ = os.Setenv("TEST_NUMBER", "42")
	defer func() {
		_ = os.Unsetenv("TEST_VAR")
		_ = os.Unsetenv("TEST_NUMBER")
	}()

	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "test-config.yaml")

	testConfig := `
server:
  address: "${TEST_VAR}:8080"
  session_name: "test-session"
  session_secret: "my-test-secret-key"
redis:
  address: "localhost:${TEST_NUMBER}"
  max_idle: 10
  idle_timeout: 240s
`

	err := os.WriteFile(configFile, []byte(testConfig), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	cfg, err := ReadConfig(configFile)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if err = cfg.Load(); err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.ServerConfig.Address != "localhost:8080" {
		t.Errorf("Expected address 'localhost:8080', got '%s'", cfg.ServerConfig.Address)
	}

	// Test that Redis config can be loaded separately
	redisCfg, err := LoadConfig[RedisExpandTestConfig]("redis", cfg)
	if err != nil {
		t.Fatalf("Failed to load Redis config: %v", err)
	}
	if redisCfg.Address != "localhost:42" {
		t.Errorf("Expected redis address 'localhost:42', got '%s'", redisCfg.Address)
	}
}

// TestExpandVariables_ComplexStructures tests expandVariables with different data types
func TestExpandVariables_ComplexStructures(t *testing.T) {
	// Register env resolver for expansion
	secrets.Register("env", secrets.NewEnvLoader())
	defer secrets.Unregister("env")

	_ = os.Setenv("TEST_EXPAND", "expanded_value")
	defer func() { _ = os.Unsetenv("TEST_EXPAND") }()

	// Test with slice of strings
	type ConfigWithSlice struct {
		Items []string `yaml:"items"`
	}

	sliceConfig := ConfigWithSlice{
		Items: []string{"${TEST_EXPAND}_1", "${TEST_EXPAND}_2"},
	}

	expandVariables(reflect.ValueOf(&sliceConfig).Elem())

	if sliceConfig.Items[0] != "expanded_value_1" {
		t.Errorf("Expected 'expanded_value_1', got '%s'", sliceConfig.Items[0])
	}
	if sliceConfig.Items[1] != "expanded_value_2" {
		t.Errorf("Expected 'expanded_value_2', got '%s'", sliceConfig.Items[1])
	}

	// Test with map
	type ConfigWithMap struct {
		Data map[string]string `yaml:"data"`
	}

	mapConfig := ConfigWithMap{
		Data: map[string]string{
			"key1": "${TEST_EXPAND}_map1",
			"key2": "${TEST_EXPAND}_map2",
		},
	}

	expandVariables(reflect.ValueOf(&mapConfig).Elem())

	if mapConfig.Data["key1"] != "expanded_value_map1" {
		t.Errorf("Expected 'expanded_value_map1', got '%s'", mapConfig.Data["key1"])
	}
	if mapConfig.Data["key2"] != "expanded_value_map2" {
		t.Errorf("Expected 'expanded_value_map2', got '%s'", mapConfig.Data["key2"])
	}

	// Test with pointer to struct
	type InnerConfig struct {
		Value string `yaml:"value"`
	}
	type ConfigWithPointer struct {
		Inner *InnerConfig `yaml:"inner"`
	}

	ptrConfig := ConfigWithPointer{
		Inner: &InnerConfig{Value: "${TEST_EXPAND}_ptr"},
	}

	expandVariables(reflect.ValueOf(&ptrConfig).Elem())

	if ptrConfig.Inner.Value != "expanded_value_ptr" {
		t.Errorf("Expected 'expanded_value_ptr', got '%s'", ptrConfig.Inner.Value)
	}

	// Test with nil pointer (should not panic)
	nilPtrConfig := ConfigWithPointer{Inner: nil}
	expandVariables(reflect.ValueOf(&nilPtrConfig).Elem())
	// Should complete without error

	// Test with unsupported type (should return without error)
	type ConfigWithChannel struct {
		Ch chan int
	}

	chanConfig := ConfigWithChannel{Ch: make(chan int)}
	expandVariables(reflect.ValueOf(&chanConfig).Elem())
	// Should complete without error
}

// TestExpand_FilePrefix_Error tests file prefix expansion error handling
func TestExpand_FilePrefix_Error(t *testing.T) {
	// Test with no secrets directory configured - unregister file provider
	originalResolver := secrets.GetResolver("file")
	secrets.Unregister("file")
	defer func() {
		if originalResolver != nil {
			secrets.Register("file", originalResolver)
		}
	}()

	defer func() {
		if r := recover(); r != nil {
			if !strings.Contains(r.(error).Error(), "no secret provider registered") {
				t.Errorf("Expected 'error retrieving secret from file' panic, got: %v", r)
			}
		} else {
			t.Fatal("Expected panic when no secrets directory is configured")
		}
	}()

	expand("file:test-file")
}
