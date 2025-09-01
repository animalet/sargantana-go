package config

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestLoad(t *testing.T) {
	t.Run("valid config file", func(t *testing.T) {
		// Create a temporary config file
		tempDir := t.TempDir()
		configFile := filepath.Join(tempDir, "test_config.yaml")

		configContent := `
server:
  address: "localhost:8080"
  debug: true
  session_name: "test_session"
  session_secret: "test_secret"
  secrets_dir: "/tmp/secrets"
  redis_session_store: "redis://localhost:6379"
vault:
  address: "https://vault.example.com"
  token: "test_token"
  path: "secret/myapp"
  namespace: "myns"
controllers:
  - type: "auth"
    name: "main_auth"
    config:
      login_path: "/auth/login"
      logout_path: "/auth/logout"
  - type: "static"
    config:
      statics_dir: "./static"
`

		err := os.WriteFile(configFile, []byte(configContent), 0644)
		if err != nil {
			t.Fatalf("Failed to create test config file: %v", err)
		}

		cfg, err := Load(configFile)
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}

		if cfg == nil {
			t.Fatal("Load() returned nil config")
		}

		// Verify server config
		if cfg.ServerConfig.Address != "localhost:8080" {
			t.Errorf("Expected address 'localhost:8080', got '%s'", cfg.ServerConfig.Address)
		}

		if !cfg.ServerConfig.Debug {
			t.Error("Expected debug to be true")
		}

		if cfg.ServerConfig.SessionName != "test_session" {
			t.Errorf("Expected session_name 'test_session', got '%s'", cfg.ServerConfig.SessionName)
		}

		// Verify vault config
		if cfg.Vault.Address != "https://vault.example.com" {
			t.Errorf("Expected vault address 'https://vault.example.com', got '%s'", cfg.Vault.Address)
		}

		// Verify controllers
		if len(cfg.ControllerBindings) != 2 {
			t.Errorf("Expected 2 controller bindings, got %d", len(cfg.ControllerBindings))
		}

		if cfg.ControllerBindings[0].TypeName != "auth" {
			t.Errorf("Expected first controller type 'auth', got '%s'", cfg.ControllerBindings[0].TypeName)
		}

		if cfg.ControllerBindings[0].Name != "main_auth" {
			t.Errorf("Expected first controller name 'main_auth', got '%s'", cfg.ControllerBindings[0].Name)
		}
	})

	t.Run("non-existent file", func(t *testing.T) {
		cfg, err := Load("non-existent-file.yaml")
		if err == nil {
			t.Error("Expected error for non-existent file, got nil")
		}

		if cfg != nil {
			t.Error("Expected nil config for non-existent file")
		}
	})

	t.Run("invalid yaml", func(t *testing.T) {
		tempDir := t.TempDir()
		configFile := filepath.Join(tempDir, "invalid_config.yaml")

		invalidContent := `
server:
  address: "localhost:8080"
  debug: true
invalid_yaml_content: [
`

		err := os.WriteFile(configFile, []byte(invalidContent), 0644)
		if err != nil {
			t.Fatalf("Failed to create test config file: %v", err)
		}

		cfg, err := Load(configFile)
		if err == nil {
			t.Error("Expected error for invalid YAML, got nil")
		}

		if cfg != nil {
			t.Error("Expected nil config for invalid YAML")
		}
	})
}

func TestLoadYaml(t *testing.T) {
	t.Run("valid yaml to struct", func(t *testing.T) {
		tempDir := t.TempDir()
		yamlFile := filepath.Join(tempDir, "test.yaml")

		type TestStruct struct {
			Name  string `yaml:"name"`
			Value int    `yaml:"value"`
		}

		content := `
name: "test"
value: 42
`

		err := os.WriteFile(yamlFile, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test YAML file: %v", err)
		}

		var result TestStruct
		err = LoadYaml(yamlFile, &result)
		if err != nil {
			t.Fatalf("LoadYaml() error = %v", err)
		}

		if result.Name != "test" {
			t.Errorf("Expected name 'test', got '%s'", result.Name)
		}

		if result.Value != 42 {
			t.Errorf("Expected value 42, got %d", result.Value)
		}
	})

	t.Run("non-existent file", func(t *testing.T) {
		var result interface{}
		err := LoadYaml("non-existent.yaml", &result)
		if err == nil {
			t.Error("Expected error for non-existent file, got nil")
		}
	})
}

func TestVaultConfig_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		config   VaultConfig
		expected bool
	}{
		{
			name: "valid config",
			config: VaultConfig{
				Address: "https://vault.example.com",
				Token:   "test_token",
				Path:    "secret/myapp",
			},
			expected: true,
		},
		{
			name: "valid config with namespace",
			config: VaultConfig{
				Address:   "https://vault.example.com",
				Token:     "test_token",
				Path:      "secret/myapp",
				Namespace: "myns",
			},
			expected: true,
		},
		{
			name: "missing address",
			config: VaultConfig{
				Token: "test_token",
				Path:  "secret/myapp",
			},
			expected: false,
		},
		{
			name: "missing token",
			config: VaultConfig{
				Address: "https://vault.example.com",
				Path:    "secret/myapp",
			},
			expected: false,
		},
		{
			name: "missing path",
			config: VaultConfig{
				Address: "https://vault.example.com",
				Token:   "test_token",
			},
			expected: false,
		},
		{
			name:     "empty config",
			config:   VaultConfig{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.IsValid()
			if result != tt.expected {
				t.Errorf("IsValid() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestControllerConfig_UnmarshalYAML(t *testing.T) {
	t.Run("valid yaml node", func(t *testing.T) {
		yamlContent := `
login_path: "/auth/login"
logout_path: "/auth/logout"
callback_path: "/auth/callback"
`

		var node yaml.Node
		err := yaml.Unmarshal([]byte(yamlContent), &node)
		if err != nil {
			t.Fatalf("Failed to unmarshal YAML: %v", err)
		}

		var config ControllerConfig
		err = config.UnmarshalYAML(&node)
		if err != nil {
			t.Fatalf("UnmarshalYAML() error = %v", err)
		}

		if len(config) == 0 {
			t.Error("Expected non-empty ControllerConfig")
		}

		// Verify we can unmarshal the config back
		type AuthConfig struct {
			LoginPath    string `yaml:"login_path"`
			LogoutPath   string `yaml:"logout_path"`
			CallbackPath string `yaml:"callback_path"`
		}

		authConfig, err := UnmarshalTo[AuthConfig](config)
		if err != nil {
			t.Fatalf("Failed to unmarshal back: %v", err)
		}

		if authConfig.LoginPath != "/auth/login" {
			t.Errorf("Expected login_path '/auth/login', got '%s'", authConfig.LoginPath)
		}
	})
}

func TestControllerConfig_To(t *testing.T) {
	t.Run("unmarshal to struct", func(t *testing.T) {
		// Create a ControllerConfig with YAML data
		yamlData := []byte(`
statics_dir: "./static"
templates_dir: "./templates"
`)

		config := ControllerConfig(yamlData)

		type StaticConfig struct {
			StaticsDir   string `yaml:"statics_dir"`
			TemplatesDir string `yaml:"templates_dir"`
		}

		staticConfig, err := UnmarshalTo[StaticConfig](config)
		if err != nil {
			t.Fatalf("To() error = %v", err)
		}

		if staticConfig.StaticsDir != "./static" {
			t.Errorf("Expected statics_dir './static', got '%s'", staticConfig.StaticsDir)
		}

		if staticConfig.TemplatesDir != "./templates" {
			t.Errorf("Expected templates_dir './templates', got '%s'", staticConfig.TemplatesDir)
		}
	})

	t.Run("invalid yaml", func(t *testing.T) {
		invalidYaml := []byte(`invalid: yaml: content: [`)
		c := ControllerConfig(invalidYaml)

		type TestStruct struct {
			Field string `yaml:"field"`
		}

		_, err := UnmarshalTo[TestStruct](c)
		if err == nil {
			t.Error("Expected error for invalid YAML, got nil")
		}
	})

	t.Run("nil pointer", func(t *testing.T) {
		result, err := UnmarshalTo[ControllerConfig](nil)
		if err != nil {
			t.Fatalf("Expected no error for nil pointer, got %v", err)
		}

		if result != nil {
			t.Fatalf("Expected nil result for nil pointer")
		}
	})
}

func TestUnmarshalTo(t *testing.T) {
	t.Run("unmarshal to generic type", func(t *testing.T) {
		yamlData := []byte(`
name: "test"
value: 42
enabled: true
`)

		config := ControllerConfig(yamlData)

		type TestConfig struct {
			Name    string `yaml:"name"`
			Value   int    `yaml:"value"`
			Enabled bool   `yaml:"enabled"`
		}

		result, err := UnmarshalTo[TestConfig](config)
		if err != nil {
			t.Fatalf("UnmarshalTo() error = %v", err)
		}

		if result == nil {
			t.Fatal("UnmarshalTo() returned nil result")
		}

		if result.Name != "test" {
			t.Errorf("Expected name 'test', got '%s'", result.Name)
		}

		if result.Value != 42 {
			t.Errorf("Expected value 42, got %d", result.Value)
		}

		if !result.Enabled {
			t.Error("Expected enabled to be true")
		}
	})

	t.Run("invalid yaml", func(t *testing.T) {
		invalidYaml := []byte(`invalid: yaml: content: [`)
		config := ControllerConfig(invalidYaml)

		type TestConfig struct {
			Field string `yaml:"field"`
		}

		result, err := UnmarshalTo[TestConfig](config)
		if err == nil {
			t.Error("Expected error for invalid YAML, got nil")
		}

		if result != nil {
			t.Error("Expected nil result for invalid YAML")
		}
	})

	t.Run("empty config", func(t *testing.T) {
		config := ControllerConfig([]byte{})

		type TestConfig struct {
			Field string `yaml:"field"`
		}

		result, err := UnmarshalTo[TestConfig](config)
		if err != nil {
			t.Fatalf("UnmarshalTo() error = %v", err)
		}

		if result == nil {
			t.Fatal("UnmarshalTo() returned nil result")
		}

		// Should have zero values
		if result.Field != "" {
			t.Errorf("Expected empty field, got '%s'", result.Field)
		}
	})
}

func TestControllerConfigIntegration(t *testing.T) {
	t.Run("full integration test", func(t *testing.T) {
		// Create a complete config file
		tempDir := t.TempDir()
		configFile := filepath.Join(tempDir, "integration_config.yaml")

		configContent := `
server:
  address: "0.0.0.0:8080"
  debug: false
  session_name: "sargantana_session"
  session_secret: "my_secret_key"
  secrets_dir: "/etc/secrets"
vault:
  address: "https://vault.company.com"
  token: "hvs.secret_token"
  path: "secret/data/myapp"
  namespace: "production"
controllers:
  - type: "auth"
    name: "oauth_controller"
    config:
      callback_path: "/auth/{provider}/callback"
      login_path: "/auth/{provider}"
      logout_path: "/auth/{provider}/logout"
      user_info_path: "/auth/{provider}/user"
      redirect_on_login: "/"
      redirect_on_logout: "/login"
  - type: "static"
    name: "file_server"
    config:
      statics_dir: "./public"
      templates_dir: "./views"
  - type: "load_balancer"
    config:
      auth: true
      path: "/api"
      endpoints:
        - "http://backend1:8001"
        - "http://backend2:8002"
        - "http://backend3:8003"
`

		err := os.WriteFile(configFile, []byte(configContent), 0644)
		if err != nil {
			t.Fatalf("Failed to create test config file: %v", err)
		}

		// Load the config
		cfg, err := Load(configFile)
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}

		// Test server config
		if cfg.ServerConfig.Address != "0.0.0.0:8080" {
			t.Errorf("Expected address '0.0.0.0:8080', got '%s'", cfg.ServerConfig.Address)
		}

		if cfg.ServerConfig.Debug {
			t.Error("Expected debug to be false")
		}

		// Test vault config
		if !cfg.Vault.IsValid() {
			t.Error("Expected vault config to be valid")
		}

		// Test controller configurations
		if len(cfg.ControllerBindings) != 3 {
			t.Fatalf("Expected 3 controller bindings, got %d", len(cfg.ControllerBindings))
		}

		// Test auth controller config
		authBinding := cfg.ControllerBindings[0]
		if authBinding.TypeName != "auth" {
			t.Errorf("Expected auth controller type, got '%s'", authBinding.TypeName)
		}

		type AuthConfig struct {
			CallbackPath     string `yaml:"callback_path"`
			LoginPath        string `yaml:"login_path"`
			LogoutPath       string `yaml:"logout_path"`
			UserInfoPath     string `yaml:"user_info_path"`
			RedirectOnLogin  string `yaml:"redirect_on_login"`
			RedirectOnLogout string `yaml:"redirect_on_logout"`
		}
		authConfig, err := UnmarshalTo[AuthConfig](authBinding.ConfigData)
		if err != nil {
			t.Fatalf("Failed to unmarshal auth config: %v", err)
		}

		if authConfig.CallbackPath != "/auth/{provider}/callback" {
			t.Errorf("Expected callback_path '/auth/{provider}/callback', got '%s'", authConfig.CallbackPath)
		}

		// Test load balancer config
		lbBinding := cfg.ControllerBindings[2]
		if lbBinding.TypeName != "load_balancer" {
			t.Errorf("Expected load_balancer controller type, got '%s'", lbBinding.TypeName)
		}

		type LoadBalancerConfig struct {
			Auth      bool     `yaml:"auth"`
			Path      string   `yaml:"path"`
			Endpoints []string `yaml:"endpoints"`
		}

		lbConfig, err := UnmarshalTo[LoadBalancerConfig](lbBinding.ConfigData)

		if err != nil {
			t.Fatalf("Failed to unmarshal load balancer config: %v", err)
		}

		if !lbConfig.Auth {
			t.Error("Expected load balancer auth to be true")
		}

		if lbConfig.Path != "/api" {
			t.Errorf("Expected load balancer path '/api', got '%s'", lbConfig.Path)
		}

		if len(lbConfig.Endpoints) != 3 {
			t.Errorf("Expected 3 endpoints, got %d", len(lbConfig.Endpoints))
		}

		expectedEndpoints := []string{
			"http://backend1:8001",
			"http://backend2:8002",
			"http://backend3:8003",
		}

		for i, endpoint := range lbConfig.Endpoints {
			if endpoint != expectedEndpoints[i] {
				t.Errorf("Expected endpoint[%d] '%s', got '%s'", i, expectedEndpoints[i], endpoint)
			}
		}
	})
}
