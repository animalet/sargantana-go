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

	Context("Lifecycle", func() {
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

	Context("Configuration", func() {
		It("should create default session store if not provided", func() {
			s := server.NewServer(cfg)
			// Don't set session store

			server.AddControllerType("mock-controller", func(cfg config.ModuleRawConfig, ctx server.ControllerContext) (server.IController, error) {
				return &MockController{}, nil
			})

			err := s.Start()
			Expect(err).NotTo(HaveOccurred())
			s.Shutdown()
		})

		It("should fail if session secret is missing for default store", func() {
			cfg.WebServerConfig.SessionSecret = ""
			s := server.NewServer(cfg)

			err := s.Start()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("session secret is not set"))
		})
	})
})
