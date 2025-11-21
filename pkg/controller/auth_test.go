//go:build unit

package controller_test

import (
	"net/http"
	"net/http/httptest"

	"github.com/animalet/sargantana-go/pkg/config"
	"github.com/animalet/sargantana-go/pkg/controller"
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

// MockProviderFactory implements controller.ProvidersFactory
type MockProviderFactory struct{}

func (m *MockProviderFactory) CreateProviders(callbackURLTemplate string) []goth.Provider {
	return []goth.Provider{&MockProvider{name: "test-provider"}}
}

var _ = Describe("Auth Controller", func() {
	var (
		mockFactory *MockProviderFactory
		origFactory controller.ProvidersFactory
	)

	BeforeEach(func() {
		mockFactory = &MockProviderFactory{}
		origFactory = controller.ProviderFactory
		controller.ProviderFactory = mockFactory
		gin.SetMode(gin.TestMode)
	})

	AfterEach(func() {
		controller.ProviderFactory = origFactory
	})

	Context("Configuration", func() {
		It("should validate correct configuration", func() {
			cfg := controller.AuthControllerConfig{
				CallbackPath:     "/auth/callback",
				LoginPath:        "/auth/login",
				LogoutPath:       "/auth/logout",
				UserInfoPath:     "/auth/user",
				RedirectOnLogin:  "/",
				RedirectOnLogout: "/",
				Providers: map[string]controller.ProviderConfig{
					"test": {Key: "key", Secret: "secret"},
				},
			}
			Expect(cfg.Validate()).To(Succeed())
		})

		It("should fail validation on missing fields", func() {
			cfg := controller.AuthControllerConfig{}
			Expect(cfg.Validate()).To(HaveOccurred())
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

			ctrl, err := controller.NewAuthController(rawConfig, ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(ctrl).NotTo(BeNil())
		})
	})

	Context("Handlers", func() {
		var (
			engine *gin.Engine
			store  sessions.Store
		)

		BeforeEach(func() {
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
			ctrl, _ := controller.NewAuthController(rawConfig, ctx)
			ctrl.Bind(engine)
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
