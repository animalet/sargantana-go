//go:build unit

package server

import (
	"net/http"
	"net/http/httptest"

	"github.com/animalet/sargantana-go/pkg/config"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Server Integration", func() {
	Context("Security Headers", func() {
		It("should apply custom CSP from configuration", func() {
			// Define configuration with custom CSP
			customCSP := "default-src 'none'; script-src 'self' https://trusted.com"
			cfg := SargantanaConfig{
				WebServerConfig: WebServerConfig{
					Address:       "localhost:0",
					SessionName:   "test-session",
					SessionSecret: "secret",
					CSP:           customCSP,
				},
				ControllerBindings: []ControllerBinding{
					{
						TypeName: "mock-controller",
						Config:   config.ModuleRawConfig{},
					},
				},
			}

			// Create server
			s := NewServer(cfg)
			s.SetSessionStore(cookie.NewStore([]byte("secret")))

			// Register a mock controller that binds a route
			addControllerType("mock-controller", func(cfg config.ModuleRawConfig, ctx ControllerContext) (IController, error) {
				return &MockController{
					BindFunc: func(engine *gin.Engine) {
						engine.GET("/test", func(c *gin.Context) {
							c.Status(http.StatusOK)
						})
					},
				}, nil
			})

			// Initialize server (creates handler but doesn't start listening)
			err := s.bootstrap()
			Expect(err).NotTo(HaveOccurred())

			// Create a request
			req, _ := http.NewRequest("GET", "/test", nil)
			w := httptest.NewRecorder()

			// Serve the request using the server's handler
			s.httpServer.Handler.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(w.Header().Get("Content-Security-Policy")).To(Equal(customCSP))
			Expect(w.Header().Get("Permissions-Policy")).To(Equal("geolocation=(), microphone=(), camera=(), payment=()"))
		})
	})
})
