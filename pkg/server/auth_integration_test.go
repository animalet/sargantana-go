//go:build integration

package server

import (
	"fmt"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/markbates/goth"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"golang.org/x/oauth2"
)

// MockAuthController is a generic mock controller that uses authentication
type MockAuthController struct {
	IController
	publicPath    string
	protectedPath string
	handlerCalled bool
}

func (m *MockAuthController) Bind(engine *gin.Engine, loginMiddleware gin.HandlerFunc) {
	// Public route - no auth required
	engine.GET(m.publicPath, func(c *gin.Context) {
		m.handlerCalled = true
		c.JSON(http.StatusOK, gin.H{"message": "public"})
	})

	// Protected route - auth required
	engine.GET(m.protectedPath, loginMiddleware, func(c *gin.Context) {
		m.handlerCalled = true
		c.JSON(http.StatusOK, gin.H{"message": "protected"})
	})
}

func (m *MockAuthController) Close() error {
	return nil
}

// MockAuthenticator for testing
type MockAuthenticator struct {
	allowAccess bool
}

func (m *MockAuthenticator) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if m.allowAccess {
			c.Next()
		} else {
			c.AbortWithStatus(http.StatusUnauthorized)
		}
	}
}

// MockGothProvider and MockGothSession for OAuth flow testing
type MockGothProvider struct {
	name string
}

func (m *MockGothProvider) Name() string { return m.name }

func (m *MockGothProvider) SetName(name string) { m.name = name }

func (m *MockGothProvider) BeginAuth(state string) (goth.Session, error) {
	return &MockGothSession{state: state}, nil
}

func (m *MockGothProvider) UnmarshalSession(string) (goth.Session, error) {
	return &MockGothSession{}, nil
}

func (m *MockGothProvider) FetchUser(goth.Session) (goth.User, error) {
	return goth.User{
		UserID:    "test-user-id",
		Email:     "test@example.com",
		Name:      "Test User",
		Provider:  m.name,
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}, nil
}

func (m *MockGothProvider) Debug(bool) {}

func (m *MockGothProvider) RefreshToken(string) (*oauth2.Token, error) {
	return nil, nil
}

func (m *MockGothProvider) RefreshTokenAvailable() bool {
	return false
}

type MockGothSession struct {
	state string
}

func (m *MockGothSession) GetAuthURL() (string, error) {
	return "http://mock-provider.com/authorize?state=" + m.state, nil
}

func (m *MockGothSession) Authorize(goth.Provider, goth.Params) (string, error) {
	return "mock-token", nil
}

func (m *MockGothSession) Marshal() string {
	return "mock-session"
}

func (m *MockGothSession) String() string {
	return "mock-session"
}

var _ = Describe("Generic Mock Controller with Authenticator", func() {
	var (
		ts            *httptest.Server
		client        *http.Client
		listener      net.Listener
		engine        *gin.Engine
		mockCtrl      *MockAuthController
		authenticator Authenticator
		sessionSecret = "test-secret-key-32-bytes-long!!"
	)

	BeforeEach(func() {
		var err error
		gin.SetMode(gin.TestMode)

		// Setup mock provider
		goth.UseProviders(&MockGothProvider{name: "mock"})

		DeferCleanup(func() {
			// Restore original providers
			goth.ClearProviders()
		})

		// Create listener with dynamic port
		listener, err = net.Listen("tcp", "127.0.0.1:0")
		Expect(err).NotTo(HaveOccurred())

		// Setup session store
		store := cookie.NewStore([]byte(sessionSecret))
		store.Options(sessions.Options{
			Path:     "/",
			HttpOnly: true,
			Secure:   false,
			MaxAge:   86400 * 30,
		})

		// Create engine
		engine = gin.New()
		engine.Use(sessions.Sessions("mysession", store))

		// Create mock controller
		mockCtrl = &MockAuthController{
			publicPath:    "/public",
			protectedPath: "/protected",
		}

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

	Context("Mock Controller with UnauthorizedAuthenticator", func() {
		BeforeEach(func() {
			authenticator = NewUnauthorizedAuthenticator()
			mockCtrl.Bind(engine, authenticator.Middleware())
		})

		It("should allow access to public routes", func() {
			resp, err := client.Get(fmt.Sprintf("%s/public", ts.URL))
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			Expect(mockCtrl.handlerCalled).To(BeTrue())
		})

		It("should return 401 for protected routes with UnauthorizedAuthenticator", func() {
			resp, err := client.Get(fmt.Sprintf("%s/protected", ts.URL))
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusUnauthorized))
		})
	})

	Context("Mock Controller with Custom MockAuthenticator", func() {
		Context("When access is denied", func() {
			BeforeEach(func() {
				authenticator = &MockAuthenticator{allowAccess: false}
				mockCtrl.Bind(engine, authenticator.Middleware())
			})

			It("should return 401 for protected routes when authenticator denies access", func() {
				resp, err := client.Get(fmt.Sprintf("%s/protected", ts.URL))
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusUnauthorized))
			})
		})

		Context("When access is allowed", func() {
			BeforeEach(func() {
				authenticator = &MockAuthenticator{allowAccess: true}
				mockCtrl.handlerCalled = false
				mockCtrl.Bind(engine, authenticator.Middleware())
			})

			It("should allow access to protected routes when authenticator allows", func() {
				resp, err := client.Get(fmt.Sprintf("%s/protected", ts.URL))
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				Expect(mockCtrl.handlerCalled).To(BeTrue())
			})
		})
	})

	Context("Mock Controller with Session-based Authenticator", func() {
		var sessionAuthenticator *SessionAuthenticator
		var sessionMockCtrl *MockAuthController

		BeforeEach(func() {
			// Create a new mock controller with unique paths to avoid route conflicts
			sessionMockCtrl = &MockAuthController{
				publicPath:    "/session-public",
				protectedPath: "/session-protected",
			}

			// Create a session-based authenticator similar to GothAuthenticator
			sessionAuthenticator = &SessionAuthenticator{}
			sessionMockCtrl.Bind(engine, sessionAuthenticator.Middleware())

			// Setup helper route to create session
			engine.GET("/session-login", func(c *gin.Context) {
				session := sessions.Default(c)
				// Use a simple string value that can be serialized by gob without registration
				session.Set("user", "test-user-id")
				err := session.Save()
				if err != nil {
					c.AbortWithError(http.StatusInternalServerError, err)
					return
				}
				c.Status(http.StatusOK)
			})
		})

		It("should return 401 when no session exists", func() {
			resp, err := client.Get(fmt.Sprintf("%s/session-protected", ts.URL))
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusUnauthorized))
		})

		It("should allow access when valid session exists", func() {
			// Create session via login
			resp, err := client.Get(fmt.Sprintf("%s/session-login", ts.URL))
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			// Access protected route
			resp, err = client.Get(fmt.Sprintf("%s/session-protected", ts.URL))
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			Expect(sessionMockCtrl.handlerCalled).To(BeTrue())
		})

		It("should handle multiple protected routes consistently", func() {
			// Login
			_, err := client.Get(fmt.Sprintf("%s/session-login", ts.URL))
			Expect(err).NotTo(HaveOccurred())

			// Access protected route multiple times
			for i := 0; i < 3; i++ {
				sessionMockCtrl.handlerCalled = false
				resp, err := client.Get(fmt.Sprintf("%s/session-protected", ts.URL))
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				Expect(sessionMockCtrl.handlerCalled).To(BeTrue())
			}
		})
	})
})

// SessionAuthenticator is a simple session-based authenticator for testing
type SessionAuthenticator struct{}

func (s *SessionAuthenticator) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		user := session.Get("user")
		if user == nil {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		c.Next()
	}
}
