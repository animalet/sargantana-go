//go:build unit

package controller_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/animalet/sargantana-go/pkg/config"
	"github.com/animalet/sargantana-go/pkg/controller"
	"github.com/animalet/sargantana-go/pkg/server"
	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v3"
)

var _ = Describe("Load Balancer Controller", func() {
	Context("LoadBalancerControllerConfig Validate", func() {
		It("should return error if path is empty", func() {
			cfg := controller.LoadBalancerControllerConfig{
				Endpoints: []string{"http://localhost:8080"},
			}
			err := cfg.Validate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("path must be set"))
		})

		It("should return error if endpoints are empty", func() {
			cfg := controller.LoadBalancerControllerConfig{
				Path: "/api",
			}
			err := cfg.Validate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("at least one endpoint must be provided"))
		})

		It("should pass with valid config", func() {
			cfg := controller.LoadBalancerControllerConfig{
				Path:      "/api",
				Endpoints: []string{"http://localhost:8080"},
			}
			err := cfg.Validate()
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("NewLoadBalancerController", func() {
		var (
			engine *gin.Engine
			w      *httptest.ResponseRecorder
		)

		BeforeEach(func() {
			gin.SetMode(gin.TestMode)
			w = httptest.NewRecorder()
			_, engine = gin.CreateTestContext(w)
		})

		It("should create a new controller", func() {
			cfgBytes := []byte(`
path: /api
endpoints:
  - http://localhost:8081
  - http://localhost:8082
`)
			ctrl, err := controller.NewLoadBalancerController(cfgBytes, server.ControllerContext{})
			Expect(err).NotTo(HaveOccurred())
			Expect(ctrl).NotTo(BeNil())

			ctrl.Bind(engine)
			Expect(ctrl.Close()).To(Succeed())
		})

		It("should return error if endpoint URL is invalid", func() {
			cfgBytes := []byte(`
path: /api
endpoints:
  - http://%
`)
			_, err := controller.NewLoadBalancerController(cfgBytes, server.ControllerContext{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid endpoint URL"))
		})

		It("should handle backend errors", func() {
			// Create a mock backend that always returns 500
			backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			}))
			defer backend.Close()

			cfgBytes := []byte("path: /api\nendpoints:\n  - " + backend.URL)
			ctrl, err := controller.NewLoadBalancerController(cfgBytes, server.ControllerContext{})
			Expect(err).NotTo(HaveOccurred())

			ctrl.Bind(engine)

			req, _ := http.NewRequest("GET", "/api/test", nil)
			engine.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusInternalServerError))
		})

		It("should handle client disconnect", func() {
			// Create a backend that sleeps
			backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Simulate slow response
				select {
				case <-r.Context().Done():
					return
				case <-time.After(100 * time.Millisecond):
					w.WriteHeader(http.StatusOK)
				}
			}))
			defer backend.Close()

			cfgBytes := []byte("path: /api\nendpoints:\n  - " + backend.URL)
			ctrl, err := controller.NewLoadBalancerController(cfgBytes, server.ControllerContext{})
			Expect(err).NotTo(HaveOccurred())

			ctrl.Bind(engine)

			// Create a request with a cancelled context
			req, _ := http.NewRequest("GET", "/api/test", nil)
			ctx, cancel := context.WithCancel(req.Context())
			req = req.WithContext(ctx)

			// Cancel immediately
			cancel()

			engine.ServeHTTP(w, req)

			// Gin/Go HTTP server might handle this differently depending on when cancel happens,
			// but usually it results in 499 or just closing connection.
			// Since we are using httptest.ResponseRecorder, it might not capture "client disconnect" fully
			// as a status code unless the handler sets it.
			// The LoadBalancer uses ReverseProxy which handles context cancellation.
			// If the client cancels, ReverseProxy should stop.
			// We mainly want to ensure it doesn't panic and covers the error handler path if possible.
			// The error handler in load_balancer.go checks for context.Canceled.
		})
	})

	Context("LoadBalancerController Integration", func() {
		var (
			backend1 *httptest.Server
			backend2 *httptest.Server
		)

		BeforeEach(func() {
			backend1 = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("backend1"))
			}))
			backend2 = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("backend2"))
			}))
			gin.SetMode(gin.TestMode)
		})

		AfterEach(func() {
			backend1.Close()
			backend2.Close()
		})

		Context("Configuration", func() {
			It("should create a new load balancer controller", func() {
				rawConfigMap := map[string]interface{}{
					"path": "/api",
					"endpoints": []string{
						backend1.URL,
						backend2.URL,
					},
				}
				rawConfigBytes, _ := yaml.Marshal(rawConfigMap)
				rawConfig := config.ModuleRawConfig(rawConfigBytes)

				ctx := server.ControllerContext{}
				ctrl, err := controller.NewLoadBalancerController(rawConfig, ctx)
				Expect(err).NotTo(HaveOccurred())
				Expect(ctrl).NotTo(BeNil())
			})

			It("should fail with invalid config", func() {
				rawConfig := config.ModuleRawConfig([]byte(`invalid yaml`))
				ctx := server.ControllerContext{}
				_, err := controller.NewLoadBalancerController(rawConfig, ctx)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("Forwarding", func() {
			It("should forward requests round-robin", func() {
				rawConfigMap := map[string]interface{}{
					"path": "/api",
					"endpoints": []string{
						backend1.URL,
						backend2.URL,
					},
				}
				rawConfigBytes, _ := yaml.Marshal(rawConfigMap)
				rawConfig := config.ModuleRawConfig(rawConfigBytes)

				ctx := server.ControllerContext{}
				ctrl, _ := controller.NewLoadBalancerController(rawConfig, ctx)

				engine := gin.New()
				ctrl.Bind(engine)

				// First request -> backend1
				req1, _ := http.NewRequest("GET", "/api/resource", nil)
				w1 := httptest.NewRecorder()
				engine.ServeHTTP(w1, req1)
				Expect(w1.Code).To(Equal(http.StatusOK))
				// Note: Round robin order might depend on implementation details,
				// but we expect it to alternate. Since we can't easily check which backend served it
				// without parsing body, let's check body.
				body1 := w1.Body.String()

				// Second request -> backend2
				req2, _ := http.NewRequest("GET", "/api/resource", nil)
				w2 := httptest.NewRecorder()
				engine.ServeHTTP(w2, req2)
				Expect(w2.Code).To(Equal(http.StatusOK))
				body2 := w2.Body.String()

				// One should be backend1, other backend2
				Expect([]string{body1, body2}).To(ConsistOf("backend1", "backend2"))
			})
		})
	})
})
