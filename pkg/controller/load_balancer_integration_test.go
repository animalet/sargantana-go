//go:build integration

package controller

import (
	"fmt"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"time"

	"github.com/animalet/sargantana-go/pkg/server"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/markbates/goth"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Load Balancer Controller Auth Integration", func() {
	var (
		ts            *httptest.Server
		client        *http.Client
		listener      net.Listener
		engine        *gin.Engine
		backend       *httptest.Server
		backendCalled bool
		authenticator server.Authenticator
		store         sessions.Store
		sessionSecret = "test-secret-key-32-bytes-long!!"
	)

	BeforeEach(func() {
		var err error
		gin.SetMode(gin.TestMode)

		// Setup backend server that will receive forwarded requests
		backendCalled = false
		backend = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			backendCalled = true
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("backend-response"))
		}))
		DeferCleanup(backend.Close)

		// Create listener with dynamic port
		listener, err = net.Listen("tcp", "127.0.0.1:0")
		Expect(err).NotTo(HaveOccurred())

		// Setup session store (use shared variable)
		store = cookie.NewStore([]byte(sessionSecret))
		store.Options(sessions.Options{
			Path:     "/",
			HttpOnly: true,
			Secure:   false,
			MaxAge:   86400 * 30,
		})

		// Create engine
		engine = gin.New()
		engine.Use(sessions.Sessions("mysession", store))

		// Create authenticator
		authenticator = NewGothAuthenticator()

		// Start test server
		ts = &httptest.Server{
			Listener: listener,
			Config:   &http.Server{Handler: engine},
		}
		ts.Start()
		DeferCleanup(ts.Close)

		// Create HTTP client with cookie jar
		jar, err := cookiejar.New(nil)
		Expect(err).NotTo(HaveOccurred())
		client = &http.Client{
			Jar: jar,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}
	})

	Context("Load Balancer with Authentication Disabled", func() {
		BeforeEach(func() {
			// Create load balancer without auth
			cfg := &LoadBalancerControllerConfig{
				Path:      "/api",
				Endpoints: []string{backend.URL},
				Auth:      false,
			}

			ctrl, err := NewLoadBalancerController(cfg, server.ControllerContext{})
			Expect(err).NotTo(HaveOccurred())
			ctrl.Bind(engine, authenticator.Middleware())
		})

		It("should forward requests without authentication", func() {
			resp, err := client.Get(fmt.Sprintf("%s/api/test", ts.URL))
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			Expect(backendCalled).To(BeTrue())
		})
	})

	Context("Load Balancer with Authentication Enabled", func() {
		BeforeEach(func() {
			// Get port for callback URL
			port := listener.Addr().(*net.TCPAddr).Port
			baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)

			// Setup auth controller for login with real OpenID Connect provider
			authCfg := &AuthControllerConfig{
				Providers: map[string]ProviderConfig{
					"openid-connect": {
						Key:    "sargantana",
						Secret: "test-secret",
						URL:    "http://localhost:8081/default/.well-known/openid-configuration",
						Scopes: []string{"openid", "profile", "email"},
					},
				},
				CallbackHost:     baseURL,
				CallbackPath:     "/auth/{provider}/callback",
				UserInfoPath:     "/auth/user",
				LoginPath:        "/auth/{provider}",
				LogoutPath:       "/auth/logout",
				RedirectOnLogin:  "/",
				RedirectOnLogout: "/",
			}
			ctx := server.ControllerContext{
				ServerConfig: server.WebServerConfig{
					Address: fmt.Sprintf(":%d", port),
				},
				SessionStore: store,
			}
			authCtrl, err := NewAuthController(authCfg, ctx)
			Expect(err).NotTo(HaveOccurred())
			authCtrl.Bind(engine, authenticator.Middleware())

			// Create load balancer with auth
			lbCfg := &LoadBalancerControllerConfig{
				Path:      "/api",
				Endpoints: []string{backend.URL},
				Auth:      true,
			}
			lbCtrl, err := NewLoadBalancerController(lbCfg, server.ControllerContext{})
			Expect(err).NotTo(HaveOccurred())
			lbCtrl.Bind(engine, authenticator.Middleware())
		})

		It("should return 401 when accessing without authentication", func() {
			resp, err := client.Get(fmt.Sprintf("%s/api/test", ts.URL))
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusUnauthorized))
			Expect(backendCalled).To(BeFalse())
		})

		It("should allow access after successful authentication", func() {
			// Step 1: Initiate login
			resp, err := client.Get(fmt.Sprintf("%s/auth/openid-connect", ts.URL))
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()
			Expect(resp.StatusCode).To(SatisfyAny(Equal(http.StatusFound), Equal(http.StatusTemporaryRedirect)))

			// Expect redirect to IDP
			location, err := resp.Location()
			Expect(err).NotTo(HaveOccurred())
			Expect(location.Host).To(ContainSubstring("localhost:8081"))
			Expect(location.Path).To(ContainSubstring("/default/authorize"))

			// Capture state parameter
			state := location.Query().Get("state")
			Expect(state).NotTo(BeEmpty())

			// Step 2: Follow redirect to IDP (mock server auto-redirects back with code)
			resp, err = client.Get(location.String())
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()
			Expect(resp.StatusCode).To(SatisfyAny(Equal(http.StatusFound), Equal(http.StatusTemporaryRedirect), Equal(http.StatusSeeOther)))

			// Expect redirect back to callback
			location, err = resp.Location()
			Expect(err).NotTo(HaveOccurred())
			Expect(location.Path).To(Equal("/auth/openid-connect/callback"))

			// Step 3: Follow redirect to callback (code exchange)
			resp, err = client.Get(location.String())
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()
			Expect(resp.StatusCode).To(SatisfyAny(Equal(http.StatusFound), Equal(http.StatusTemporaryRedirect), Equal(http.StatusSeeOther)))

			// Expect redirect to configured redirect path
			location, err = resp.Location()
			Expect(err).NotTo(HaveOccurred())
			Expect(location.Path).To(Equal("/"))

			// Step 4: Access protected load balancer endpoint with session
			resp, err = client.Get(fmt.Sprintf("%s/api/test", ts.URL))
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			Expect(backendCalled).To(BeTrue())
		})

		It("should forward all HTTP methods when authenticated", func() {
			// Complete OAuth flow first
			resp, err := client.Get(fmt.Sprintf("%s/auth/openid-connect", ts.URL))
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			location, err := resp.Location()
			Expect(err).NotTo(HaveOccurred())

			// Follow to IDP
			resp, err = client.Get(location.String())
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			location, err = resp.Location()
			Expect(err).NotTo(HaveOccurred())

			// Follow to callback
			resp, err = client.Get(location.String())
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			// Test various HTTP methods
			methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"}
			for _, method := range methods {
				backendCalled = false
				req, err := http.NewRequest(method, fmt.Sprintf("%s/api/test", ts.URL), nil)
				Expect(err).NotTo(HaveOccurred())
				resp, err := client.Do(req)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusOK), "Method %s should succeed", method)
				Expect(backendCalled).To(BeTrue(), "Backend should be called for method %s", method)
			}
		})

		It("should return 401 when session expires", func() {
			// Create a session with expired user
			engine.GET("/test-expired-session", func(c *gin.Context) {
				session := sessions.Default(c)
				expiredUser := UserObject{
					User: goth.User{
						UserID:    "test-user",
						Provider:  "openid-connect",
						Email:     "test@example.com",
						ExpiresAt: time.Now().Add(-1 * time.Hour), // Expired
					},
				}
				session.Set("user", expiredUser)
				_ = session.Save()
				c.Status(http.StatusOK)
			})

			// Set expired session
			resp, err := client.Get(fmt.Sprintf("%s/test-expired-session", ts.URL))
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			// Try to access protected endpoint with expired session
			resp, err = client.Get(fmt.Sprintf("%s/api/test", ts.URL))
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusUnauthorized))
			Expect(backendCalled).To(BeFalse())
		})
	})
})
