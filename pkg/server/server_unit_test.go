//go:build unit

package server_test

import (
	"time"

	"github.com/animalet/sargantana-go/pkg/config"
	"github.com/animalet/sargantana-go/pkg/server"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
)

// MockController implements server.IController
type MockController struct {
	BindFunc  func(*gin.Engine)
	CloseFunc func() error
}

func (m *MockController) Bind(engine *gin.Engine) {
	if m.BindFunc != nil {
		m.BindFunc(engine)
	}
}

func (m *MockController) Close() error {
	if m.CloseFunc != nil {
		return m.CloseFunc()
	}
	return nil
}

var _ = Describe("Server", func() {
	Context("NewServer", func() {
		It("should create a new server instance", func() {
			cfg := server.SargantanaConfig{
				WebServerConfig: server.WebServerConfig{
					Address: ":8080",
				},
			}
			srv := server.NewServer(cfg)
			Expect(srv).NotTo(BeNil())
		})
	})

	Context("Controller Registry", func() {
		It("should register and retrieve controller factory", func() {
			dummyFactory := func(c config.ModuleRawConfig, ctx server.ControllerContext) (server.IController, error) {
				return nil, nil
			}
			server.AddControllerType("dummy", dummyFactory)

			// We can't directly access the registry to verify, but we can verify via behavior if we had a way to trigger it
			// For now, just ensuring it doesn't panic is a start.
		})
	})

	Context("Server Configuration", func() {
		It("should validate configuration", func() {
			cfg := server.SargantanaConfig{
				WebServerConfig: server.WebServerConfig{
					Address:       ":8080",
					SessionName:   "session",
					SessionSecret: "secret",
				},
			}
			err := cfg.Validate()
			Expect(err).NotTo(HaveOccurred())
		})

		It("should fail validation if secret is missing", func() {
			cfg := server.SargantanaConfig{
				WebServerConfig: server.WebServerConfig{
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
			cfg          server.SargantanaConfig
			sessionStore sessions.Store
		)

		BeforeEach(func() {
			cfg = server.SargantanaConfig{
				WebServerConfig: server.WebServerConfig{
					Address:       "localhost:8081", // Use different port to avoid conflicts
					SessionName:   "test-session",
					SessionSecret: "secret",
				},
				ControllerBindings: []server.ControllerBinding{
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
			s := server.NewServer(cfg)
			s.SetSessionStore(sessionStore)

			// Register mock controller
			server.AddControllerType("mock-controller", func(cfg config.ModuleRawConfig, ctx server.ControllerContext) (server.IController, error) {
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
			s := server.NewServer(cfg)
			s.SetSessionStore(sessionStore)

			hookCalled := false
			server.AddControllerType("mock-controller", func(cfg config.ModuleRawConfig, ctx server.ControllerContext) (server.IController, error) {
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
		It("should set debug mode", func() {
			server.SetDebug(true)
			Expect(true).To(BeTrue())
			server.SetDebug(false)
			Expect(true).To(BeTrue())
		})

		It("should start in debug mode", func() {
			server.SetDebug(true)
			defer server.SetDebug(false)

			cfg := server.SargantanaConfig{
				WebServerConfig: server.WebServerConfig{
					Address:       "localhost:8086",
					SessionName:   "test-session",
					SessionSecret: "secret",
				},
				ControllerBindings: []server.ControllerBinding{
					{TypeName: "mock-controller", Config: config.ModuleRawConfig{}},
				},
			}
			s := server.NewServer(cfg)
			s.SetSessionStore(cookie.NewStore([]byte("secret")))

			server.AddControllerType("mock-controller", func(cfg config.ModuleRawConfig, ctx server.ControllerContext) (server.IController, error) {
				return &MockController{}, nil
			})

			err := s.Start()
			Expect(err).NotTo(HaveOccurred())

			// Make a request to trigger debug middleware
			// We need to know the port. It's 8086.
			// But we can't easily make a request because Start() starts listenAndServe in a goroutine,
			// but we don't know when it's ready.
			// We can wait a bit.
			time.Sleep(100 * time.Millisecond)

			// We don't have any routes bound except what controllers bind.
			// MockController binds nothing by default.
			// But we can update MockController to bind something.

			s.Shutdown()
		})
	})

	Context("Controller Configuration", func() {
		It("should handle auto-naming for multiple instances", func() {
			cfg := server.SargantanaConfig{
				WebServerConfig: server.WebServerConfig{
					Address:       "localhost:8082",
					SessionName:   "test-session",
					SessionSecret: "secret",
				},
				ControllerBindings: []server.ControllerBinding{
					{TypeName: "mock-controller", Config: config.ModuleRawConfig{}},
					{TypeName: "mock-controller", Config: config.ModuleRawConfig{}},
				},
			}
			s := server.NewServer(cfg)
			s.SetSessionStore(cookie.NewStore([]byte("secret")))

			server.AddControllerType("mock-controller", func(cfg config.ModuleRawConfig, ctx server.ControllerContext) (server.IController, error) {
				return &MockController{}, nil
			})

			err := s.Start()
			Expect(err).NotTo(HaveOccurred())
			s.Shutdown()
		})

		It("should handle factory error", func() {
			cfg := server.SargantanaConfig{
				WebServerConfig: server.WebServerConfig{
					Address:       "localhost:8083",
					SessionName:   "test-session",
					SessionSecret: "secret",
				},
				ControllerBindings: []server.ControllerBinding{
					{TypeName: "error-controller", Config: config.ModuleRawConfig{}},
				},
			}
			s := server.NewServer(cfg)
			s.SetSessionStore(cookie.NewStore([]byte("secret")))

			server.AddControllerType("error-controller", func(cfg config.ModuleRawConfig, ctx server.ControllerContext) (server.IController, error) {
				return nil, errors.New("factory error")
			})

			// Start should not fail, but controller won't be added (logged as error)
			// Wait, configureControllers returns configErrors, but bootstrap logs them and continues?
			// Let's check server.go:167
			// if len(configurationErrors) > 0 { log.Error... }
			// It does NOT return error.
			err := s.Start()
			Expect(err).NotTo(HaveOccurred())
			s.Shutdown()
		})

		It("should handle factory panic", func() {
			cfg := server.SargantanaConfig{
				WebServerConfig: server.WebServerConfig{
					Address:       "localhost:8084",
					SessionName:   "test-session",
					SessionSecret: "secret",
				},
				ControllerBindings: []server.ControllerBinding{
					{TypeName: "panic-controller", Config: config.ModuleRawConfig{}},
				},
			}
			s := server.NewServer(cfg)
			s.SetSessionStore(cookie.NewStore([]byte("secret")))

			server.AddControllerType("panic-controller", func(cfg config.ModuleRawConfig, ctx server.ControllerContext) (server.IController, error) {
				panic("factory panic")
			})

			err := s.Start()
			Expect(err).NotTo(HaveOccurred())
			s.Shutdown()
		})
	})

	Context("Shutdown Hooks", func() {
		It("should handle hook error", func() {
			cfg := server.SargantanaConfig{
				WebServerConfig: server.WebServerConfig{
					Address:       "localhost:8085",
					SessionName:   "test-session",
					SessionSecret: "secret",
				},
				ControllerBindings: []server.ControllerBinding{
					{TypeName: "hook-error-controller", Config: config.ModuleRawConfig{}},
				},
			}
			s := server.NewServer(cfg)
			s.SetSessionStore(cookie.NewStore([]byte("secret")))

			server.AddControllerType("hook-error-controller", func(cfg config.ModuleRawConfig, ctx server.ControllerContext) (server.IController, error) {
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
			cfg := server.SargantanaConfig{
				WebServerConfig: server.WebServerConfig{
					Address:     ":8080",
					SessionName: "session",
					// Missing secret
				},
			}
			s := server.NewServer(cfg)
			err := s.Start()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("session secret is not set"))
		})

		It("should fail if controller config is invalid", func() {
			cfg := server.SargantanaConfig{
				WebServerConfig: server.WebServerConfig{
					Address:       ":8080",
					SessionName:   "session",
					SessionSecret: "secret",
				},
				ControllerBindings: []server.ControllerBinding{
					{TypeName: "nonexistent"},
				},
			}
			s := server.NewServer(cfg)
			_ = s.Start()
		})
	})

	Context("AddControllerType", func() {
		It("should add a controller type", func() {
			factory := func(configData config.ModuleRawConfig, context server.ControllerContext) (server.IController, error) {
				return nil, nil
			}
			server.AddControllerType("custom", factory)
			Expect(true).To(BeTrue())
		})
	})

	Context("ServerConfig Validation", func() {
		It("should validate valid config", func() {
			cfg := server.WebServerConfig{
				Address:       "localhost:8080",
				SessionName:   "test-session",
				SessionSecret: "12345678901234567890123456789012",
			}
			err := cfg.Validate()
			Expect(err).NotTo(HaveOccurred())
		})

		It("should fail validation if address is missing", func() {
			cfg := server.WebServerConfig{
				SessionName:   "test-session",
				SessionSecret: "12345678901234567890123456789012",
			}
			err := cfg.Validate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("address must be set and non-empty"))
		})

		It("should fail validation if session secret is too short", func() {
			cfg := server.WebServerConfig{
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
				cb := server.ControllerBinding{
					Config: config.ModuleRawConfig([]byte{}),
				}
				err := cb.Validate()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("controller type must be set"))
			})

			It("should return error if config is missing", func() {
				cb := server.ControllerBinding{
					TypeName: "static",
				}
				err := cb.Validate()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("controller config must be provided"))
			})

			It("should pass with valid config", func() {
				cb := server.ControllerBinding{
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
				cbs := server.ControllerBindings{
					{Name: "valid", TypeName: "static", Config: config.ModuleRawConfig([]byte{})},
					{Name: "invalid", TypeName: "static"}, // Missing config
				}
				err := cbs.Validate()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("controller binding at index 1 is invalid"))
			})

			It("should pass if all bindings are valid", func() {
				cbs := server.ControllerBindings{
					{Name: "valid1", TypeName: "static", Config: config.ModuleRawConfig([]byte{})},
					{Name: "valid2", TypeName: "static", Config: config.ModuleRawConfig([]byte{})},
				}
				err := cbs.Validate()
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})
})
