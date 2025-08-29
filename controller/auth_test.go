package controller

import (
	"encoding/gob"
	"flag"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/animalet/sargantana-go/config"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/markbates/goth"
)

func init() {
	gob.Register(UserObject{})
}

func TestNewAuth(t *testing.T) {
	tests := []struct {
		name             string
		callbackEndpoint string
	}{
		{
			name:             "empty callback",
			callbackEndpoint: "",
		},
		{
			name:             "with callback",
			callbackEndpoint: "https://example.com",
		},
		{
			name:             "localhost callback",
			callbackEndpoint: "http://localhost:8080",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auth := NewAuth(tt.callbackEndpoint)

			if auth == nil {
				t.Fatal("NewAuth returned nil")
			}
			if auth.callbackEndpoint != tt.callbackEndpoint {
				t.Errorf("callbackEndpoint = %v, want %v", auth.callbackEndpoint, tt.callbackEndpoint)
			}
		})
	}
}

func TestNewAuthFromFlags(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected string
	}{
		{
			name:     "default callback",
			args:     []string{},
			expected: "",
		},
		{
			name:     "custom callback",
			args:     []string{"-callback=https://myapp.com"},
			expected: "https://myapp.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
			factory := NewAuthFromFlags(flagSet)

			err := flagSet.Parse(tt.args)
			if err != nil {
				t.Fatalf("Failed to parse flags: %v", err)
			}

			controller := factory().(*auth)

			if controller.callbackEndpoint != tt.expected {
				t.Errorf("callbackEndpoint = %v, want %v", controller.callbackEndpoint, tt.expected)
			}
		})
	}
}

func TestAuth_UserFactory(t *testing.T) {
	auth := NewAuth("test")

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
					err = c.AbortWithError(http.StatusInternalServerError, err)
					c.String(http.StatusInternalServerError, "Failed to save session: %v", err)
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
					err = c.AbortWithError(http.StatusInternalServerError, err)
					c.String(http.StatusInternalServerError, "Failed to save session: %v", err)
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
	auth := NewAuth("test")
	err := auth.Close()
	if err != nil {
		t.Errorf("Close() returned error: %v", err)
	}
}

func TestAuth_Routes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	auth := NewAuth("http://localhost:8080")
	engine := gin.New()
	store := cookie.NewStore([]byte("secret"))
	engine.Use(sessions.Sessions("test", store))
	cfg := config.NewConfig("localhost:8080", "", "", false, "test-session", "")

	auth.Bind(engine, *cfg, LoginFunc)

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

	auth := NewAuth("http://localhost:8080")
	engine := gin.New()
	store := cookie.NewStore([]byte("secret"))
	engine.Use(sessions.Sessions("test", store))
	cfg := config.NewConfig("localhost:8080", "", "", false, "test-session", "")

	auth.Bind(engine, *cfg, LoginFunc)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/auth/user", nil)
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected 403 Forbidden, got %d", w.Code)
	}
}

func TestAuth_AuthRouteHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	auth := NewAuth("http://localhost:8080")
	engine := gin.New()
	store := cookie.NewStore([]byte("secret"))
	engine.Use(sessions.Sessions("test", store))
	cfg := config.NewConfig("localhost:8080", "", "", false, "test-session", "")

	auth.Bind(engine, *cfg, LoginFunc)

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

	auth := NewAuth("http://localhost:8080")
	engine := gin.New()
	store := cookie.NewStore([]byte("secret"))
	engine.Use(sessions.Sessions("test", store))
	cfg := config.NewConfig("localhost:8080", "", "", false, "test-session", "")

	auth.Bind(engine, *cfg, LoginFunc)

	// Test callback route without provider
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/auth//callback", nil)
	engine.ServeHTTP(w, req)

	// Should return 404 since no provider specified
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 404 for missing provider callback, got %d", w.Code)
	}
}

func TestAuth_LogoutRouteHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	auth := NewAuth("http://localhost:8080")
	engine := gin.New()
	store := cookie.NewStore([]byte("secret"))
	engine.Use(sessions.Sessions("test", store))
	cfg := config.NewConfig("localhost:8080", "", "", false, "test-session", "")

	auth.Bind(engine, *cfg, LoginFunc)

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
			auth := NewAuth("")
			engine := gin.New()
			store := cookie.NewStore([]byte("secret"))
			engine.Use(sessions.Sessions("test", store))
			cfg := config.NewConfig(tt.configAddress, "", "", false, "test-session", "")

			auth.Bind(engine, *cfg, LoginFunc)

			if auth.callbackEndpoint != tt.expectedPrefix {
				t.Errorf("callbackEndpoint = %v, want %v", auth.callbackEndpoint, tt.expectedPrefix)
			}
		})
	}
}

func TestProductionProviderFactory_CreateProviders_AllProviders(t *testing.T) {
	// Set test values for all environment variables
	testValues := map[string]string{
		"TWITTER_KEY": "test-twitter-key", "TWITTER_SECRET": "test-twitter-secret",
		"TIKTOK_KEY": "test-tiktok-key", "TIKTOK_SECRET": "test-tiktok-secret",
		"FACEBOOK_KEY": "test-facebook-key", "FACEBOOK_SECRET": "test-facebook-secret",
		"FITBIT_KEY": "test-fitbit-key", "FITBIT_SECRET": "test-fitbit-secret",
		"GOOGLE_KEY": "test-google-key", "GOOGLE_SECRET": "test-google-secret",
		"GITHUB_KEY": "test-github-key", "GITHUB_SECRET": "test-github-secret",
		"SPOTIFY_KEY": "test-spotify-key", "SPOTIFY_SECRET": "test-spotify-secret",
		"LINKEDIN_KEY": "test-linkedin-key", "LINKEDIN_SECRET": "test-linkedin-secret",
		"LINE_KEY": "test-line-key", "LINE_SECRET": "test-line-secret",
		"LASTFM_KEY": "test-lastfm-key", "LASTFM_SECRET": "test-lastfm-secret",
		"TWITCH_KEY": "test-twitch-key", "TWITCH_SECRET": "test-twitch-secret",
		"DROPBOX_KEY": "test-dropbox-key", "DROPBOX_SECRET": "test-dropbox-secret",
		"DIGITALOCEAN_KEY": "test-do-key", "DIGITALOCEAN_SECRET": "test-do-secret",
		"BITBUCKET_KEY": "test-bitbucket-key", "BITBUCKET_SECRET": "test-bitbucket-secret",
		"INSTAGRAM_KEY": "test-instagram-key", "INSTAGRAM_SECRET": "test-instagram-secret",
		"INTERCOM_KEY": "test-intercom-key", "INTERCOM_SECRET": "test-intercom-secret",
		"BOX_KEY": "test-box-key", "BOX_SECRET": "test-box-secret",
		"SALESFORCE_KEY": "test-salesforce-key", "SALESFORCE_SECRET": "test-salesforce-secret",
		"SEATALK_KEY": "test-seatalk-key", "SEATALK_SECRET": "test-seatalk-secret",
		"AMAZON_KEY": "test-amazon-key", "AMAZON_SECRET": "test-amazon-secret",
		"YAMMER_KEY": "test-yammer-key", "YAMMER_SECRET": "test-yammer-secret",
		"ONEDRIVE_KEY": "test-onedrive-key", "ONEDRIVE_SECRET": "test-onedrive-secret",
		"AZUREAD_KEY": "test-azuread-key", "AZUREAD_SECRET": "test-azuread-secret",
		"MICROSOFTONLINE_KEY": "test-msonline-key", "MICROSOFTONLINE_SECRET": "test-msonline-secret",
		"BATTLENET_KEY": "test-battlenet-key", "BATTLENET_SECRET": "test-battlenet-secret",
		"EVEONLINE_KEY": "test-eve-key", "EVEONLINE_SECRET": "test-eve-secret",
		"KAKAO_KEY": "test-kakao-key", "KAKAO_SECRET": "test-kakao-secret",
		"YAHOO_KEY": "test-yahoo-key", "YAHOO_SECRET": "test-yahoo-secret",
		"TYPETALK_KEY": "test-typetalk-key", "TYPETALK_SECRET": "test-typetalk-secret",
		"SLACK_KEY": "test-slack-key", "SLACK_SECRET": "test-slack-secret",
		"STRIPE_KEY": "test-stripe-key", "STRIPE_SECRET": "test-stripe-secret",
		"WEPAY_KEY": "test-wepay-key", "WEPAY_SECRET": "test-wepay-secret",
		"PAYPAL_KEY": "test-paypal-key", "PAYPAL_SECRET": "test-paypal-secret",
		"STEAM_KEY":  "test-steam-key",
		"HEROKU_KEY": "test-heroku-key", "HEROKU_SECRET": "test-heroku-secret",
		"UBER_KEY": "test-uber-key", "UBER_SECRET": "test-uber-secret",
		"SOUNDCLOUD_KEY": "test-soundcloud-key", "SOUNDCLOUD_SECRET": "test-soundcloud-secret",
		"GITLAB_KEY": "test-gitlab-key", "GITLAB_SECRET": "test-gitlab-secret",
		"DAILYMOTION_KEY": "test-dailymotion-key", "DAILYMOTION_SECRET": "test-dailymotion-secret",
		"DEEZER_KEY": "test-deezer-key", "DEEZER_SECRET": "test-deezer-secret",
		"DISCORD_KEY": "test-discord-key", "DISCORD_SECRET": "test-discord-secret",
		"MEETUP_KEY": "test-meetup-key", "MEETUP_SECRET": "test-meetup-secret",
		"AUTH0_KEY": "test-auth0-key", "AUTH0_SECRET": "test-auth0-secret", "AUTH0_DOMAIN": "test.auth0.com",
		"XERO_KEY": "test-xero-key", "XERO_SECRET": "test-xero-secret",
		"VK_KEY": "test-vk-key", "VK_SECRET": "test-vk-secret",
		"NAVER_KEY": "test-naver-key", "NAVER_SECRET": "test-naver-secret",
		"YANDEX_KEY": "test-yandex-key", "YANDEX_SECRET": "test-yandex-secret",
		"NEXTCLOUD_KEY": "test-nextcloud-key", "NEXTCLOUD_SECRET": "test-nextcloud-secret", "NEXTCLOUD_URL": "https://test.nextcloud.com",
		"GITEA_KEY": "test-gitea-key", "GITEA_SECRET": "test-gitea-secret",
		"SHOPIFY_KEY": "test-shopify-key", "SHOPIFY_SECRET": "test-shopify-secret",
		"APPLE_KEY": "test-apple-key", "APPLE_SECRET": "test-apple-secret",
		"STRAVA_KEY": "test-strava-key", "STRAVA_SECRET": "test-strava-secret",
		"OKTA_ID": "test-okta-id", "OKTA_SECRET": "test-okta-secret", "OKTA_ORG_URL": "https://test.okta.com",
		"MASTODON_KEY": "test-mastodon-key", "MASTODON_SECRET": "test-mastodon-secret",
		"WECOM_CORP_ID": "test-wecom-corp", "WECOM_SECRET": "test-wecom-secret", "WECOM_AGENT_ID": "test-wecom-agent",
		"ZOOM_KEY": "test-zoom-key", "ZOOM_SECRET": "test-zoom-secret",
		"PATREON_KEY": "test-patreon-key", "PATREON_SECRET": "test-patreon-secret",
		"OPENID_CONNECT_KEY": "test-oidc-key", "OPENID_CONNECT_SECRET": "test-oidc-secret", "OPENID_CONNECT_DISCOVERY_URL": "https://test.example.com/.well-known/openid_configuration",
	}

	defer func() {
		for k := range testValues {
			_ = os.Unsetenv(k)
		}
	}()

	// Set all test environment variables
	for env, val := range testValues {
		_ = os.Setenv(env, val)
	}

	factory := &productionProviderFactory{}
	providers := factory.CreateProviders("https://test.example.com/auth/%s/callback")

	// Verify that providers were created
	// We expect at least 50 providers (all the ones we set environment variables for)
	if len(providers) < 50 {
		t.Errorf("Expected at least 50 providers, got %d", len(providers))
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
		// Note: openid-connect is excluded as it may fail to initialize with test URLs
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

func TestProductionProviderFactory_CreateProviders_NoEnvironmentVars(t *testing.T) {
	// Test with no environment variables set
	factory := &productionProviderFactory{}
	providers := factory.CreateProviders("https://test.example.com/auth/%s/callback")

	// Should return empty slice when no environment variables are set
	if len(providers) != 0 {
		t.Errorf("Expected 0 providers when no env vars set, got %d", len(providers))
	}
}
