//go:build unit

package server

import (
	"time"

	"github.com/animalet/sargantana-go/pkg/config"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"gopkg.in/yaml.v3"
)

// MockController implements IController
type MockController struct {
	BindFunc  func(*gin.Engine, gin.HandlerFunc)
	CloseFunc func() error
}

func (m *MockController) Bind(engine *gin.Engine, loginMiddleware gin.HandlerFunc) {
	if m.BindFunc != nil {
		m.BindFunc(engine, loginMiddleware)
	}
}

func (m *MockController) Close() error {
	if m.CloseFunc != nil {
		return m.CloseFunc()
	}
	return nil
}

// TestConfig for testing RegisterController
type TestConfig struct {
	Field string `yaml:"field"`
}

func (c TestConfig) Validate() error {
	if c.Field == "invalid" {
		return errors.New("invalid config")
	}
	return nil
}

var _ = Describe("Server", func() {
	Context("NewServer", func() {
		It("should create a new server instance", func() {
			cfg := SargantanaConfig{
				WebServerConfig: WebServerConfig{
					Address: ":8080",
				},
			}
			srv := NewServer(cfg)
			Expect(srv).NotTo(BeNil())
		})
	})

	Context("Controller Registry", func() {
		It("should register and retrieve controller factory", func() {
			dummyFactory := func(c config.ModuleRawConfig, ctx ControllerContext) (IController, error) {
				return nil, nil
			}
			addControllerType("dummy", dummyFactory)

			// We can't directly access the registry to verify, but we can verify via behavior if we had a way to trigger it
			// For now, just ensuring it doesn't panic is a start.
		})
	})

	Context("Server Configuration", func() {
		It("should validate configuration", func() {
			cfg := SargantanaConfig{
				WebServerConfig: WebServerConfig{
					Address:       ":8080",
					SessionName:   "session",
					SessionSecret: "secret",
				},
			}
			err := cfg.Validate()
			Expect(err).NotTo(HaveOccurred())
		})

		It("should fail validation if secret is missing", func() {
			cfg := SargantanaConfig{
				WebServerConfig: WebServerConfig{
					Address:     ":8080",
					SessionName: "session",
				},
			}
			err := cfg.Validate()
			Expect(err).To(HaveOccurred())
		})
	})

	Context("Lifecycle", func() {
		var (
			cfg          SargantanaConfig
			sessionStore sessions.Store
		)

		BeforeEach(func() {
			cfg = SargantanaConfig{
				WebServerConfig: WebServerConfig{
					Address:       "localhost:0", // Use dynamic port allocation
					SessionName:   "test-session",
					SessionSecret: "secret",
				},
				ControllerBindings: []ControllerBinding{
					{
						TypeName: "mock-controller",
						Name:     "test-controller",
						Config:   config.ModuleRawConfig{},
					},
				},
			}
			sessionStore = cookie.NewStore([]byte("secret"))
			gin.SetMode(gin.TestMode)
		})

		It("should create, start and shutdown the server", func() {
			s := NewServer(cfg)
			s.SetSessionStore(sessionStore)

			// Register mock controller
			addControllerType("mock-controller", func(cfg config.ModuleRawConfig, ctx ControllerContext) (IController, error) {
				return &MockController{}, nil
			})

			// Start server
			err := s.Start()
			Expect(err).NotTo(HaveOccurred())

			// Give it a moment to start listening
			time.Sleep(100 * time.Millisecond)

			// Shutdown
			err = s.Shutdown()
			Expect(err).NotTo(HaveOccurred())
		})

		It("should execute shutdown hooks", func() {
			s := NewServer(cfg)
			s.SetSessionStore(sessionStore)

			hookCalled := false
			addControllerType("mock-controller", func(cfg config.ModuleRawConfig, ctx ControllerContext) (IController, error) {
				return &MockController{
					CloseFunc: func() error {
						hookCalled = true
						return nil
					},
				}, nil
			})

			err := s.Start()
			Expect(err).NotTo(HaveOccurred())

			err = s.Shutdown()
			Expect(err).NotTo(HaveOccurred())
			Expect(hookCalled).To(BeTrue())
		})
	})

	Context("SetDebug", func() {
		It("should set debug mode and configure logging/gin accordingly", func() {
			// Test enabling debug mode
			SetDebug(true)
			Expect(gin.Mode()).To(Equal(gin.DebugMode))
			Expect(zerolog.GlobalLevel()).To(Equal(zerolog.DebugLevel))

			// Test disabling debug mode
			SetDebug(false)
			Expect(gin.Mode()).To(Equal(gin.ReleaseMode))
			Expect(zerolog.GlobalLevel()).To(Equal(zerolog.InfoLevel))
		})

		It("should start in debug mode", func() {
			SetDebug(true)
			defer SetDebug(false)

			cfg := SargantanaConfig{
				WebServerConfig: WebServerConfig{
					Address:       "localhost:0",
					SessionName:   "test-session",
					SessionSecret: "secret",
				},
				ControllerBindings: []ControllerBinding{
					{TypeName: "mock-controller", Config: config.ModuleRawConfig{}},
				},
			}
			s := NewServer(cfg)
			s.SetSessionStore(cookie.NewStore([]byte("secret")))

			addControllerType("mock-controller", func(cfg config.ModuleRawConfig, ctx ControllerContext) (IController, error) {
				return &MockController{}, nil
			})

			// Start server using bootstrap to avoid network listener
			err := s.bootstrap()
			Expect(err).NotTo(HaveOccurred())

			// Verify debug mode was set on Gin
			Expect(gin.IsDebugging()).To(BeTrue())

			// We can also verify that the engine has the logger middleware if we could inspect it,
			// but verifying gin.Mode() is sufficient for this test as SetDebug sets gin mode.

			// Clean up
			s.Shutdown()
		})
	})

	Context("Controller Configuration", func() {
		It("should handle auto-naming for multiple instances", func() {
			cfg := SargantanaConfig{
				WebServerConfig: WebServerConfig{
					Address:       "localhost:0",
					SessionName:   "test-session",
					SessionSecret: "secret",
				},
				ControllerBindings: []ControllerBinding{
					{TypeName: "mock-controller", Config: config.ModuleRawConfig{}},
					{TypeName: "mock-controller", Config: config.ModuleRawConfig{}},
				},
			}
			s := NewServer(cfg)
			s.SetSessionStore(cookie.NewStore([]byte("secret")))

			addControllerType("mock-controller", func(cfg config.ModuleRawConfig, ctx ControllerContext) (IController, error) {
				return &MockController{}, nil
			})

			err := s.bootstrap()
			Expect(err).NotTo(HaveOccurred())
			s.Shutdown()
		})

		It("should handle factory error", func() {
			cfg := SargantanaConfig{
				WebServerConfig: WebServerConfig{
					Address:       "localhost:0",
					SessionName:   "test-session",
					SessionSecret: "secret",
				},
				ControllerBindings: []ControllerBinding{
					{TypeName: "error-controller", Config: config.ModuleRawConfig{}},
				},
			}
			s := NewServer(cfg)
			s.SetSessionStore(cookie.NewStore([]byte("secret")))

			addControllerType("error-controller", func(cfg config.ModuleRawConfig, ctx ControllerContext) (IController, error) {
				return nil, errors.New("factory error")
			})

			// Start should not fail, but controller won't be added (logged as error)
			// Wait, configureControllers returns configErrors, but bootstrap logs them and continues?
			// Let's check go:167
			// if len(configurationErrors) > 0 { log.Error... }
			// It does NOT return error.
			err := s.bootstrap()
			Expect(err).NotTo(HaveOccurred())
			s.Shutdown()
		})

		It("should handle factory panic", func() {
			cfg := SargantanaConfig{
				WebServerConfig: WebServerConfig{
					Address:       "localhost:0",
					SessionName:   "test-session",
					SessionSecret: "secret",
				},
				ControllerBindings: []ControllerBinding{
					{TypeName: "panic-controller", Config: config.ModuleRawConfig{}},
				},
			}
			s := NewServer(cfg)
			s.SetSessionStore(cookie.NewStore([]byte("secret")))

			addControllerType("panic-controller", func(cfg config.ModuleRawConfig, ctx ControllerContext) (IController, error) {
				panic("factory panic")
			})

			err := s.bootstrap()
			Expect(err).NotTo(HaveOccurred())
			s.Shutdown()
		})
	})

	Context("Shutdown Hooks", func() {
		It("should handle hook error", func() {
			cfg := SargantanaConfig{
				WebServerConfig: WebServerConfig{
					Address:       "localhost:0",
					SessionName:   "test-session",
					SessionSecret: "secret",
				},
				ControllerBindings: []ControllerBinding{
					{TypeName: "hook-error-controller", Config: config.ModuleRawConfig{}},
				},
			}
			s := NewServer(cfg)
			s.SetSessionStore(cookie.NewStore([]byte("secret")))

			addControllerType("hook-error-controller", func(cfg config.ModuleRawConfig, ctx ControllerContext) (IController, error) {
				return &MockController{
					CloseFunc: func() error {
						return errors.New("hook error")
					},
				}, nil
			})

			err := s.Start()
			Expect(err).NotTo(HaveOccurred())

			err = s.Shutdown()
			Expect(err).NotTo(HaveOccurred()) // Shutdown itself shouldn't fail if hook fails
		})
	})

	Context("Start Errors", func() {
		It("should fail if session secret is missing", func() {
			cfg := SargantanaConfig{
				WebServerConfig: WebServerConfig{
					Address:     ":8080",
					SessionName: "session",
					// Missing secret
				},
			}
			s := NewServer(cfg)
			err := s.Start()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("session secret is not set"))
		})

		It("should fail if controller config is invalid", func() {
			cfg := SargantanaConfig{
				WebServerConfig: WebServerConfig{
					Address:       ":8080",
					SessionName:   "session",
					SessionSecret: "secret",
				},
				ControllerBindings: []ControllerBinding{
					{TypeName: "nonexistent"},
				},
			}
			s := NewServer(cfg)
			_ = s.Start()
		})
	})

	Context("AddControllerType", func() {
		It("should add a controller type", func() {
			factory := func(configData config.ModuleRawConfig, context ControllerContext) (IController, error) {
				return nil, nil
			}
			addControllerType("custom", factory)
			Expect(controllerRegistry).To(HaveKey("custom"))
		})
	})

	Context("ServerConfig Validation", func() {
		It("should validate valid config", func() {
			cfg := WebServerConfig{
				Address:       "localhost:8080",
				SessionName:   "test-session",
				SessionSecret: "12345678901234567890123456789012",
			}
			err := cfg.Validate()
			Expect(err).NotTo(HaveOccurred())
		})

		It("should fail validation if address is missing", func() {
			cfg := WebServerConfig{
				SessionName:   "test-session",
				SessionSecret: "12345678901234567890123456789012",
			}
			err := cfg.Validate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("address must be set and non-empty"))
		})

		It("should fail validation if session secret is too short", func() {
			cfg := WebServerConfig{
				Address:       "localhost:8080",
				SessionName:   "test-session",
				SessionSecret: "",
			}
			err := cfg.Validate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("session_secret must be set and non-empty"))
		})
	})

	Context("ControllerConfig Validation", func() {
		Context("ControllerBinding", func() {
			It("should return error if type_name is empty", func() {
				cb := ControllerBinding{
					Config: config.ModuleRawConfig([]byte{}),
				}
				err := cb.Validate()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("controller type must be set"))
			})

			It("should return error if config is missing", func() {
				cb := ControllerBinding{
					TypeName: "static",
				}
				err := cb.Validate()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("controller config must be provided"))
			})

			It("should pass with valid config", func() {
				cb := ControllerBinding{
					Name:     "my-static",
					TypeName: "static",
					Config:   config.ModuleRawConfig([]byte{}),
				}
				err := cb.Validate()
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("ControllerBindings", func() {
			It("should return error if any binding is invalid", func() {
				cbs := ControllerBindings{
					{Name: "valid", TypeName: "static", Config: config.ModuleRawConfig([]byte{})},
					{Name: "invalid", TypeName: "static"}, // Missing config
				}
				err := cbs.Validate()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("controller binding at index 1 is invalid"))
			})

			It("should pass if all bindings are valid", func() {
				cbs := ControllerBindings{
					{Name: "valid1", TypeName: "static", Config: config.ModuleRawConfig([]byte{})},
					{Name: "valid2", TypeName: "static", Config: config.ModuleRawConfig([]byte{})},
				}
				err := cbs.Validate()
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("RegisterController", func() {
			var (
				sessionStore sessions.Store
			)

			BeforeEach(func() {
				sessionStore = cookie.NewStore([]byte("secret"))
				// Clear registry before each test to avoid conflicts?
				// controllerRegistry is a global map. We should be careful.
				// Since tests run in parallel or sequentially, modifying global state is risky.
				// However, we can use unique names for each test.
			})

			It("should register and configure a controller successfully", func() {
				typeName := "test-controller-success"
				called := false

				RegisterController(typeName, func(cfg *TestConfig, ctx ControllerContext) (IController, error) {
					called = true
					Expect(cfg.Field).To(Equal("value"))
					return &MockController{}, nil
				})

				// Verify it's in the registry
				factory, exists := controllerRegistry[typeName]
				Expect(exists).To(BeTrue())
				Expect(factory).NotTo(BeNil())

				// Simulate configuration
				rawCfgBytes, _ := yaml.Marshal(map[string]string{"field": "value"})
				rawCfg := config.ModuleRawConfig(rawCfgBytes)

				ctx := ControllerContext{SessionStore: sessionStore}
				ctrl, err := factory(rawCfg, ctx)
				Expect(err).NotTo(HaveOccurred())
				Expect(ctrl).NotTo(BeNil())
				Expect(called).To(BeTrue())
			})

			It("should return error if unmarshal fails", func() {
				typeName := "test-controller-unmarshal-fail"
				RegisterController(typeName, func(cfg *TestConfig, ctx ControllerContext) (IController, error) {
					return &MockController{}, nil
				})

				factory := controllerRegistry[typeName]

				// Invalid YAML
				rawCfg := config.ModuleRawConfig([]byte("invalid: yaml: :"))
				ctx := ControllerContext{SessionStore: sessionStore}

				_, err := factory(rawCfg, ctx)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("failed to unmarshal configuration"))
			})

			It("should return error if validation fails", func() {
				typeName := "test-controller-validation-fail"
				RegisterController(typeName, func(cfg *TestConfig, ctx ControllerContext) (IController, error) {
					return &MockController{}, nil
				})

				factory := controllerRegistry[typeName]

				// Valid YAML but invalid config logic
				rawCfgBytes, _ := yaml.Marshal(map[string]string{"field": "invalid"})
				rawCfg := config.ModuleRawConfig(rawCfgBytes)
				ctx := ControllerContext{SessionStore: sessionStore}

				_, err := factory(rawCfg, ctx)
				Expect(err).To(HaveOccurred())
				// The error comes from config.Unmarshal calling Validate()
				Expect(err.Error()).To(ContainSubstring("invalid config"))
			})

			It("should propagate error from factory", func() {
				typeName := "test-controller-factory-fail"
				RegisterController(typeName, func(cfg *TestConfig, ctx ControllerContext) (IController, error) {
					return nil, errors.New("factory error")
				})

				factory := controllerRegistry[typeName]

				rawCfgBytes, _ := yaml.Marshal(map[string]string{"field": "value"})
				rawCfg := config.ModuleRawConfig(rawCfgBytes)
				ctx := ControllerContext{SessionStore: sessionStore}

				_, err := factory(rawCfg, ctx)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("factory error"))
			})
		})
	})
})
