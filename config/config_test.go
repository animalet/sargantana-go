package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// TestLoadYaml tests loading YAML configuration from file
func TestLoadYaml(t *testing.T) {
	// Create a temporary YAML file for testing
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "test-config.yaml")

	testConfig := `
server:
  address: ":8080"
  redis_session_store:
    address: "localhost:6379"
    max_idle: 10
    idle_timeout: 240s
  session_name: "test-session"
  session_secret: "test-secret-key"
  debug: true
`

	err := os.WriteFile(configFile, []byte(testConfig), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	cfg, err := ReadConfig(configFile)
	if err != nil {
		t.Fatalf("LoadYaml failed: %v", err)
	}

	if cfg.ServerConfig.Address != ":8080" {
		t.Errorf("Expected address ':8080', got '%s'", cfg.ServerConfig.Address)
	}
	if cfg.ServerConfig.SessionSecret != "test-secret-key" {
		t.Errorf("Expected session secret 'test-secret-key', got '%s'", cfg.ServerConfig.SessionSecret)
	}
}

// TestLoadYaml_FileNotFound tests error handling when config file doesn't exist
func TestLoadYaml_FileNotFound(t *testing.T) {
	_, err := ReadConfig("nonexistent-file.yaml")
	if err == nil {
		t.Fatal("Expected error when loading nonexistent file")
	}
}

// TestLoad_MissingSessionSecret tests that missing session secret causes error
func TestLoad_MissingSessionSecret(t *testing.T) {
	// Register env resolver for expansion
	RegisterPropertyResolver("env", NewEnvResolver())
	defer UnregisterPropertyResolver("env")

	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "test-config.yaml")

	testConfig := `
server:
  address: ":8080"
  redis_session_store:
    address: "localhost:6379"
    max_idle: 10
    idle_timeout: 240s
  session_name: "test-session"
  # session_secret is missing
`

	err := os.WriteFile(configFile, []byte(testConfig), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	cfg, err := ReadConfig(configFile)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	err = cfg.Load()
	if err == nil || !strings.HasSuffix(err.Error(), "session_secret must be set and non-empty") {
		t.Errorf("Expected specific error message, got: %v", err)
	}
}

// TestVaultConfig_IsValid tests the VaultConfig validation
func TestVaultConfig_IsValid(t *testing.T) {
	tests := []struct {
		name          string
		config        *VaultConfig
		errorExpected bool
	}{
		{
			name: "valid config",
			config: &VaultConfig{
				Address: "http://localhost:8200",
				Token:   "test-token",
				Path:    "secret/data/test",
			},
			errorExpected: false,
		},
		{
			name: "missing address",
			config: &VaultConfig{
				Address: "",
				Token:   "test-token",
				Path:    "secret/data/test",
			},
			errorExpected: true,
		},
		{
			name: "missing token",
			config: &VaultConfig{
				Address: "http://localhost:8200",
				Token:   "",
				Path:    "secret/data/test",
			},
			errorExpected: true,
		},
		{
			name: "missing path",
			config: &VaultConfig{
				Address: "http://localhost:8200",
				Token:   "test-token",
				Path:    "",
			},
			errorExpected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if err != nil && !tt.errorExpected {
				t.Errorf("Validate() = %v, errorExpected %v", err, tt.errorExpected)
			}
		})
	}
}

// TestControllerConfig_UnmarshalYAML tests ControllerConfig YAML unmarshaling
func TestControllerConfig_UnmarshalYAML(t *testing.T) {
	yamlData := `
controllers:
  - type: "TestController"
    name: "test"
    config:
      key1: "value1"
      key2: 42
      nested:
        subkey: "subvalue"
`

	var config Config
	err := yaml.Unmarshal([]byte(yamlData), &config)
	if err != nil {
		t.Fatalf("Failed to unmarshal YAML: %v", err)
	}

	if len(config.ControllerBindings) != 1 {
		t.Fatalf("Expected 1 controller binding, got %d", len(config.ControllerBindings))
	}

	binding := config.ControllerBindings[0]
	if binding.TypeName != "TestController" {
		t.Errorf("Expected TypeName 'TestController', got '%s'", binding.TypeName)
	}
	if binding.Name != "test" {
		t.Errorf("Expected Name 'test', got '%s'", binding.Name)
	}

	// Test that ConfigData contains the raw YAML
	if len(binding.ConfigData) == 0 {
		t.Error("Expected ConfigData to contain YAML data")
	}
}

// TestUnmarshalTo_Error tests error handling in UnmarshalTo function
type TestConfig struct {
	Key1         string   `yaml:"key1"`
	Key2         int      `yaml:"key2"`
	Key          string   `yaml:"key"`
	EnvVar       string   `yaml:"env_var"`
	InvalidField chan int `yaml:"invalid_field"` // channels can't be marshaled/unmarshaled
}

// TestUnmarshalTo tests the generic unmarshaling function
func TestUnmarshalTo(t *testing.T) {
	// Register env resolver for expansion
	RegisterPropertyResolver("env", NewEnvResolver())
	defer UnregisterPropertyResolver("env")

	// Set up environment variable for testing
	_ = os.Setenv("TEST_ENV_VAR", "test-value")
	defer func() { _ = os.Unsetenv("TEST_ENV_VAR") }()

	configYAML := `
key1: "value1"
key2: 42
env_var: "${TEST_ENV_VAR}"
`

	var controllerConfig ControllerConfig
	err := yaml.Unmarshal([]byte(configYAML), &controllerConfig)
	if err != nil {
		t.Fatalf("Failed to create ControllerConfig: %v", err)
	}

	result, err := UnmarshalTo[TestConfig](controllerConfig)
	if err != nil {
		t.Fatalf("UnmarshalTo failed: %v", err)
	}

	if result.Key1 != "value1" {
		t.Errorf("Expected Key1 'value1', got '%s'", result.Key1)
	}
	if result.Key2 != 42 {
		t.Errorf("Expected Key2 42, got %d", result.Key2)
	}
	if result.EnvVar != "test-value" {
		t.Errorf("Expected EnvVar 'test-value', got '%s'", result.EnvVar)
	}
}

// TestUnmarshalTo_NilConfig tests UnmarshalTo with nil config
func TestUnmarshalTo_NilConfig(t *testing.T) {
	result, err := UnmarshalTo[TestConfig](nil)
	if err != nil {
		t.Fatalf("UnmarshalTo with nil config failed: %v", err)
	}
	if result != nil {
		t.Error("Expected nil result for nil config")
	}
}

// TestExpandVariables tests environment variable expansion
func TestExpandVariables(t *testing.T) {
	// Register env resolver for expansion
	RegisterPropertyResolver("env", NewEnvResolver())
	defer UnregisterPropertyResolver("env")

	// Set up test environment variables
	_ = os.Setenv("TEST_VAR", "test-value")
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
  redis_session_store:
    address: "localhost:${TEST_NUMBER}"
    max_idle: 10
    idle_timeout: 240s
  session_name: "test-session"
  session_secret: "my-test-secret-key"
vault:
  address: "http://localhost:8200"
  token: "${TEST_VAR}"
  path: "secret/data/test"
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
	if cfg.ServerConfig.Address != "test-value:8080" {
		t.Errorf("Expected address 'test-value:8080', got '%s'", cfg.ServerConfig.Address)
	}
	if cfg.ServerConfig.RedisSessionStore == nil {
		t.Fatal("Expected RedisSessionStore to be configured")
	}
	if cfg.ServerConfig.RedisSessionStore.Address != "localhost:42" {
		t.Errorf("Expected redis store address 'localhost:42', got '%s'", cfg.ServerConfig.RedisSessionStore.Address)
	}
	if cfg.Vault.Token != "test-value" {
		t.Errorf("Expected Vault token 'test-value', got '%s'", cfg.Vault.Token)
	}
}

// TestLoad_VaultCreationError tests error handling when Vault client creation fails
func TestLoad_VaultCreationError(t *testing.T) {
	vaultCfg := &VaultConfig{
		Address: "://invalid-malformed-url",
		Token:   "test-token",
		Path:    "secret/data/test",
	}

	_, err := CreateVaultClient(vaultCfg)
	if err == nil {
		t.Fatal("Expected error when creating Vault client with invalid address")
	}
}

// TestControllerConfig_UnmarshalYAML_Error tests error handling in YAML unmarshaling
func TestControllerConfig_UnmarshalYAML_Error(t *testing.T) {
	// Create an invalid yaml.Node that will cause Marshal to fail
	var config ControllerConfig

	// Create a yaml.Node with invalid content that will cause marshaling to fail
	node := &yaml.Node{
		Kind:  yaml.ScalarNode,
		Tag:   "!!invalid",
		Value: string([]byte{0xFF, 0xFE}), // Invalid UTF-8 sequence
	}

	err := config.UnmarshalYAML(node)
	if err == nil {
		t.Fatal("Expected error when marshaling invalid YAML node")
	}
}

func (t TestConfig) Validate() error {
	return nil
}
func TestUnmarshalTo_Error(t *testing.T) {

	// Create invalid YAML that will cause unmarshaling to fail
	invalidYAML := []byte("invalid_field: this_will_fail_to_unmarshal_to_channel")
	controllerConfig := ControllerConfig(invalidYAML)

	_, err := UnmarshalTo[TestConfig](controllerConfig)
	if err == nil {
		t.Fatal("Expected error when unmarshaling invalid YAML to struct with channel field")
	}
}

// TestExpandVariables_ComplexStructures tests expandVariables with different data types
func TestExpandVariables_ComplexStructures(t *testing.T) {
	// Register env resolver for expansion
	RegisterPropertyResolver("env", NewEnvResolver())
	defer UnregisterPropertyResolver("env")

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
	// Test with no secrets directory configured - unregister file resolver
	originalResolver := globalResolverRegistry.GetResolver("file")
	UnregisterPropertyResolver("file")
	defer func() {
		if originalResolver != nil {
			RegisterPropertyResolver("file", originalResolver)
		}
	}()

	defer func() {
		if r := recover(); r != nil {
			if !strings.Contains(r.(error).Error(), "no resolver registered") {
				t.Errorf("Expected 'error retrieving secret from file' panic, got: %v", r)
			}
		} else {
			t.Fatal("Expected panic when no secrets directory is configured")
		}
	}()

	expand("file:test-file")
}

// TestLoadYaml_InvalidYAML tests ReadConfig with malformed YAML
func TestLoadYaml_InvalidYAML(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "invalid-config.yaml")

	// Create malformed YAML
	invalidYAML := `
server:
  address: ":8080"
  invalid: [unclosed array
`

	err := os.WriteFile(configFile, []byte(invalidYAML), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	_, err = ReadConfig(configFile)
	if err == nil {
		t.Fatal("Expected error when loading malformed YAML")
	}
}
