// Package session provides session storage implementations for the Sargantana Go web framework.
// It supports both cookie-based and Redis-based session storage with secure configuration
// options optimized for web applications.
package session

import (
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
)

// NewCookieStore creates a new cookie-based session store with secure default settings.
// The store is configured with appropriate security settings based on the deployment mode
// and integrates with the Goth authentication library for OAuth2 session management.
//
// Parameters:
//   - isReleaseMode: Whether the application is running in production mode (affects cookie security)
//   - secret: Secret key used for cookie signing and encryption (should be random and secure)
//
// Returns a configured cookie session store with the following settings:
//   - Path: "/" (cookies available for entire site)
//   - MaxAge: 86400 seconds (24 hours)
//   - Secure: true in release mode, false in debug mode
//   - HttpOnly: true (prevents JavaScript access to cookies)
//   - SameSite: Lax mode (balanced security and functionality)
func NewCookieStore(isReleaseMode bool, secret []byte) sessions.Store {
	store := cookie.NewStore(secret)

	store.Options(sessions.Options{
		Path:     "/",
		MaxAge:   86400, // 24 hours
		Secure:   isReleaseMode,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	return store
}
