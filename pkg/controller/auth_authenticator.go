package controller

import (
	"net/http"
	"time"

	"github.com/animalet/sargantana-go/pkg/server"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

// GothAuthenticator implements server.Authenticator using goth for OAuth2 authentication.
// It validates sessions created by the auth controller and ensures users are authenticated
// before accessing protected routes.
type GothAuthenticator struct{}

// NewGothAuthenticator creates a new authenticator that uses goth sessions for authentication.
// This authenticator should be used in conjunction with the auth controller which handles
// the OAuth2 login flow and session management.
//
// Example usage:
//
//	server := server.NewServer(cfg)
//	server.SetAuthenticator(controller.NewGothAuthenticator())
func NewGothAuthenticator() server.Authenticator {
	return &GothAuthenticator{}
}

// Middleware returns a Gin middleware function that validates goth-based authentication.
// It checks for a valid user session and ensures the OAuth2 token has not expired.
// If authentication fails, it returns 401 Unauthorized.
func (g *GothAuthenticator) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		userSession := sessions.Default(c)
		userObject := userSession.Get("user")
		if userObject == nil {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		u, ok := userObject.(UserObject)

		if !ok || time.Now().After(u.User.ExpiresAt) {
			userSession.Clear()
			err := userSession.Save()
			if err != nil {
				_ = c.AbortWithError(http.StatusInternalServerError, err)
				return
			}
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		c.Next()
	}
}
