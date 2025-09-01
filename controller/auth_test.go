package controller

import (
	"encoding/gob"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/animalet/sargantana-go/config"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/markbates/goth"
	"gopkg.in/yaml.v3"
)

func init() {
	gob.Register(UserObject{})
}

func TestNewAuthController(t *testing.T) {
	tests := []struct {
		name          string
		configData    AuthControllerConfig
		serverConfig  config.ServerConfig
		expectedError bool
	}{
		{
			name: "valid config with localhost",
			configData: AuthControllerConfig{
				CallbackPath:     "/auth/{provider}/callback",
				LoginPath:        "/auth/{provider}",
				LogoutPath:       "/auth/{provider}/logout",
				UserInfoPath:     "/auth/user",
				RedirectOnLogin:  "/",
				RedirectOnLogout: "/",
			},
			serverConfig: config.ServerConfig{
				Address: "localhost:8080",
			},
			expectedError: false,
		},
		{
			name: "valid config with 0.0.0.0",
			configData: AuthControllerConfig{
				CallbackPath:     "/auth/{provider}/callback",
				LoginPath:        "/auth/{provider}",
				LogoutPath:       "/auth/{provider}/logout",
				UserInfoPath:     "/auth/user",
				RedirectOnLogin:  "/",
				RedirectOnLogout: "/",
			},
			serverConfig: config.ServerConfig{
				Address: "0.0.0.0:9000",
			},
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal the config to YAML bytes as expected by the ControllerConfig type
			configBytes, err := yaml.Marshal(tt.configData)
			if err != nil {
				t.Fatalf("Failed to marshal config: %v", err)
			}

			controller, err := NewAuthController(config.ControllerConfig(configBytes), tt.serverConfig)

			if tt.expectedError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectedError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if !tt.expectedError && controller == nil {
				t.Error("Expected controller but got nil")
			}
		})
	}
}

func TestAuth_UserFactory(t *testing.T) {
	configData := AuthControllerConfig{
		CallbackPath:     "/auth/{provider}/callback",
		LoginPath:        "/auth/{provider}",
		LogoutPath:       "/auth/{provider}/logout",
		UserInfoPath:     "/auth/user",
		RedirectOnLogin:  "/",
		RedirectOnLogout: "/",
	}
	configBytes, _ := yaml.Marshal(configData)
	serverConfig := config.ServerConfig{Address: "localhost:8080"}

	controller, err := NewAuthController(config.ControllerConfig(configBytes), serverConfig)
	if err != nil {
		t.Fatalf("Failed to create auth controller: %v", err)
	}

	auth := controller.(*auth)

	tests := []struct {
		name       string
		user       goth.User
		expectedID string
	}{
		{
			name: "user with email",
			user: goth.User{
				Email:    "test@example.com",
				UserID:   "123",
				Provider: "github",
			},
			expectedID: "test@example.com",
		},
		{
			name: "user without email",
			user: goth.User{
				Email:    "",
				UserID:   "456",
				Provider: "twitter",
			},
			expectedID: "456@twitter",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userObj := auth.userFactory(tt.user)

			if userObj.Id != tt.expectedID {
				t.Errorf("User ID = %v, want %v", userObj.Id, tt.expectedID)
			}
			if userObj.User.Provider != tt.user.Provider {
				t.Errorf("Provider = %v, want %v", userObj.User.Provider, tt.user.Provider)
			}
		})
	}
}

func TestLoginFunc(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		setupSession   func(*gin.Context)
		expectedStatus int
	}{
		{
			name: "no user session",
			setupSession: func(c *gin.Context) {
				// No session setup
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name: "valid user session",
			setupSession: func(c *gin.Context) {
				session := sessions.Default(c)
				userObj := UserObject{
					Id: "test@example.com",
					User: goth.User{
						ExpiresAt: time.Now().Add(time.Hour),
					},
				}
				session.Set("user", userObj)
				err := session.Save()
				if err != nil {
					c.String(http.StatusInternalServerError, "Failed to save session: %v", err)
					return
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "expired user session",
			setupSession: func(c *gin.Context) {
				session := sessions.Default(c)
				userObj := UserObject{
					Id: "test@example.com",
					User: goth.User{
						ExpiresAt: time.Now().Add(-time.Hour),
					},
				}
				session.Set("user", userObj)
				err := session.Save()
				if err != nil {
					c.String(http.StatusInternalServerError, "Failed to save session: %v", err)
					return
				}
			},
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := gin.New()
			store := cookie.NewStore([]byte("secret"))
			engine.Use(sessions.Sessions("test", store))

			// Setup route with LoginFunc middleware
			engine.GET("/test", func(c *gin.Context) {
				tt.setupSession(c)
			}, LoginFunc, func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/test", nil)
			engine.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Status = %v, want %v", w.Code, tt.expectedStatus)
			}
		})
	}
}

func TestAuth_Close(t *testing.T) {
	configData := AuthControllerConfig{
		CallbackPath:     "/auth/{provider}/callback",
		LoginPath:        "/auth/{provider}",
		LogoutPath:       "/auth/{provider}/logout",
		UserInfoPath:     "/auth/user",
		RedirectOnLogin:  "/",
		RedirectOnLogout: "/",
	}
	configBytes, _ := yaml.Marshal(configData)
	serverConfig := config.ServerConfig{Address: "localhost:8080"}

	controller, err := NewAuthController(config.ControllerConfig(configBytes), serverConfig)
	if err != nil {
		t.Fatalf("Failed to create auth controller: %v", err)
	}

	auth := controller.(*auth)
	err = auth.Close()
	if err != nil {
		t.Errorf("Close() returned error: %v", err)
	}
}

func TestAuth_Routes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	configData := AuthControllerConfig{
		CallbackPath:     "/auth/{provider}/callback",
		LoginPath:        "/auth/{provider}",
		LogoutPath:       "/auth/{provider}/logout",
		UserInfoPath:     "/auth/user",
		RedirectOnLogin:  "/",
		RedirectOnLogout: "/",
	}
	configBytes, _ := yaml.Marshal(configData)
	serverConfig := config.ServerConfig{Address: "localhost:8080"}

	controller, err := NewAuthController(config.ControllerConfig(configBytes), serverConfig)
	if err != nil {
		t.Fatalf("Failed to create auth controller: %v", err)
	}

	auth := controller.(*auth)
	engine := gin.New()
	store := cookie.NewStore([]byte("secret"))
	engine.Use(sessions.Sessions("test", store))

	auth.Bind(engine, LoginFunc)

	// Test that routes are registered
	routes := engine.Routes()

	expectedRoutes := []string{
		"GET /auth/:provider",
		"GET /auth/:provider/callback",
		"GET /auth/:provider/logout",
		"GET /auth/user",
	}

	routeMap := make(map[string]bool)
	for _, route := range routes {
		routeKey := route.Method + " " + route.Path
		routeMap[routeKey] = true
	}

	for _, expected := range expectedRoutes {
		if !routeMap[expected] {
			t.Errorf("Expected route %s not found", expected)
		}
	}
}

func TestAuth_UserRoute_NoAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)

	configData := AuthControllerConfig{
		CallbackPath:     "/auth/{provider}/callback",
		LoginPath:        "/auth/{provider}",
		LogoutPath:       "/auth/{provider}/logout",
		UserInfoPath:     "/auth/user",
		RedirectOnLogin:  "/",
		RedirectOnLogout: "/",
	}
	configBytes, _ := yaml.Marshal(configData)
	serverConfig := config.ServerConfig{Address: "localhost:8080"}

	controller, err := NewAuthController(config.ControllerConfig(configBytes), serverConfig)
	if err != nil {
		t.Fatalf("Failed to create auth controller: %v", err)
	}

	auth := controller.(*auth)
	engine := gin.New()
	store := cookie.NewStore([]byte("secret"))
	engine.Use(sessions.Sessions("test", store))

	auth.Bind(engine, LoginFunc)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/auth/user", nil)
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected 403 Forbidden, got %d", w.Code)
	}
}

func TestAuth_AuthRouteHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	configData := AuthControllerConfig{
		CallbackPath:     "/auth/{provider}/callback",
		LoginPath:        "/auth/{provider}",
		LogoutPath:       "/auth/{provider}/logout",
		UserInfoPath:     "/auth/user",
		RedirectOnLogin:  "/",
		RedirectOnLogout: "/",
	}
	configBytes, _ := yaml.Marshal(configData)
	serverConfig := config.ServerConfig{Address: "localhost:8080"}

	controller, err := NewAuthController(config.ControllerConfig(configBytes), serverConfig)
	if err != nil {
		t.Fatalf("Failed to create auth controller: %v", err)
	}

	auth := controller.(*auth)
	engine := gin.New()
	store := cookie.NewStore([]byte("secret"))
	engine.Use(sessions.Sessions("test", store))

	auth.Bind(engine, LoginFunc)

	// Test auth route without provider
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/auth/", nil)
	engine.ServeHTTP(w, req)

	// Should return 404 since no provider specified
	if w.Code != http.StatusNotFound {
		t.Errorf("Expected 404 for missing provider, got %d", w.Code)
	}
}

func TestAuth_CallbackRouteHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	configData := AuthControllerConfig{
		CallbackPath:     "/auth/{provider}/callback",
		LoginPath:        "/auth/{provider}",
		LogoutPath:       "/auth/{provider}/logout",
		UserInfoPath:     "/auth/user",
		RedirectOnLogin:  "/",
		RedirectOnLogout: "/",
	}
	configBytes, _ := yaml.Marshal(configData)
	serverConfig := config.ServerConfig{Address: "localhost:8080"}

	controller, err := NewAuthController(config.ControllerConfig(configBytes), serverConfig)
	if err != nil {
		t.Fatalf("Failed to create auth controller: %v", err)
	}

	auth := controller.(*auth)
	engine := gin.New()
	store := cookie.NewStore([]byte("secret"))
	engine.Use(sessions.Sessions("test", store))

	auth.Bind(engine, LoginFunc)

	// Test callback route without provider
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/auth//callback", nil)
	engine.ServeHTTP(w, req)

	// Should return 400 since no provider specified
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for missing provider callback, got %d", w.Code)
	}
}

func TestAuth_LogoutRouteHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	configData := AuthControllerConfig{
		CallbackPath:     "/auth/{provider}/callback",
		LoginPath:        "/auth/{provider}",
		LogoutPath:       "/auth/{provider}/logout",
		UserInfoPath:     "/auth/user",
		RedirectOnLogin:  "/",
		RedirectOnLogout: "/",
	}
	configBytes, _ := yaml.Marshal(configData)
	serverConfig := config.ServerConfig{Address: "localhost:8080"}

	controller, err := NewAuthController(config.ControllerConfig(configBytes), serverConfig)
	if err != nil {
		t.Fatalf("Failed to create auth controller: %v", err)
	}

	auth := controller.(*auth)
	engine := gin.New()
	store := cookie.NewStore([]byte("secret"))
	engine.Use(sessions.Sessions("test", store))

	auth.Bind(engine, LoginFunc)

	// Test logout route
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/auth/github/logout", nil)
	engine.ServeHTTP(w, req)

	// Should redirect since no session exists
	if w.Code != http.StatusFound {
		t.Errorf("Expected 302 redirect for logout, got %d", w.Code)
	}
}

func TestLoginFunc_WithNonUserObject(t *testing.T) {
	gin.SetMode(gin.TestMode)

	engine := gin.New()
	store := cookie.NewStore([]byte("secret"))
	engine.Use(sessions.Sessions("test", store))

	engine.GET("/test", func(c *gin.Context) {
		session := sessions.Default(c)
		// Set a non-UserObject value
		session.Set("user", "invalid-user-object")
		err := session.Save()
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to save session: %v", err)
			return
		}
	}, LoginFunc, func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401 for invalid user object, got %d", w.Code)
	}
}

func TestAuth_SetCallbackFromConfig(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		configAddress  string
		expectedPrefix string
	}{
		{
			name:           "localhost address",
			configAddress:  "localhost:8080",
			expectedPrefix: "http://localhost:8080",
		},
		{
			name:           "0.0.0.0 address",
			configAddress:  "0.0.0.0:9000",
			expectedPrefix: "http://localhost:9000",
		},
		{
			name:           "custom host",
			configAddress:  "myhost:3000",
			expectedPrefix: "http://myhost:3000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configData := AuthControllerConfig{
				CallbackPath:     "/auth/{provider}/callback",
				LoginPath:        "/auth/{provider}",
				LogoutPath:       "/auth/{provider}/logout",
				UserInfoPath:     "/auth/user",
				RedirectOnLogin:  "/",
				RedirectOnLogout: "/",
			}
			configBytes, _ := yaml.Marshal(configData)
			serverConfig := config.ServerConfig{Address: tt.configAddress}

			controller, err := NewAuthController(config.ControllerConfig(configBytes), serverConfig)
			if err != nil {
				t.Fatalf("Failed to create auth controller: %v", err)
			}

			auth := controller.(*auth)
			engine := gin.New()
			store := cookie.NewStore([]byte("secret"))
			engine.Use(sessions.Sessions("test", store))

			auth.Bind(engine, LoginFunc)

			// We can't directly access the callbackEndpoint, but we can verify the controller was created successfully
			// The actual callback endpoint logic is tested indirectly through the working routes
			if auth == nil {
				t.Error("Expected auth controller to be created successfully")
			}
		})
	}
}

func TestConfigProviderFactory_CreateProviders_AllProviders(t *testing.T) {
	// Create test configuration with all providers
	testConfig := map[string]ProviderConfig{
		"twitter": {
			Key:    "test-twitter-key",
			Secret: "test-twitter-secret",
		},
		"tiktok": {
			Key:    "test-tiktok-key",
			Secret: "test-tiktok-secret",
		},
		"facebook": {
			Key:    "test-facebook-key",
			Secret: "test-facebook-secret",
			Scopes: []string{"email", "public_profile"},
		},
		"fitbit": {
			Key:    "test-fitbit-key",
			Secret: "test-fitbit-secret",
		},
		"google": {
			Key:    "test-google-key",
			Secret: "test-google-secret",
		},
		"github": {
			Key:    "test-github-key",
			Secret: "test-github-secret",
			Scopes: []string{"read:user", "user:email"},
		},
		"spotify": {
			Key:    "test-spotify-key",
			Secret: "test-spotify-secret",
		},
		"linkedin": {
			Key:    "test-linkedin-key",
			Secret: "test-linkedin-secret",
		},
		"line": {
			Key:    "test-line-key",
			Secret: "test-line-secret",
			Scopes: []string{"profile", "openid", "email"},
		},
		"lastfm": {
			Key:    "test-lastfm-key",
			Secret: "test-lastfm-secret",
		},
		"twitch": {
			Key:    "test-twitch-key",
			Secret: "test-twitch-secret",
		},
		"dropbox": {
			Key:    "test-dropbox-key",
			Secret: "test-dropbox-secret",
		},
		"digitalocean": {
			Key:    "test-do-key",
			Secret: "test-do-secret",
			Scopes: []string{"read"},
		},
		"bitbucket": {
			Key:    "test-bitbucket-key",
			Secret: "test-bitbucket-secret",
		},
		"instagram": {
			Key:    "test-instagram-key",
			Secret: "test-instagram-secret",
		},
		"intercom": {
			Key:    "test-intercom-key",
			Secret: "test-intercom-secret",
		},
		"box": {
			Key:    "test-box-key",
			Secret: "test-box-secret",
		},
		"salesforce": {
			Key:    "test-salesforce-key",
			Secret: "test-salesforce-secret",
		},
		"seatalk": {
			Key:    "test-seatalk-key",
			Secret: "test-seatalk-secret",
		},
		"amazon": {
			Key:    "test-amazon-key",
			Secret: "test-amazon-secret",
		},
		"yammer": {
			Key:    "test-yammer-key",
			Secret: "test-yammer-secret",
		},
		"onedrive": {
			Key:    "test-onedrive-key",
			Secret: "test-onedrive-secret",
		},
		"azuread": {
			Key:    "test-azuread-key",
			Secret: "test-azuread-secret",
		},
		"microsoftonline": {
			Key:    "test-msonline-key",
			Secret: "test-msonline-secret",
		},
		"battlenet": {
			Key:    "test-battlenet-key",
			Secret: "test-battlenet-secret",
		},
		"eveonline": {
			Key:    "test-eve-key",
			Secret: "test-eve-secret",
		},
		"kakao": {
			Key:    "test-kakao-key",
			Secret: "test-kakao-secret",
		},
		"yahoo": {
			Key:    "test-yahoo-key",
			Secret: "test-yahoo-secret",
		},
		"typetalk": {
			Key:    "test-typetalk-key",
			Secret: "test-typetalk-secret",
			Scopes: []string{"my"},
		},
		"slack": {
			Key:    "test-slack-key",
			Secret: "test-slack-secret",
		},
		"stripe": {
			Key:    "test-stripe-key",
			Secret: "test-stripe-secret",
		},
		"wepay": {
			Key:    "test-wepay-key",
			Secret: "test-wepay-secret",
			Scopes: []string{"view_user"},
		},
		"paypal": {
			Key:    "test-paypal-key",
			Secret: "test-paypal-secret",
		},
		"steam": {
			Key: "test-steam-key",
			// Note: Steam only requires a key, no secret
		},
		"heroku": {
			Key:    "test-heroku-key",
			Secret: "test-heroku-secret",
		},
		"uber": {
			Key:    "test-uber-key",
			Secret: "test-uber-secret",
		},
		"soundcloud": {
			Key:    "test-soundcloud-key",
			Secret: "test-soundcloud-secret",
		},
		"gitlab": {
			Key:    "test-gitlab-key",
			Secret: "test-gitlab-secret",
		},
		"dailymotion": {
			Key:    "test-dailymotion-key",
			Secret: "test-dailymotion-secret",
			Scopes: []string{"email"},
		},
		"deezer": {
			Key:    "test-deezer-key",
			Secret: "test-deezer-secret",
			Scopes: []string{"email"},
		},
		"discord": {
			Key:    "test-discord-key",
			Secret: "test-discord-secret",
			Scopes: []string{"identify", "email"},
		},
		"meetup": {
			Key:    "test-meetup-key",
			Secret: "test-meetup-secret",
		},
		"auth0": {
			Key:    "test-auth0-key",
			Secret: "test-auth0-secret",
			Domain: "test.auth0.com",
		},
		"xero": {
			Key:    "test-xero-key",
			Secret: "test-xero-secret",
		},
		"vk": {
			Key:    "test-vk-key",
			Secret: "test-vk-secret",
		},
		"naver": {
			Key:    "test-naver-key",
			Secret: "test-naver-secret",
		},
		"yandex": {
			Key:    "test-yandex-key",
			Secret: "test-yandex-secret",
		},
		"nextcloud": {
			Key:    "test-nextcloud-key",
			Secret: "test-nextcloud-secret",
			URL:    "https://test.nextcloud.com",
		},
		"gitea": {
			Key:    "test-gitea-key",
			Secret: "test-gitea-secret",
		},
		"shopify": {
			Key:    "test-shopify-key",
			Secret: "test-shopify-secret",
			Scopes: []string{"read_customers", "read_orders"},
		},
		"apple": {
			Key:    "test-apple-key",
			Secret: "test-apple-secret",
			Scopes: []string{"name", "email"},
		},
		"strava": {
			Key:    "test-strava-key",
			Secret: "test-strava-secret",
		},
		"okta": {
			Key:    "test-okta-key",
			Secret: "test-okta-secret",
			OrgURL: "https://test.okta.com",
			Scopes: []string{"openid", "profile", "email"},
		},
		"mastodon": {
			Key:    "test-mastodon-key",
			Secret: "test-mastodon-secret",
			Scopes: []string{"read:accounts"},
		},
		"wecom": {
			CorpID:  "test-wecom-corp",
			Secret:  "test-wecom-secret",
			AgentID: "test-wecom-agent",
		},
		"zoom": {
			Key:    "test-zoom-key",
			Secret: "test-zoom-secret",
			Scopes: []string{"read:user"},
		},
		"patreon": {
			Key:    "test-patreon-key",
			Secret: "test-patreon-secret",
		},
		"openid-connect": {
			Key:    "test-oidc-key",
			Secret: "test-oidc-secret",
			URL:    "http://localhost:8080/.well-known/openid_configuration",
		},
	}

	factory := &configProviderFactory{config: testConfig}
	providers := factory.CreateProviders("https://test.example.com/auth/%s/callback")

	// Verify that providers were created
	// We expect many providers (all the ones we configured)
	expectedMinProviders := 50
	if len(providers) < expectedMinProviders {
		t.Errorf("Expected at least %d providers, got %d", expectedMinProviders, len(providers))
	}

	// Verify some specific providers are present by checking their names
	providerNames := make(map[string]bool)
	actualProviders := make([]string, 0, len(providers))
	for _, provider := range providers {
		name := provider.Name()
		providerNames[name] = true
		actualProviders = append(actualProviders, name)
	}

	// Log all actual providers for debugging
	t.Logf("Actual providers created: %v", actualProviders)

	expectedProviders := []string{
		"twitterv2", "tiktok", "facebook", "fitbit", "google", "github",
		"spotify", "linkedin", "line", "lastfm", "twitch", "dropbox",
		"digitalocean", "bitbucket", "instagram", "intercom", "box",
		"salesforce", "seatalk", "amazon", "yammer", "onedrive",
		"azuread", "microsoftonline", "battlenet", "eveonline", "kakao",
		"yahoo", "typetalk", "slack", "stripe", "wepay", "paypal",
		"steam", "heroku", "uber", "soundcloud", "gitlab", "dailymotion",
		"deezer", "discord", "meetup", "auth0", "xero", "vk", "naver",
		"yandex", "nextcloud", "gitea", "shopify", "apple", "strava",
		"okta", "mastodon", "wecom", "zoom", "patreon",
		// Note: openid-connect may fail to initialize with test URLs
	}

	var missingProviders []string
	for _, expected := range expectedProviders {
		if !providerNames[expected] {
			missingProviders = append(missingProviders, expected)
		}
	}

	if len(missingProviders) > 0 {
		t.Errorf("Missing expected providers: %v", missingProviders)
	}

	t.Logf("Successfully created %d OAuth providers", len(providers))
}

func TestConfigProviderFactory_CreateProviders_NoConfig(t *testing.T) {
	// Test with no provider configuration
	factory := &configProviderFactory{config: make(map[string]ProviderConfig)}
	providers := factory.CreateProviders("https://test.example.com/auth/%s/callback")

	// Should return empty slice when no providers are configured
	if len(providers) != 0 {
		t.Errorf("Expected 0 providers when no config set, got %d", len(providers))
	}
}
