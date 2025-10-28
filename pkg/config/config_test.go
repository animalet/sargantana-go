package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/animalet/sargantana-go/pkg/secrets"
	"github.com/pkg/errors"
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
	secrets.Register("env", secrets.NewEnvLoader())
	defer secrets.Unregister("env")

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
	secrets.Register("env", secrets.NewEnvLoader())
	defer secrets.Unregister("env")

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

// TestLoadYaml_InvalidYAML tests ReadServerConfig with malformed YAML
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

// PostgresTestConfig is a test config type for LoadConfig tests
type PostgresTestConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Database string `yaml:"database"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
}

func (p PostgresTestConfig) Validate() error {
	return nil
}

// NonExistentConfig is a test config type for LoadConfig tests
type NonExistentConfig struct {
	Value string `yaml:"value"`
}

func (n NonExistentConfig) Validate() error {
	return nil
}

// TestControllerBinding_Validate tests controller binding validation
func TestControllerBinding_Validate(t *testing.T) {
	tests := []struct {
		name        string
		binding     ControllerBinding
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid binding",
			binding: ControllerBinding{
				TypeName:   "auth",
				Name:       "oauth",
				ConfigData: []byte("key: value"),
			},
			expectError: false,
		},
		{
			name: "missing type",
			binding: ControllerBinding{
				Name:       "oauth",
				ConfigData: []byte("key: value"),
			},
			expectError: true,
			errorMsg:    "controller type must be set and non-empty",
		},
		{
			name: "missing config",
			binding: ControllerBinding{
				TypeName: "auth",
				Name:     "oauth",
			},
			expectError: true,
			errorMsg:    "controller config must be provided",
		},
		{
			name: "valid binding without name",
			binding: ControllerBinding{
				TypeName:   "static",
				ConfigData: []byte("path: /public"),
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.binding.Validate()
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

// TestConfig_Load_ValidatesControllerBindings tests that Load validates all controller bindings
func TestConfig_Load_ValidatesControllerBindings(t *testing.T) {
	// Register env resolver for expansion
	secrets.Register("env", secrets.NewEnvLoader())
	defer secrets.Unregister("env")

	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "test-config.yaml")

	testConfig := `
server:
  address: ":8080"
  session_name: "test-session"
  session_secret: "my-test-secret-key"
controllers:
  - type: "auth"
    name: "oauth"
    config:
      key: "value"
  - type: ""
    name: "invalid"
    config:
      key: "value"
`

	err := os.WriteFile(configFile, []byte(testConfig), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	cfg, err := ReadConfig(configFile)
	if err != nil {
		t.Fatalf("ReadConfig failed: %v", err)
	}

	err = cfg.Load()
	if err == nil {
		t.Fatal("Expected error when loading config with invalid controller binding")
	}
	if !strings.Contains(err.Error(), "controller binding at index 1 is invalid") {
		t.Errorf("Expected error about invalid controller binding at index 1, got: %v", err)
	}
	if !strings.Contains(err.Error(), "controller type must be set and non-empty") {
		t.Errorf("Expected error about missing controller type, got: %v", err)
	}
}

// TestLoadConfig tests the generic LoadConfig function
func TestLoadConfig(t *testing.T) {
	// Register env resolver for expansion
	secrets.Register("env", secrets.NewEnvLoader())
	defer secrets.Unregister("env")

	// Set up test environment variables
	_ = os.Setenv("TEST_HOST", "testhost")
	_ = os.Setenv("TEST_PORT", "5432")
	defer func() {
		_ = os.Unsetenv("TEST_HOST")
		_ = os.Unsetenv("TEST_PORT")
	}()

	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "test-config.yaml")

	testConfig := `
server:
  address: ":8080"
  session_name: "test-session"
  session_secret: "my-test-secret-key"
postgres:
  host: "${TEST_HOST}"
  port: 5432
  database: "testdb"
  user: "testuser"
  password: "testpass"
redis:
  address: "localhost:6379"
  max_idle: 10
  idle_timeout: 240s
`

	err := os.WriteFile(configFile, []byte(testConfig), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	cfg, err := ReadConfig(configFile)
	if err != nil {
		t.Fatalf("ReadConfig failed: %v", err)
	}

	// Test loading postgres config
	postgresConfig, err := LoadConfig[PostgresTestConfig]("postgres", cfg)
	if err != nil {
		t.Fatalf("LoadConfig for postgres failed: %v", err)
	}

	if postgresConfig.Host != "testhost" {
		t.Errorf("Expected Host 'testhost', got '%s'", postgresConfig.Host)
	}
	if postgresConfig.Port != 5432 {
		t.Errorf("Expected Port 5432, got %d", postgresConfig.Port)
	}
	if postgresConfig.Database != "testdb" {
		t.Errorf("Expected Database 'testdb', got '%s'", postgresConfig.Database)
	}

	// Test loading non-existent config
	_, err = LoadConfig[NonExistentConfig]("nonexistent", cfg)
	if err == nil {
		t.Fatal("Expected error when loading non-existent config")
	}
	if !strings.Contains(err.Error(), "no configuration found") {
		t.Errorf("Expected 'no configuration found' error, got: %v", err)
	}
}

// InvalidTestConfig is a test config that always fails validation
type InvalidTestConfig struct {
	Value string `yaml:"value"`
}

func (i InvalidTestConfig) Validate() error {
	return errors.New("always invalid")
}

// TestLoadConfig_ValidationError tests LoadConfig with a config that fails validation
func TestLoadConfig_ValidationError(t *testing.T) {
	// Register env resolver for expansion
	secrets.Register("env", secrets.NewEnvLoader())
	defer secrets.Unregister("env")

	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "test-config.yaml")

	testConfig := `
server:
  address: ":8080"
  session_name: "test-session"
  session_secret: "my-test-secret-key"
invalid_section:
  value: "test"
`

	err := os.WriteFile(configFile, []byte(testConfig), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	cfg, err := ReadConfig(configFile)
	if err != nil {
		t.Fatalf("ReadConfig failed: %v", err)
	}

	// This should fail validation
	_, err = LoadConfig[InvalidTestConfig]("invalid_section", cfg)
	if err == nil {
		t.Fatal("Expected validation error but got none")
	}
	if !strings.Contains(err.Error(), "configuration is invalid") {
		t.Errorf("Expected 'configuration is invalid' error, got: %v", err)
	}
}

// TestUnmarshalTo_ValidationError tests UnmarshalTo with invalid config
func TestUnmarshalTo_ValidationError(t *testing.T) {
	configYAML := `value: "test"`

	var controllerConfig ControllerConfig
	err := yaml.Unmarshal([]byte(configYAML), &controllerConfig)
	if err != nil {
		t.Fatalf("Failed to create ControllerConfig: %v", err)
	}

	_, err = UnmarshalTo[InvalidTestConfig](controllerConfig)
	if err == nil {
		t.Fatal("Expected validation error but got none")
	}
	if !strings.Contains(err.Error(), "configuration is invalid") {
		t.Errorf("Expected 'configuration is invalid' error, got: %v", err)
	}
}

// TestConfig_Validate tests Config validation
func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name        string
		config      Config
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid config",
			config: Config{
				ServerConfig: ServerConfig{
					Address:       ":8080",
					SessionName:   "test",
					SessionSecret: "secret",
				},
				ControllerBindings: ControllerBindings{},
			},
			expectError: false,
		},
		{
			name: "invalid server config",
			config: Config{
				ServerConfig: ServerConfig{
					Address:     ":8080",
					SessionName: "test",
					// Missing SessionSecret
				},
				ControllerBindings: ControllerBindings{},
			},
			expectError: true,
			errorMsg:    "server configuration is invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}
