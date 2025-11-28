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

var _ = Describe("Static Controller Auth Integration", func() {
	var (
		ts            *httptest.Server
		client        *http.Client
		listener      net.Listener
		engine        *gin.Engine
		authenticator server.Authenticator
		store         sessions.Store
		sessionSecret = "test-secret-key-32-bytes-long!!"
	)

	BeforeEach(func() {
		var err error
		gin.SetMode(gin.TestMode)

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

	Context("Static File Serving without Authentication", func() {
		BeforeEach(func() {
			cfg := &StaticControllerConfig{
				Path: "/testfile",
				File: "./testdata/test.txt",
				Auth: false,
			}

			ctrl, err := NewStaticController(cfg, server.ControllerContext{})
			Expect(err).NotTo(HaveOccurred())
			ctrl.Bind(engine, authenticator.Middleware())
		})

		It("should serve static file without authentication", func() {
			resp, err := client.Get(fmt.Sprintf("%s/testfile", ts.URL))
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			defer resp.Body.Close()
		})
	})

	Context("Static Directory Serving without Authentication", func() {
		BeforeEach(func() {
			cfg := &StaticControllerConfig{
				Path: "/static",
				Dir:  "./testdata",
				Auth: false,
			}

			ctrl, err := NewStaticController(cfg, server.ControllerContext{})
			Expect(err).NotTo(HaveOccurred())
			ctrl.Bind(engine, authenticator.Middleware())
		})

		It("should serve files from directory without authentication", func() {
			resp, err := client.Get(fmt.Sprintf("%s/static/test.txt", ts.URL))
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			defer resp.Body.Close()
		})
	})

	Context("Static File Serving with Authentication", func() {
		BeforeEach(func() {
			// Get port for callback URL
			port := listener.Addr().(*net.TCPAddr).Port
			baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)

			// Setup auth controller with real OpenID Connect provider
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

			// Setup static file controller with auth
			staticCfg := &StaticControllerConfig{
				Path: "/secure-file",
				File: "./testdata/test.txt",
				Auth: true,
			}
			staticCtrl, err := NewStaticController(staticCfg, server.ControllerContext{})
			Expect(err).NotTo(HaveOccurred())
			staticCtrl.Bind(engine, authenticator.Middleware())
		})

		It("should return 401 when accessing without authentication", func() {
			resp, err := client.Get(fmt.Sprintf("%s/secure-file", ts.URL))
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusUnauthorized))
		})

		It("should serve file after successful authentication", func() {
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

			// Step 2: Follow redirect to IDP
			resp, err = client.Get(location.String())
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()
			Expect(resp.StatusCode).To(SatisfyAny(Equal(http.StatusFound), Equal(http.StatusTemporaryRedirect), Equal(http.StatusSeeOther)))

			location, err = resp.Location()
			Expect(err).NotTo(HaveOccurred())

			// Step 3: Follow redirect to callback
			resp, err = client.Get(location.String())
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			// Access protected file
			resp, err = client.Get(fmt.Sprintf("%s/secure-file", ts.URL))
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			defer resp.Body.Close()
		})

		It("should return 401 when session expires", func() {
			// Create expired session
			engine.GET("/test-expired-session", func(c *gin.Context) {
				session := sessions.Default(c)
				expiredUser := UserObject{
					User: goth.User{
						UserID:    "test-user",
						Provider:  "openid-connect",
						Email:     "test@example.com",
						ExpiresAt: time.Now().Add(-1 * time.Hour),
					},
				}
				session.Set("user", expiredUser)
				_ = session.Save()
				c.Status(http.StatusOK)
			})

			resp, err := client.Get(fmt.Sprintf("%s/test-expired-session", ts.URL))
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			// Try to access protected file with expired session
			resp, err = client.Get(fmt.Sprintf("%s/secure-file", ts.URL))
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusUnauthorized))
		})
	})

	Context("Static Directory Serving with Authentication", func() {
		BeforeEach(func() {
			// Get port for callback URL
			port := listener.Addr().(*net.TCPAddr).Port
			baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)

			// Setup auth controller with real OpenID Connect provider
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

			// Setup static directory controller with auth
			staticCfg := &StaticControllerConfig{
				Path: "/secure-static",
				Dir:  "./testdata",
				Auth: true,
			}
			staticCtrl, err := NewStaticController(staticCfg, server.ControllerContext{})
			Expect(err).NotTo(HaveOccurred())
			staticCtrl.Bind(engine, authenticator.Middleware())
		})

		It("should return 401 when accessing directory without authentication", func() {
			resp, err := client.Get(fmt.Sprintf("%s/secure-static/test.txt", ts.URL))
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusUnauthorized))
		})

		It("should serve directory files after successful authentication", func() {
			// Step 1: Initiate login
			resp, err := client.Get(fmt.Sprintf("%s/auth/openid-connect", ts.URL))
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			location, err := resp.Location()
			Expect(err).NotTo(HaveOccurred())

			// Step 2: Follow to IDP
			resp, err = client.Get(location.String())
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			location, err = resp.Location()
			Expect(err).NotTo(HaveOccurred())

			// Step 3: Follow to callback
			resp, err = client.Get(location.String())
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			// Access protected directory file
			resp, err = client.Get(fmt.Sprintf("%s/secure-static/test.txt", ts.URL))
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			defer resp.Body.Close()
		})

		It("should return 401 after logout", func() {
			// Complete OAuth flow
			resp, err := client.Get(fmt.Sprintf("%s/auth/openid-connect", ts.URL))
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			location, err := resp.Location()
			Expect(err).NotTo(HaveOccurred())

			resp, err = client.Get(location.String())
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			location, err = resp.Location()
			Expect(err).NotTo(HaveOccurred())

			resp, err = client.Get(location.String())
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			// Verify access works
			resp, err = client.Get(fmt.Sprintf("%s/secure-static/test.txt", ts.URL))
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			// Logout
			resp, err = client.Get(fmt.Sprintf("%s/auth/logout", ts.URL))
			Expect(err).NotTo(HaveOccurred())

			// Try to access after logout
			resp, err = client.Get(fmt.Sprintf("%s/secure-static/test.txt", ts.URL))
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusUnauthorized))
		})
	})
})
