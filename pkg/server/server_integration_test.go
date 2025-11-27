//go:build unit

package server

import (
	"net/http"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Server Integration", func() {
	var (
		serverConf SargantanaConfig
		srv        *Server
		port       string
	)

	BeforeEach(func() {
		// Find a free port
		port = "48080" // simplified for this example, or use a random port
		serverConf = SargantanaConfig{
			WebServerConfig: WebServerConfig{
				Address:       ":" + port,
				SessionName:   "test-session",
				SessionSecret: "secret",
			},
		}
	})

	AfterEach(func() {
		if srv != nil {
			_ = srv.Shutdown()
		}
	})

	Context("when security middleware is configured", func() {
		It("should set security headers", func() {
			serverConf.WebServerConfig.Security = &SecurityConfig{
				FrameDeny:               true,
				CustomFrameOptionsValue: "DENY",
				ContentTypeNosniff:      true,
				BrowserXssFilter:        true,
			}

			srv = NewServer(serverConf)
			go func() {
				defer GinkgoRecover()
				_ = srv.Start()
			}()

			// Wait for server to start
			Eventually(func() error {
				_, err := http.Get("http://localhost:" + port + "/health")
				return err
			}, 5*time.Second, 100*time.Millisecond).Should(Succeed())

			resp, err := http.Get("http://localhost:" + port + "/health") // Assuming /health or any endpoint exists, or 404 is fine for headers
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			Expect(resp.Header.Get("X-Frame-Options")).To(Equal("DENY"))
			Expect(resp.Header.Get("X-Content-Type-Options")).To(Equal("nosniff"))
			Expect(resp.Header.Get("X-XSS-Protection")).To(Equal("1; mode=block"))
		})
	})

	Context("when security middleware is NOT configured", func() {
		It("should NOT set security headers", func() {
			serverConf.WebServerConfig.Security = nil

			srv = NewServer(serverConf)
			go func() {
				defer GinkgoRecover()
				_ = srv.Start()
			}()

			// Wait for server to start
			Eventually(func() error {
				_, err := http.Get("http://localhost:" + port + "/health")
				return err
			}, 5*time.Second, 100*time.Millisecond).Should(Succeed())

			resp, err := http.Get("http://localhost:" + port + "/health")
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			Expect(resp.Header.Get("X-Frame-Options")).To(BeEmpty())
			Expect(resp.Header.Get("X-Content-Type-Options")).To(BeEmpty())
		})
	})
})
