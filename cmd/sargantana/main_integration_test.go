//go:build integration

package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/animalet/sargantana-go/pkg/config/secrets"
	"github.com/animalet/sargantana-go/pkg/database"
	"github.com/animalet/sargantana-go/pkg/server"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v3"
)

var _ = Describe("Sargantana Binary Integration Tests", func() {
	var (
		tmpDir string
	)

	BeforeEach(func() {
		var err error
		tmpDir, err = os.MkdirTemp("", "sargantana-test-*")
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		if tmpDir != "" {
			_ = os.RemoveAll(tmpDir)
		}
	})

	Context("CLI Arguments", func() {
		It("should show help message", func() {
			// Capture stdout
			exitCode := runWithArgs([]string{"--help"})
			Expect(exitCode).To(Equal(exitSuccess))
		})

		It("should show version", func() {
			exitCode := runWithArgs([]string{"--version"})
			Expect(exitCode).To(Equal(exitSuccess))
		})

		It("should fail when config is not provided", func() {
			exitCode := runWithArgs([]string{})
			Expect(exitCode).To(Equal(exitError))
		})

		It("should fail when config file does not exist", func() {
			exitCode := runWithArgs([]string{"--config", "/nonexistent/config.yaml"})
			Expect(exitCode).To(Equal(exitError))
		})
	})

	Context("Basic Server Configuration", func() {
		It("should start with minimal configuration and cookie sessions", func() {
			port := findFreePort()
			configPath := filepath.Join(tmpDir, "config.yaml")
			writeBasicConfig(configPath, port)

			stopServer, errChan := startServerInBackground(configPath)
			defer func() { _ = stopServer() }()

			// Wait for server to be ready
			Eventually(func() error {
				select {
				case err := <-errChan:
					return fmt.Errorf("server error: %w", err)
				default:
				}
				resp, err := http.Get(fmt.Sprintf("http://localhost:%d/health", port))
				if err != nil {
					return err
				}
				_ = resp.Body.Close()
				return nil
			}, 10*time.Second, 500*time.Millisecond).Should(Succeed())

			// Check for startup errors
			select {
			case err := <-errChan:
				Fail(fmt.Sprintf("Server failed to start: %v", err))
			default:
			}

			// Verify server responds
			resp, err := http.Get(fmt.Sprintf("http://localhost:%d/health", port))
			Expect(err).NotTo(HaveOccurred())
			defer func() { _ = resp.Body.Close() }()
			Expect(resp.StatusCode).To(BeNumerically(">=", 200))
		})
	})

	Context("Redis Session Store", func() {
		It("should start with Redis session configuration", func() {
			port := findFreePort()
			configPath := filepath.Join(tmpDir, "config-redis.yaml")
			writeRedisConfig(configPath, port)

			stopServer, errChan := startServerInBackground(configPath)
			defer func() { _ = stopServer() }()

			// Wait for server to be ready
			Eventually(func() error {
				select {
				case err := <-errChan:
					return fmt.Errorf("server error: %w", err)
				default:
				}
				resp, err := http.Get(fmt.Sprintf("http://localhost:%d/health", port))
				if err != nil {
					return err
				}
				_ = resp.Body.Close()
				return nil
			}, 10*time.Second, 500*time.Millisecond).Should(Succeed())
		})
	})

	Context("MongoDB Session Store", func() {
		It("should start with MongoDB session configuration", func() {
			port := findFreePort()
			configPath := filepath.Join(tmpDir, "config-mongodb.yaml")
			writeMongoDBConfig(configPath, port)

			stopServer, errChan := startServerInBackground(configPath)
			defer func() { _ = stopServer() }()

			// Wait for server to be ready
			Eventually(func() error {
				select {
				case err := <-errChan:
					return fmt.Errorf("server error: %w", err)
				default:
				}
				resp, err := http.Get(fmt.Sprintf("http://localhost:%d/health", port))
				if err != nil {
					return err
				}
				_ = resp.Body.Close()
				return nil
			}, 10*time.Second, 500*time.Millisecond).Should(Succeed())
		})
	})

	Context("PostgreSQL Session Store", func() {
		It("should start with PostgreSQL session configuration", func() {
			port := findFreePort()
			configPath := filepath.Join(tmpDir, "config-postgres.yaml")
			writePostgresConfig(configPath, port)

			stopServer, errChan := startServerInBackground(configPath)
			defer func() { _ = stopServer() }()

			// Wait for server to be ready
			Eventually(func() error {
				select {
				case err := <-errChan:
					return fmt.Errorf("server error: %w", err)
				default:
				}
				resp, err := http.Get(fmt.Sprintf("http://localhost:%d/health", port))
				if err != nil {
					return err
				}
				_ = resp.Body.Close()
				return nil
			}, 10*time.Second, 500*time.Millisecond).Should(Succeed())
		})
	})

	Context("Memcached Session Store", func() {
		It("should start with Memcached session configuration", func() {
			port := findFreePort()
			configPath := filepath.Join(tmpDir, "config-memcached.yaml")
			writeMemcachedConfig(configPath, port)

			stopServer, errChan := startServerInBackground(configPath)
			defer func() { _ = stopServer() }()

			// Wait for server to be ready
			Eventually(func() error {
				select {
				case err := <-errChan:
					return fmt.Errorf("server error: %w", err)
				default:
				}
				resp, err := http.Get(fmt.Sprintf("http://localhost:%d/health", port))
				if err != nil {
					return err
				}
				_ = resp.Body.Close()
				return nil
			}, 10*time.Second, 500*time.Millisecond).Should(Succeed())
		})
	})

	Context("Vault Secret Provider", func() {
		It("should start with Vault configuration", func() {
			port := findFreePort()
			configPath := filepath.Join(tmpDir, "config-vault.yaml")
			writeVaultConfig(configPath, port)

			stopServer, errChan := startServerInBackground(configPath)
			defer func() { _ = stopServer() }()

			// Wait for server to be ready
			Eventually(func() error {
				select {
				case err := <-errChan:
					return fmt.Errorf("server error: %w", err)
				default:
				}
				resp, err := http.Get(fmt.Sprintf("http://localhost:%d/health", port))
				if err != nil {
					return err
				}
				_ = resp.Body.Close()
				return nil
			}, 10*time.Second, 500*time.Millisecond).Should(Succeed())
		})
	})

	Context("AWS Secrets Manager Provider", func() {
		It("should start with AWS Secrets Manager configuration (LocalStack)", func() {
			port := findFreePort()
			configPath := filepath.Join(tmpDir, "config-aws.yaml")
			writeAWSConfig(configPath, port)

			stopServer, errChan := startServerInBackground(configPath)
			defer func() { _ = stopServer() }()

			// Wait for server to be ready
			Eventually(func() error {
				select {
				case err := <-errChan:
					return fmt.Errorf("server error: %w", err)
				default:
				}
				resp, err := http.Get(fmt.Sprintf("http://localhost:%d/health", port))
				if err != nil {
					return err
				}
				_ = resp.Body.Close()
				return nil
			}, 10*time.Second, 500*time.Millisecond).Should(Succeed())
		})
	})

	Context("All Controllers", func() {
		It("should configure all controller types", func() {
			port := findFreePort()
			configPath := filepath.Join(tmpDir, "config-controllers.yaml")
			writeAllControllersConfig(configPath, port)

			stopServer, errChan := startServerInBackground(configPath)
			defer func() { _ = stopServer() }()

			// Wait for server to be ready
			Eventually(func() error {
				select {
				case err := <-errChan:
					return fmt.Errorf("server error: %w", err)
				default:
				}
				resp, err := http.Get(fmt.Sprintf("http://localhost:%d/health", port))
				if err != nil {
					return err
				}
				_ = resp.Body.Close()
				return nil
			}, 10*time.Second, 500*time.Millisecond).Should(Succeed())
		})
	})
})

// Helper functions

// startServerInBackground starts the server in a goroutine and returns a function to stop it
func startServerInBackground(configPath string) (stopFunc func() error, errChan chan error) {
	errChan = make(chan error, 1)

	// Initialize server
	srv, closeSessionStore, err := initServer(&options{configPath: configPath})
	if err != nil {
		errChan <- fmt.Errorf("failed to initialize server: %w", err)
		return func() error { return nil }, errChan
	}

	// Start server in goroutine
	go func() {
		if err := srv.Start(); err != nil {
			errChan <- fmt.Errorf("server failed to start: %w", err)
		}
	}()

	stopFunc = func() error {
		// Shutdown server gracefully
		if err := srv.Shutdown(); err != nil {
			return err
		}
		// Close session store
		return closeSessionStore()
	}

	return stopFunc, errChan
}

func findFreePort() int {
	// Simple port finder - starts from 18080 and increments
	basePort := 18080
	for port := basePort; port < basePort+100; port++ {
		if !isPortInUse(port) {
			return port
		}
	}
	return basePort
}

func isPortInUse(port int) bool {
	_, err := http.Get(fmt.Sprintf("http://localhost:%d", port))
	return err == nil
}

// writeConfig serializes a config struct to YAML and writes it to a file
func writeConfig(path string, cfg interface{}) {
	data, err := yaml.Marshal(cfg)
	Expect(err).NotTo(HaveOccurred())
	Expect(os.WriteFile(path, data, 0644)).To(Succeed())
}

// toModuleRawConfig converts a config map to config.ModuleRawConfig ([]byte)
func toModuleRawConfig(cfg map[string]interface{}) []byte {
	data, err := yaml.Marshal(cfg)
	Expect(err).NotTo(HaveOccurred())
	return data
}

// configFile represents the top-level configuration
type configFile struct {
	Sargantana server.SargantanaConfig `yaml:"sargantana"`
}

func writeBasicConfig(path string, port int) {
	cfg := configFile{}
	cfg.Sargantana.WebServerConfig = server.WebServerConfig{
		Address:       fmt.Sprintf(":%d", port),
		SessionName:   "test-session",
		SessionSecret: "test-secret-that-is-at-least-32-chars-long",
	}
	cfg.Sargantana.ControllerBindings = server.ControllerBindings{
		{
			TypeName: "static",
			Config: toModuleRawConfig(map[string]interface{}{
				"status": 200,
				"body":   "OK",
			}),
		},
	}
	writeConfig(path, cfg)
}

type configWithRedis struct {
	Sargantana server.SargantanaConfig `yaml:"sargantana"`
	Redis      database.RedisConfig    `yaml:"redis"`
}

func writeRedisConfig(path string, port int) {
	cfg := configWithRedis{}
	cfg.Sargantana.WebServerConfig = server.WebServerConfig{
		Address:       fmt.Sprintf(":%d", port),
		SessionName:   "test-session",
		SessionSecret: "test-secret-that-is-at-least-32-chars-long",
	}
	cfg.Sargantana.ControllerBindings = server.ControllerBindings{
		{
			TypeName: "static",
			Config: toModuleRawConfig(map[string]interface{}{
				"status": 200,
				"body":   "OK",
			}),
		},
	}
	cfg.Redis = database.RedisConfig{
		Address:  "localhost:6379",
		Username: "redisuser",
		Password: "redispass",
		Database: 0,
	}
	writeConfig(path, cfg)
}

type configWithMongoDB struct {
	Sargantana server.SargantanaConfig `yaml:"sargantana"`
	MongoDB    database.MongoDBConfig  `yaml:"mongodb"`
}

func writeMongoDBConfig(path string, port int) {
	cfg := configWithMongoDB{}
	cfg.Sargantana.WebServerConfig = server.WebServerConfig{
		Address:       fmt.Sprintf(":%d", port),
		SessionName:   "test-session",
		SessionSecret: "test-secret-that-is-at-least-32-chars-long",
	}
	cfg.Sargantana.ControllerBindings = server.ControllerBindings{
		{
			TypeName: "static",
			Config: toModuleRawConfig(map[string]interface{}{
				"status": 200,
				"body":   "OK",
			}),
		},
	}
	cfg.MongoDB = database.MongoDBConfig{
		URI:        "mongodb://testuser:testpass@localhost:27017",
		Database:   "sessions_test",
		AuthSource: "admin",
	}
	writeConfig(path, cfg)
}

type configWithPostgres struct {
	Sargantana server.SargantanaConfig `yaml:"sargantana"`
	Postgres   database.PostgresConfig `yaml:"postgres"`
}

func writePostgresConfig(path string, port int) {
	cfg := configWithPostgres{}
	cfg.Sargantana.WebServerConfig = server.WebServerConfig{
		Address:       fmt.Sprintf(":%d", port),
		SessionName:   "test-session",
		SessionSecret: "test-secret-that-is-at-least-32-chars-long",
	}
	cfg.Sargantana.ControllerBindings = server.ControllerBindings{
		{
			TypeName: "static",
			Config: toModuleRawConfig(map[string]interface{}{
				"status": 200,
				"body":   "OK",
			}),
		},
	}
	cfg.Postgres = database.PostgresConfig{
		Host:     "localhost",
		Port:     5432,
		User:     "user",
		Password: "password",
		Database: "my_blog_db",
		SSLMode:  "disable",
	}
	writeConfig(path, cfg)
}

type configWithMemcached struct {
	Sargantana server.SargantanaConfig  `yaml:"sargantana"`
	Memcached  database.MemcachedConfig `yaml:"memcached"`
}

func writeMemcachedConfig(path string, port int) {
	cfg := configWithMemcached{}
	cfg.Sargantana.WebServerConfig = server.WebServerConfig{
		Address:       fmt.Sprintf(":%d", port),
		SessionName:   "test-session",
		SessionSecret: "test-secret-that-is-at-least-32-chars-long",
	}
	cfg.Sargantana.ControllerBindings = server.ControllerBindings{
		{
			TypeName: "static",
			Config: toModuleRawConfig(map[string]interface{}{
				"status": 200,
				"body":   "OK",
			}),
		},
	}
	cfg.Memcached = database.MemcachedConfig{
		Servers: []string{"localhost:11211"},
	}
	writeConfig(path, cfg)
}

type configWithVault struct {
	Sargantana server.SargantanaConfig `yaml:"sargantana"`
	Vault      secrets.VaultConfig     `yaml:"vault"`
}

func writeVaultConfig(path string, port int) {
	cfg := configWithVault{}
	cfg.Sargantana.WebServerConfig = server.WebServerConfig{
		Address:       fmt.Sprintf(":%d", port),
		SessionName:   "test-session",
		SessionSecret: "${vault:SESSION_SECRET}",
	}
	cfg.Sargantana.ControllerBindings = server.ControllerBindings{
		{
			TypeName: "static",
			Config: toModuleRawConfig(map[string]interface{}{
				"status": 200,
				"body":   "OK",
			}),
		},
	}
	cfg.Vault = secrets.VaultConfig{
		Address: "http://localhost:8200",
		Token:   "dev-root-token",
		Path:    "secret/data/sargantana",
	}
	writeConfig(path, cfg)
}

type configWithAWS struct {
	Sargantana server.SargantanaConfig `yaml:"sargantana"`
	AWS        secrets.AWSConfig       `yaml:"aws"`
}

func writeAWSConfig(path string, port int) {
	cfg := configWithAWS{}
	cfg.Sargantana.WebServerConfig = server.WebServerConfig{
		Address:       fmt.Sprintf(":%d", port),
		SessionName:   "test-session",
		SessionSecret: "${aws:SESSION_SECRET}",
	}
	cfg.Sargantana.ControllerBindings = server.ControllerBindings{
		{
			TypeName: "static",
			Config: toModuleRawConfig(map[string]interface{}{
				"status": 200,
				"body":   "OK",
			}),
		},
	}
	cfg.AWS = secrets.AWSConfig{
		Region:          "us-east-1",
		SecretName:      "sargantana/test",
		AccessKeyID:     "test",
		SecretAccessKey: "test",
		Endpoint:        "http://localhost:4566",
	}
	writeConfig(path, cfg)
}

func writeAllControllersConfig(path string, port int) {
	cfg := configFile{}
	cfg.Sargantana.WebServerConfig = server.WebServerConfig{
		Address:       fmt.Sprintf(":%d", port),
		SessionName:   "test-session",
		SessionSecret: "test-secret-that-is-at-least-32-chars-long",
	}
	cfg.Sargantana.ControllerBindings = server.ControllerBindings{
		{
			TypeName: "static",
			Name:     "health",
			Config: toModuleRawConfig(map[string]interface{}{
				"status": 200,
				"body":   "OK",
			}),
		},
		{
			TypeName: "static",
			Name:     "static-content",
			Config: toModuleRawConfig(map[string]interface{}{
				"status": 200,
				"body":   "Static content",
			}),
		},
		{
			TypeName: "load_balancer",
			Name:     "backend",
			Config: toModuleRawConfig(map[string]interface{}{
				"backends": []map[string]interface{}{
					{"url": "http://localhost:8080"},
				},
			}),
		},
		{
			TypeName: "template",
			Name:     "templates",
			Config: toModuleRawConfig(map[string]interface{}{
				"path": "/tmp/templates",
			}),
		},
		{
			TypeName: "auth",
			Name:     "authentication",
			Config: toModuleRawConfig(map[string]interface{}{
				"callback_host":     fmt.Sprintf("http://localhost:%d", port),
				"login_path":        "/auth/{provider}",
				"callback_path":     "/auth/{provider}/callback",
				"redirect_on_login": "/dashboard",
			}),
		},
	}
	writeConfig(path, cfg)
}
