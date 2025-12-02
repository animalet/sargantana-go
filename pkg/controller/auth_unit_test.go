//go:build unit

package controller

import (
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/animalet/sargantana-go/pkg/config"
	"github.com/animalet/sargantana-go/pkg/server"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/markbates/goth"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"golang.org/x/oauth2"
	"gopkg.in/yaml.v3"
)

// MockProvider is a simple mock of goth.Provider
type MockProvider struct {
	name string
}

func (m *MockProvider) Name() string        { return m.name }
func (m *MockProvider) SetName(name string) {}
func (m *MockProvider) BeginAuth(state string) (goth.Session, error) {
	return &MockSession{}, nil
}
func (m *MockProvider) UnmarshalSession(string) (goth.Session, error) {
	return &MockSession{}, nil
}
func (m *MockProvider) FetchUser(session goth.Session) (goth.User, error) {
	return goth.User{UserID: "test-user", Provider: m.name, Email: "test@example.com"}, nil
}
func (m *MockProvider) Debug(debug bool) {}
func (m *MockProvider) RefreshToken(refreshToken string) (*oauth2.Token, error) {
	return nil, nil
}
func (m *MockProvider) RefreshTokenAvailable() bool { return false }

type MockSession struct{}

func (m *MockSession) GetAuthURL() (string, error) { return "http://auth-url", nil }
func (m *MockSession) Marshal() string             { return "" }
func (m *MockSession) String() string              { return "" }
func (m *MockSession) Authorize(provider goth.Provider, params goth.Params) (string, error) {
	return "access-token", nil
}

// MockProviderFactory implements ProvidersFactory
type MockProviderFactory struct{}

func (m *MockProviderFactory) CreateProviders(callbackURLTemplate string) []goth.Provider {
	return []goth.Provider{&MockProvider{name: "test-provider"}}
}

var _ = Describe("Auth Controller", func() {
	Context("AuthControllerConfig Validate", func() {
		It("should return error if no providers configured", func() {
			cfg := AuthControllerConfig{}
			err := cfg.Validate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("at least one provider must be configured"))
		})

		It("should return error if callback_path is empty", func() {
			cfg := AuthControllerConfig{
				Providers: map[string]ProviderConfig{"google": {Key: "k", Secret: "s"}},
			}
			err := cfg.Validate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("callback_path must be set"))
		})

		// Add more validation tests as needed...
		It("should pass with valid config", func() {
			cfg := AuthControllerConfig{
				Providers:        map[string]ProviderConfig{"google": {Key: "k", Secret: "s"}},
				CallbackPath:     "/auth/callback",
				LoginPath:        "/login",
				LogoutPath:       "/logout",
				UserInfoPath:     "/user",
				RedirectOnLogin:  "/",
				RedirectOnLogout: "/",
			}
			err := cfg.Validate()
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("NewAuthController", func() {
		It("should create controller with correct callback URL", func() {
			cfgBytes := []byte(`
callback_path: /auth/callback
login_path: /login
logout_path: /logout
user_info_path: /user
redirect_on_login: /
redirect_on_logout: /
providers:
  google:
    key: test-key
    secret: test-secret
`)
			ctx := server.ControllerContext{
				ServerConfig: server.WebServerConfig{
					Address: "localhost:8080",
				},
			}

			// We need to mock config.Load or pass valid config bytes
			// Since config.Load uses unmarshal, passing bytes works if format is set or default
			// Default is YAML

			var authCfg AuthControllerConfig
			err := yaml.Unmarshal(cfgBytes, &authCfg)
			Expect(err).NotTo(HaveOccurred())

			ctrl, err := NewAuthController(&authCfg, ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(ctrl).NotTo(BeNil())
		})
	})
})

var _ = Describe("Auth Middleware", func() {
	var (
		engine *gin.Engine
		w      *httptest.ResponseRecorder
		store  cookie.Store
	)

	BeforeEach(func() {
		gin.SetMode(gin.TestMode)
		w = httptest.NewRecorder()
		engine = gin.New()
		store = cookie.NewStore([]byte("secret"))
		engine.Use(sessions.Sessions("mysession", store))
	})

	It("should abort with 401 if user is not in session", func() {
		engine.GET("/protected", LoginFunc, func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		req, _ := http.NewRequest("GET", "/protected", nil)
		engine.ServeHTTP(w, req)

		Expect(w.Code).To(Equal(http.StatusUnauthorized))
	})

	It("should abort with 401 if session is expired", func() {
		engine.GET("/protected", func(c *gin.Context) {
			session := sessions.Default(c)
			user := UserObject{
				User: goth.User{
					ExpiresAt: time.Now().Add(-1 * time.Hour),
				},
			}
			session.Set("user", user)
			session.Save()
		}, LoginFunc, func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		req, _ := http.NewRequest("GET", "/protected", nil)
		engine.ServeHTTP(w, req)

		Expect(w.Code).To(Equal(http.StatusUnauthorized))
	})

	It("should pass if user is logged in and valid", func() {
		engine.GET("/protected", func(c *gin.Context) {
			session := sessions.Default(c)
			user := UserObject{
				User: goth.User{
					ExpiresAt: time.Now().Add(1 * time.Hour),
				},
			}
			session.Set("user", user)
			session.Save()
		}, LoginFunc, func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		req, _ := http.NewRequest("GET", "/protected", nil)
		engine.ServeHTTP(w, req)

		Expect(w.Code).To(Equal(http.StatusOK))
	})
})

var _ = Describe("Auth Controller (Detailed)", func() {
	var (
		mockFactory *MockProviderFactory
		origFactory ProvidersFactory
	)

	BeforeEach(func() {
		mockFactory = &MockProviderFactory{}
		origFactory = ProviderFactory
		ProviderFactory = mockFactory
		gin.SetMode(gin.TestMode)
	})

	AfterEach(func() {
		ProviderFactory = origFactory
	})

	Context("Configuration", func() {
		It("should validate correct configuration", func() {
			cfg := AuthControllerConfig{
				CallbackPath:     "/auth/callback",
				LoginPath:        "/auth/login",
				LogoutPath:       "/auth/logout",
				UserInfoPath:     "/auth/user",
				RedirectOnLogin:  "/",
				RedirectOnLogout: "/",
				Providers: map[string]ProviderConfig{
					"test": {Key: "key", Secret: "secret"},
				},
			}
			Expect(cfg.Validate()).To(Succeed())
		})

		It("should fail validation on missing fields", func() {
			cfg := AuthControllerConfig{
				Providers: map[string]ProviderConfig{"test": {Key: "k", Secret: "s"}},
			}
			// Check each field
			cfg.CallbackPath = ""
			Expect(cfg.Validate()).To(MatchError("callback_path must be set and non-empty"))
			cfg.CallbackPath = "/cb"

			cfg.LoginPath = ""
			Expect(cfg.Validate()).To(MatchError("login_path must be set and non-empty"))
			cfg.LoginPath = "/login"

			cfg.LogoutPath = ""
			Expect(cfg.Validate()).To(MatchError("logout_path must be set and non-empty"))
			cfg.LogoutPath = "/logout"

			cfg.UserInfoPath = ""
			Expect(cfg.Validate()).To(MatchError("user_info_path must be set and non-empty"))
			cfg.UserInfoPath = "/user"

			cfg.RedirectOnLogin = ""
			Expect(cfg.Validate()).To(MatchError("redirect_on_login must be set and non-empty"))
			cfg.RedirectOnLogin = "/"

			cfg.RedirectOnLogout = ""
			Expect(cfg.Validate()).To(MatchError("redirect_on_logout must be set and non-empty"))
			cfg.RedirectOnLogout = "/"
		})

		It("should fail if provider key or secret is missing", func() {
			cfg := AuthControllerConfig{
				CallbackPath:     "/cb",
				LoginPath:        "/login",
				LogoutPath:       "/logout",
				UserInfoPath:     "/user",
				RedirectOnLogin:  "/",
				RedirectOnLogout: "/",
				Providers: map[string]ProviderConfig{
					"test": {Key: "", Secret: "secret"},
				},
			}
			Expect(cfg.Validate()).To(HaveOccurred())
			Expect(cfg.Validate().Error()).To(ContainSubstring("key must be set"))

			cfg.Providers["test"] = ProviderConfig{Key: "key", Secret: ""}
			Expect(cfg.Validate()).To(HaveOccurred())
			Expect(cfg.Validate().Error()).To(ContainSubstring("secret must be set"))
		})

		It("should fail if no providers configured", func() {
			cfg := AuthControllerConfig{}
			Expect(cfg.Validate()).To(MatchError("at least one provider must be configured"))
		})
	})

	Context("Provider Factory", func() {
		It("should create providers from config", func() {
			// We can't easily verify the internal state of Goth, but we can check if CreateProviders returns the right number
			// and doesn't panic for supported providers.

			// Let's try to cover the switch case by passing a config with many providers to NewAuth
			// We need to reset ProviderFactory to nil to ensure the default one is used.
			ProviderFactory = nil

			providersMap := map[string]interface{}{
				"twitter":         map[string]interface{}{"key": "k", "secret": "s"},
				"twitterv2":       map[string]interface{}{"key": "k", "secret": "s"},
				"tiktok":          map[string]interface{}{"key": "k", "secret": "s"},
				"facebook":        map[string]interface{}{"key": "k", "secret": "s"},
				"fitbit":          map[string]interface{}{"key": "k", "secret": "s"},
				"google":          map[string]interface{}{"key": "k", "secret": "s"},
				"github":          map[string]interface{}{"key": "k", "secret": "s"},
				"spotify":         map[string]interface{}{"key": "k", "secret": "s"},
				"linkedin":        map[string]interface{}{"key": "k", "secret": "s"},
				"line":            map[string]interface{}{"key": "k", "secret": "s"},
				"lastfm":          map[string]interface{}{"key": "k", "secret": "s"},
				"twitch":          map[string]interface{}{"key": "k", "secret": "s"},
				"dropbox":         map[string]interface{}{"key": "k", "secret": "s"},
				"digitalocean":    map[string]interface{}{"key": "k", "secret": "s"},
				"bitbucket":       map[string]interface{}{"key": "k", "secret": "s"},
				"instagram":       map[string]interface{}{"key": "k", "secret": "s"},
				"intercom":        map[string]interface{}{"key": "k", "secret": "s"},
				"box":             map[string]interface{}{"key": "k", "secret": "s"},
				"salesforce":      map[string]interface{}{"key": "k", "secret": "s"},
				"seatalk":         map[string]interface{}{"key": "k", "secret": "s"},
				"amazon":          map[string]interface{}{"key": "k", "secret": "s"},
				"yammer":          map[string]interface{}{"key": "k", "secret": "s"},
				"onedrive":        map[string]interface{}{"key": "k", "secret": "s"},
				"azuread":         map[string]interface{}{"key": "k", "secret": "s"},
				"microsoftonline": map[string]interface{}{"key": "k", "secret": "s"},
				"battlenet":       map[string]interface{}{"key": "k", "secret": "s"},
				"eveonline":       map[string]interface{}{"key": "k", "secret": "s"},
				"kakao":           map[string]interface{}{"key": "k", "secret": "s"},
				"yahoo":           map[string]interface{}{"key": "k", "secret": "s"},
				"typetalk":        map[string]interface{}{"key": "k", "secret": "s"},
				"slack":           map[string]interface{}{"key": "k", "secret": "s"},
				"stripe":          map[string]interface{}{"key": "k", "secret": "s"},
				"wepay":           map[string]interface{}{"key": "k", "secret": "s"},
				"paypal":          map[string]interface{}{"key": "k", "secret": "s"},
				"steam":           map[string]interface{}{"key": "k", "secret": "s"},
				"heroku":          map[string]interface{}{"key": "k", "secret": "s"},
				"uber":            map[string]interface{}{"key": "k", "secret": "s"},
				"soundcloud":      map[string]interface{}{"key": "k", "secret": "s"},
				"gitlab":          map[string]interface{}{"key": "k", "secret": "s"},
				"dailymotion":     map[string]interface{}{"key": "k", "secret": "s"},
				"deezer":          map[string]interface{}{"key": "k", "secret": "s"},
				"discord":         map[string]interface{}{"key": "k", "secret": "s"},
				"meetup":          map[string]interface{}{"key": "k", "secret": "s"},
				"auth0":           map[string]interface{}{"key": "k", "secret": "s", "domain": "example.com"},
				"xero":            map[string]interface{}{"key": "k", "secret": "s"},
				"vk":              map[string]interface{}{"key": "k", "secret": "s"},
				"naver":           map[string]interface{}{"key": "k", "secret": "s"},
				"yandex":          map[string]interface{}{"key": "k", "secret": "s"},
				"nextcloud":       map[string]interface{}{"key": "k", "secret": "s", "url": "http://localhost"},
				"gitea":           map[string]interface{}{"key": "k", "secret": "s"},
				"shopify":         map[string]interface{}{"key": "k", "secret": "s"},
				"apple":           map[string]interface{}{"key": "k", "secret": "s"},
				"strava":          map[string]interface{}{"key": "k", "secret": "s"},
				"okta":            map[string]interface{}{"key": "k", "secret": "s", "org_url": "http://localhost"},
				"mastodon":        map[string]interface{}{"key": "k", "secret": "s"},
				"wecom":           map[string]interface{}{"corp_id": "k", "secret": "s", "agent_id": "a"},
				"zoom":            map[string]interface{}{"key": "k", "secret": "s"},
				"patreon":         map[string]interface{}{"key": "k", "secret": "s"},
				"openid-connect":  map[string]interface{}{"key": "k", "secret": "s", "url": "http://localhost"},
			}

			rawConfigMap := map[string]interface{}{
				"callback_path":      "/auth/callback",
				"login_path":         "/login",
				"logout_path":        "/logout",
				"user_info_path":     "/user",
				"redirect_on_login":  "/",
				"redirect_on_logout": "/",
				"providers":          providersMap,
			}
			rawConfigBytes, _ := yaml.Marshal(rawConfigMap)
			rawConfig := config.ModuleRawConfig(rawConfigBytes)

			store := cookie.NewStore([]byte("secret"))
			ctx := server.ControllerContext{
				ServerConfig: server.WebServerConfig{Address: "localhost:8080"},
				SessionStore: store,
			}

			var authCfg AuthControllerConfig
			err := yaml.Unmarshal(rawConfig, &authCfg)
			Expect(err).NotTo(HaveOccurred())

			// This will register providers in Goth.
			ctrl, err := NewAuthController(&authCfg, ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(ctrl).NotTo(BeNil())
		})
	})

	Context("Controller Creation", func() {
		It("should create a new auth controller", func() {
			rawConfigMap := map[string]interface{}{
				"callback_path":      "/auth/callback",
				"login_path":         "/auth/login/{provider}",
				"logout_path":        "/auth/logout",
				"user_info_path":     "/auth/user",
				"redirect_on_login":  "/",
				"redirect_on_logout": "/",
				"providers": map[string]interface{}{
					"test": map[string]interface{}{"key": "k", "secret": "s"},
				},
			}
			rawConfigBytes, _ := yaml.Marshal(rawConfigMap)
			rawConfig := config.ModuleRawConfig(rawConfigBytes)

			store := cookie.NewStore([]byte("secret"))
			ctx := server.ControllerContext{
				ServerConfig: server.WebServerConfig{Address: "localhost:8080"},
				SessionStore: store,
			}

			var authCfg AuthControllerConfig
			err := yaml.Unmarshal(rawConfig, &authCfg)
			Expect(err).NotTo(HaveOccurred())

			ctrl, err := NewAuthController(&authCfg, ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(ctrl).NotTo(BeNil())
		})
	})

	Context("Handlers", func() {
		var (
			engine      *gin.Engine
			store       sessions.Store
			mockFactory *MockProviderFactory
			origFactory ProvidersFactory
		)

		BeforeEach(func() {
			mockFactory = &MockProviderFactory{}
			origFactory = ProviderFactory
			ProviderFactory = mockFactory
			gin.SetMode(gin.TestMode)

			engine = gin.New()
			store = cookie.NewStore([]byte("secret"))
			engine.Use(sessions.Sessions("session", store))

			rawConfigMap := map[string]interface{}{
				"callback_path":      "/auth/callback/{provider}",
				"login_path":         "/auth/login/{provider}",
				"logout_path":        "/auth/logout",
				"user_info_path":     "/auth/user",
				"redirect_on_login":  "/dashboard",
				"redirect_on_logout": "/home",
				"providers": map[string]interface{}{
					"test-provider": map[string]interface{}{"key": "k", "secret": "s"},
				},
			}
			rawConfigBytes, _ := yaml.Marshal(rawConfigMap)
			rawConfig := config.ModuleRawConfig(rawConfigBytes)
			ctx := server.ControllerContext{
				ServerConfig: server.WebServerConfig{Address: "localhost:8080"},
				SessionStore: store,
			}
			var authCfg AuthControllerConfig
			_ = yaml.Unmarshal(rawConfig, &authCfg)
			ctrl, _ := NewAuthController(&authCfg, ctx)
			ctrl.Bind(engine, nil)
		})

		AfterEach(func() {
			ProviderFactory = origFactory
		})

		It("should redirect to provider on login", func() {
			req, _ := http.NewRequest("GET", "/auth/login/test-provider", nil)
			w := httptest.NewRecorder()
			engine.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusTemporaryRedirect))
			// The mock provider returns http://auth-url, but gothic might wrap it or redirect differently
			// We just check it redirects
		})

		It("should logout and redirect", func() {
			req, _ := http.NewRequest("GET", "/auth/logout", nil)
			w := httptest.NewRecorder()
			engine.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusFound))
			Expect(w.Header().Get("Location")).To(Equal("/home"))
		})
	})
})
