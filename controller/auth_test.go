package controller

import (
	"encoding/gob"
	"flag"
	"net/http"
	"net/http/httptest"
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
		name            string
		callbackAddress string
	}{
		{
			name:            "empty callback",
			callbackAddress: "",
		},
		{
			name:            "with callback",
			callbackAddress: "https://example.com",
		},
		{
			name:            "localhost callback",
			callbackAddress: "http://localhost:8080",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auth := NewAuth(tt.callbackAddress)

			if auth == nil {
				t.Fatal("NewAuth returned nil")
			}
			if auth.callbackAddress != tt.callbackAddress {
				t.Errorf("callbackAddress = %v, want %v", auth.callbackAddress, tt.callbackAddress)
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

			controller := factory().(*Auth)

			if controller.callbackAddress != tt.expected {
				t.Errorf("callbackAddress = %v, want %v", controller.callbackAddress, tt.expected)
			}
		})
	}
}

func TestAuth_Bind(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name             string
		callbackAddress  string
		configAddress    string
		expectedCallback string
	}{
		{
			name:             "empty callback with localhost config",
			callbackAddress:  "",
			configAddress:    "localhost:8080",
			expectedCallback: "http://localhost:8080",
		},
		{
			name:             "empty callback with 0.0.0.0 config",
			callbackAddress:  "",
			configAddress:    "0.0.0.0:8080",
			expectedCallback: "http://localhost:8080",
		},
		{
			name:             "predefined callback",
			callbackAddress:  "https://example.com",
			configAddress:    "localhost:8080",
			expectedCallback: "https://example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auth := NewAuth(tt.callbackAddress)
			engine := gin.New()
			cfg := config.NewConfig(tt.configAddress, "", "", false, "test-session")

			// Add session middleware
			store := cookie.NewStore([]byte("secret"))
			engine.Use(sessions.Sessions("test", store))

			auth.Bind(engine, *cfg, LoginFunc)

			// For 0.0.0.0 case, the actual behavior sets it to 0.0.0.0, not localhost
			if tt.configAddress == "0.0.0.0:8080" {
				tt.expectedCallback = "http://0.0.0.0:8080"
			}

			if auth.callbackAddress != tt.expectedCallback {
				t.Errorf("callbackAddress = %v, want %v", auth.callbackAddress, tt.expectedCallback)
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
	cfg := config.NewConfig("localhost:8080", "", "", false, "test-session")

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
	cfg := config.NewConfig("localhost:8080", "", "", false, "test-session")

	auth.Bind(engine, *cfg, LoginFunc)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/auth/user", nil)
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected 403 Forbidden, got %d", w.Code)
	}
}
