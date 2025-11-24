//go:build integration

package controller

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"

	"github.com/animalet/sargantana-go/pkg/server"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v3"
)

var _ = Describe("Auth Controller Integration", func() {
	var (
		ts       *httptest.Server
		client   *http.Client
		authCtrl server.IController
		engine   *gin.Engine
	)

	BeforeEach(func() {
		// 1. Setup listener to get a free port
		l, err := net.Listen("tcp", "127.0.0.1:0")
		Expect(err).NotTo(HaveOccurred())
		port := l.Addr().(*net.TCPAddr).Port
		baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)

		// 2. Prepare Config
		cfgMap := map[string]interface{}{
			"callback_host":      baseURL,
			"callback_path":      "/auth/{provider}/callback",
			"login_path":         "/login/{provider}",
			"logout_path":        "/logout",
			"user_info_path":     "/user-info",
			"redirect_on_login":  "/dashboard",
			"redirect_on_logout": "/",
			"providers": map[string]interface{}{
				"openid-connect": map[string]interface{}{
					"key":    "sargantana",
					"secret": "test-secret",
					"url":    "http://localhost:8081/default/.well-known/openid-configuration", // Mock server URL
					"scopes": []string{"openid", "profile", "email"},
				},
			},
		}
		cfgBytes, err := yaml.Marshal(cfgMap)
		Expect(err).NotTo(HaveOccurred())

		// 3. Setup Session Store
		store := cookie.NewStore([]byte("secret"))
		store.Options(sessions.Options{
			Path:     "/",
			HttpOnly: true,
			Secure:   false,
			MaxAge:   86400 * 30,
		})

		// 4. Create Controller
		ctx := server.ControllerContext{
			ServerConfig: server.WebServerConfig{
				Address: fmt.Sprintf(":%d", port),
			},
			SessionStore: store,
		}
		var authCfg AuthControllerConfig
		err = yaml.Unmarshal(cfgBytes, &authCfg)
		Expect(err).NotTo(HaveOccurred())

		authCtrl, err = NewAuthController(&authCfg, ctx)
		Expect(err).NotTo(HaveOccurred())

		// 5. Setup Gin Engine
		gin.SetMode(gin.ReleaseMode)
		engine = gin.New()
		engine.Use(gin.Recovery())
		engine.Use(sessions.Sessions("mysession", store))

		// Bind controller
		authCtrl.Bind(engine)

		// Add dummy dashboard handler to verify redirect
		engine.GET("/dashboard", func(c *gin.Context) {
			c.String(http.StatusOK, "Dashboard")
		})

		// 6. Start Server with the pre-allocated listener
		ts = httptest.NewUnstartedServer(engine)
		ts.Listener.Close()
		ts.Listener = l
		ts.Start()

		// 7. Setup Client with Cookie Jar
		jar, _ := cookiejar.New(nil)
		client = &http.Client{
			Jar: jar,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse // Don't follow redirects automatically
			},
		}
	})

	AfterEach(func() {
		if ts != nil {
			ts.Close()
		}
	})

	It("should perform full OIDC authentication flow", func() {

		// Step 1: Initiate Login
		resp, err := client.Get(ts.URL + "/login/openid-connect")
		Expect(err).NotTo(HaveOccurred())
		defer resp.Body.Close()

		// Expect Redirect to Identity Provider
		Expect(resp.StatusCode).To(SatisfyAny(Equal(http.StatusFound), Equal(http.StatusTemporaryRedirect)))
		location, err := resp.Location()
		Expect(err).NotTo(HaveOccurred())
		Expect(location.Host).To(ContainSubstring("localhost:8081"))
		Expect(location.Path).To(ContainSubstring("/default/authorize"))

		// Capture 'state' parameter
		state := location.Query().Get("state")
		Expect(state).NotTo(BeEmpty())

		// Step 2: Follow Redirect to IDP
		// The mock server (interactiveLogin=false) should immediately redirect back with a code
		resp, err = client.Get(location.String())
		Expect(err).NotTo(HaveOccurred())
		defer resp.Body.Close()

		// Expect Redirect back to Callback
		Expect(resp.StatusCode).To(SatisfyAny(Equal(http.StatusFound), Equal(http.StatusTemporaryRedirect), Equal(http.StatusSeeOther)))
		location, err = resp.Location()
		Expect(err).NotTo(HaveOccurred())

		// Verify we are back at the callback
		Expect(location.Path).To(Equal("/auth/openid-connect/callback"))
		code := location.Query().Get("code")
		Expect(code).NotTo(BeEmpty())

		// Step 3: Follow Redirect to Callback (Code Exchange)
		resp, err = client.Get(location.String())
		Expect(err).NotTo(HaveOccurred())
		defer resp.Body.Close()

		// Expect Redirect to Identity Provider
		Expect(resp.StatusCode).To(SatisfyAny(Equal(http.StatusFound), Equal(http.StatusTemporaryRedirect), Equal(http.StatusSeeOther)))
		location, err = resp.Location()
		Expect(err).NotTo(HaveOccurred())
		Expect(location.Path).To(Equal("/dashboard"))

		// Step 4: Verify Session / User Info
		// Follow the redirect to dashboard (optional, just to prove cookie works)
		resp, err = client.Get(ts.URL + "/dashboard")
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(http.StatusOK))

		// Check User Info endpoint
		resp, err = client.Get(ts.URL + "/user-info")
		Expect(err).NotTo(HaveOccurred())
		defer resp.Body.Close()

		Expect(resp.StatusCode).To(Equal(http.StatusOK))

		body, err := io.ReadAll(resp.Body)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(body)).To(ContainSubstring("test-user"))
	})
})
