//go:build unit

package controller

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"time"

	"github.com/animalet/sargantana-go/pkg/config"
	"github.com/animalet/sargantana-go/pkg/server"
	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v3"
)

var _ = Describe("Load Balancer Controller", func() {
	Context("LoadBalancerControllerConfig Validate", func() {
		It("should return error if path is empty", func() {
			cfg := LoadBalancerControllerConfig{
				Endpoints: []string{"http://localhost:8080"},
			}
			err := cfg.Validate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("path must be set"))
		})

		It("should return error if endpoints are empty", func() {
			cfg := LoadBalancerControllerConfig{
				Path: "/api",
			}
			err := cfg.Validate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("at least one endpoint must be provided"))
		})

		It("should return error for invalid endpoint URL", func() {
			cfg := LoadBalancerControllerConfig{
				Path:      "/api",
				Endpoints: []string{"http://%invalid"},
			}
			err := cfg.Validate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid endpoint URL"))
		})

		It("should pass with valid config", func() {
			cfg := LoadBalancerControllerConfig{
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
			var lbCfg LoadBalancerControllerConfig
			err := yaml.Unmarshal(cfgBytes, &lbCfg)
			Expect(err).NotTo(HaveOccurred())

			ctrl, err := NewLoadBalancerController(&lbCfg, server.ControllerContext{})
			Expect(err).NotTo(HaveOccurred())
			Expect(ctrl).NotTo(BeNil())

			ctrl.Bind(engine, nil)
			Expect(ctrl.Close()).To(Succeed())
		})

		It("should return error if endpoint URL is invalid", func() {
			cfgBytes := []byte(`
path: /api
endpoints:
  - http://%
`)
			var lbCfg LoadBalancerControllerConfig
			err := yaml.Unmarshal(cfgBytes, &lbCfg)
			Expect(err).NotTo(HaveOccurred())

			_, err = NewLoadBalancerController(&lbCfg, server.ControllerContext{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to parse load balancer path"))
		})

		It("should handle backend errors", func() {
			// Create a mock backend that always returns 500
			backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			}))
			defer backend.Close()

			cfgBytes := []byte("path: /api\nendpoints:\n  - " + backend.URL)
			var lbCfg LoadBalancerControllerConfig
			err := yaml.Unmarshal(cfgBytes, &lbCfg)
			Expect(err).NotTo(HaveOccurred())

			ctrl, err := NewLoadBalancerController(&lbCfg, server.ControllerContext{})
			Expect(err).NotTo(HaveOccurred())

			ctrl.Bind(engine, nil)

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
			var lbCfg LoadBalancerControllerConfig
			err := yaml.Unmarshal(cfgBytes, &lbCfg)
			Expect(err).NotTo(HaveOccurred())

			ctrl, err := NewLoadBalancerController(&lbCfg, server.ControllerContext{})
			Expect(err).NotTo(HaveOccurred())

			ctrl.Bind(engine, nil)

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

		It("should handle Bind with empty endpoints", func() {
			lb := &loadBalancer{
				endpoints: []url.URL{},
				path:      "/api/*proxyPath",
			}
			lb.Bind(engine, nil)
		})

		It("should bind with auth enabled", func() {
			backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))
			defer backend.Close()

			cfgBytes := []byte("path: /api\nauth: true\nendpoints:\n  - " + backend.URL)
			var lbCfg LoadBalancerControllerConfig
			err := yaml.Unmarshal(cfgBytes, &lbCfg)
			Expect(err).NotTo(HaveOccurred())

			ctrl, err := NewLoadBalancerController(&lbCfg, server.ControllerContext{})
			Expect(err).NotTo(HaveOccurred())

			ctrl.Bind(engine, nil)
		})

		It("should filter sensitive headers", func() {
			receivedHeaders := make(map[string]string)
			backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Store headers for verification after the request
				receivedHeaders["Authorization"] = r.Header.Get("Authorization")
				receivedHeaders["Cookie"] = r.Header.Get("Cookie")
				receivedHeaders["Host"] = r.Header.Get("Host")
				receivedHeaders["X-Forwarded-Proto"] = r.Header.Get("X-Forwarded-Proto")
				receivedHeaders["X-Forwarded-For"] = r.Header.Get("X-Forwarded-For")
				receivedHeaders["Custom-Header"] = r.Header.Get("Custom-Header")
				w.Header().Set("Set-Cookie", "session=123")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("ok"))
			}))
			defer backend.Close()

			cfgBytes := []byte("path: /api\nendpoints:\n  - " + backend.URL)
			var lbCfg LoadBalancerControllerConfig
			err := yaml.Unmarshal(cfgBytes, &lbCfg)
			Expect(err).NotTo(HaveOccurred())

			ctrl, err := NewLoadBalancerController(&lbCfg, server.ControllerContext{})
			Expect(err).NotTo(HaveOccurred())

			ctrl.Bind(engine, nil)

			req, _ := http.NewRequest("GET", "/api/test", nil)
			req.Header.Set("Authorization", "Bearer token")
			req.Header.Set("Cookie", "session=456")
			req.Header.Set("Host", "original-host")
			req.Header.Set("X-Forwarded-Proto", "https")
			req.Header.Set("Custom-Header", "test-value")

			engine.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusOK))
			// Verify headers after the request completes
			Expect(receivedHeaders["Authorization"]).To(BeEmpty())
			Expect(receivedHeaders["Cookie"]).To(BeEmpty())
			Expect(receivedHeaders["Host"]).To(BeEmpty())
			Expect(receivedHeaders["X-Forwarded-Proto"]).To(BeEmpty())
			// X-Forwarded-For will be set (could be empty in test environment)
			_, xForwardedForExists := receivedHeaders["X-Forwarded-For"]
			Expect(xForwardedForExists).To(BeTrue())
			Expect(receivedHeaders["Custom-Header"]).To(Equal("test-value"))
			// Check that Set-Cookie is not forwarded to client
			Expect(w.Header().Get("Set-Cookie")).To(BeEmpty())
		})

		It("should handle unreachable backend", func() {
			cfgBytes := []byte("path: /api\nendpoints:\n  - http://localhost:1")
			var lbCfg LoadBalancerControllerConfig
			err := yaml.Unmarshal(cfgBytes, &lbCfg)
			Expect(err).NotTo(HaveOccurred())

			ctrl, err := NewLoadBalancerController(&lbCfg, server.ControllerContext{})
			Expect(err).NotTo(HaveOccurred())

			ctrl.Bind(engine, nil)

			req, _ := http.NewRequest("GET", "/api/test", nil)
			engine.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusBadGateway))
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

				var lbCfg LoadBalancerControllerConfig
				err := yaml.Unmarshal(rawConfig, &lbCfg)
				Expect(err).NotTo(HaveOccurred())

				ctx := server.ControllerContext{}
				ctrl, err := NewLoadBalancerController(&lbCfg, ctx)
				Expect(err).NotTo(HaveOccurred())
				Expect(ctrl).NotTo(BeNil())
			})

			It("should fail with invalid config", func() {
				rawConfig := config.ModuleRawConfig([]byte(`invalid yaml`))
				var lbCfg LoadBalancerControllerConfig
				err := yaml.Unmarshal(rawConfig, &lbCfg)
				// Expect(err).To(HaveOccurred()) // Unmarshal might fail or succeed depending on garbage, but Validate is called inside RegisterController usually.
				// Here we are calling NewLoadBalancerController directly which assumes valid config struct if passed.
				// But wait, NewLoadBalancerController used to unmarshal. Now it takes struct.
				// So if we pass a struct, it is "valid" as a struct.
				// The validation logic is in Validate() which is called by Register
				// So unit testing NewLoadBalancerController with invalid yaml is now "testing yaml unmarshal" which is outside scope,
				// OR we test that NewLoadBalancerController handles invalid struct data if it does any validation itself?
				// It seems NewLoadBalancerController does some validation (url parsing).

				// Let's adjust the test to pass an invalid config struct that causes NewLoadBalancerController to fail.
				// The original test passed "invalid yaml".
				// If we pass a struct with invalid URL in endpoints, it should fail.
				lbCfg = LoadBalancerControllerConfig{
					Path:      "/api",
					Endpoints: []string{"http://%"},
				}
				ctx := server.ControllerContext{}
				_, err = NewLoadBalancerController(&lbCfg, ctx)
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

				var lbCfg LoadBalancerControllerConfig
				err := yaml.Unmarshal(rawConfig, &lbCfg)
				Expect(err).NotTo(HaveOccurred())

				ctx := server.ControllerContext{}
				ctrl, _ := NewLoadBalancerController(&lbCfg, ctx)

				engine := gin.New()
				ctrl.Bind(engine, nil)

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

			It("should forward POST, PUT, DELETE, PATCH, HEAD, OPTIONS methods", func() {
				backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(r.Method))
				}))
				defer backend.Close()

				rawConfigMap := map[string]interface{}{
					"path": "/api",
					"endpoints": []string{
						backend.URL,
					},
				}
				rawConfigBytes, _ := yaml.Marshal(rawConfigMap)
				rawConfig := config.ModuleRawConfig(rawConfigBytes)

				var lbCfg LoadBalancerControllerConfig
				err := yaml.Unmarshal(rawConfig, &lbCfg)
				Expect(err).NotTo(HaveOccurred())

				ctx := server.ControllerContext{}
				ctrl, _ := NewLoadBalancerController(&lbCfg, ctx)

				engine := gin.New()
				ctrl.Bind(engine, nil)

				methods := []string{"POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"}
				for _, method := range methods {
					req, _ := http.NewRequest(method, "/api/test", nil)
					w := httptest.NewRecorder()
					engine.ServeHTTP(w, req)
					Expect(w.Code).To(Equal(http.StatusOK))
					if method != "HEAD" {
						Expect(w.Body.String()).To(Equal(method))
					}
				}
			})
		})
	})
})
