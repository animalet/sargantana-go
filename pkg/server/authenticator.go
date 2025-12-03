package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Authenticator defines the interface for authentication middleware providers.
// Implementations of this interface can provide custom authentication logic
// that will be used throughout the application to protect routes.
//
// This abstraction allows for flexible authentication strategies such as:
//   - OAuth2/OIDC via goth (default implementation in controller package)
//   - JWT token validation
//   - API key authentication
//   - Session-based authentication
//   - Custom authentication schemes
type Authenticator interface {
	// Middleware returns a Gin middleware function that performs authentication.
	// This middleware will be called for routes that require authentication.
	// It should check if the request is authenticated and either:
	//   - Call c.Next() to allow the request to proceed
	//   - Call c.AbortWithStatus() to reject the request
	Middleware() gin.HandlerFunc
}

// UnauthorizedAuthenticator is a fail-safe authenticator that rejects all requests.
// This is the default authenticator used by the server if no other authenticator
// is configured. It ensures that routes requiring authentication are protected
// by default, forcing developers to explicitly configure an authentication provider.
//
// This fail-safe approach prevents accidental exposure of protected routes.
type UnauthorizedAuthenticator struct{}

// Middleware returns a Gin middleware that always returns 401 Unauthorized.
// This ensures that any route using authentication middleware will fail
// unless a proper authenticator is configured.
func (u *UnauthorizedAuthenticator) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.AbortWithStatus(http.StatusUnauthorized)
	}
}

// NewUnauthorizedAuthenticator creates a new instance of the fail-safe authenticator.
func NewUnauthorizedAuthenticator() Authenticator {
	return &UnauthorizedAuthenticator{}
}
