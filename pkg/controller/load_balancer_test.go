//go:build unit

package controller_test

import (
	"net/http"
	"net/http/httptest"

	"github.com/animalet/sargantana-go/pkg/config"
	"github.com/animalet/sargantana-go/pkg/controller"
	"github.com/animalet/sargantana-go/pkg/server"
	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v3"
)

var _ = Describe("LoadBalancerController", func() {
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
