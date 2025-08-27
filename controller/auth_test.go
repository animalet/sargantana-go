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

			controller := factory().(*Auth)

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

func TestAuth_AuthRouteHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	auth := NewAuth("http://localhost:8080")
	engine := gin.New()
	store := cookie.NewStore([]byte("secret"))
	engine.Use(sessions.Sessions("test", store))
	cfg := config.NewConfig("localhost:8080", "", "", false, "test-session")

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
	cfg := config.NewConfig("localhost:8080", "", "", false, "test-session")

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
	cfg := config.NewConfig("localhost:8080", "", "", false, "test-session")

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
			cfg := config.NewConfig(tt.configAddress, "", "", false, "test-session")

			auth.Bind(engine, *cfg, LoginFunc)

			if auth.callbackEndpoint != tt.expectedPrefix {
				t.Errorf("callbackEndpoint = %v, want %v", auth.callbackEndpoint, tt.expectedPrefix)
			}
		})
	}
}
