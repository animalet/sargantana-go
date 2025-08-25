package session

import (
	"log"
	"net/http"

	"github.com/gin-contrib/sessions"
	redissessions "github.com/gin-contrib/sessions/redis"
	"github.com/gomodule/redigo/redis"
	"github.com/markbates/goth/gothic"
)

// NewRedisSessionStore creates a new Redis-based session store with secure default settings.
// This store provides distributed session storage suitable for multi-instance deployments
// and integrates with the Goth authentication library for OAuth2 session management.
//
// Parameters:
//   - isReleaseMode: Whether the application is running in production mode (affects cookie security)
//   - secret: Secret key used for session data encryption (should be random and secure)
//   - pool: Pre-configured Redis connection pool for database operations
//
// Returns a configured Redis session store with the following settings:
//   - Path: "/" (cookies available for entire site)
//   - MaxAge: 86400 seconds (24 hours)
//   - Secure: true in release mode, false in debug mode
//   - HttpOnly: true (prevents JavaScript access to cookies)
//   - SameSite: Lax mode (balanced security and functionality)
//
// The function will log fatal errors and exit if Redis store creation fails.
func NewRedisSessionStore(isReleaseMode bool, secret []byte, pool *redis.Pool) sessions.Store {
	store, err := redissessions.NewStoreWithPool(pool, secret)
	if err != nil {
		log.Fatalf("Failed to create session store: %v", err)
	}

	rediStore, err := redissessions.GetRedisStore(store)
	if err != nil {
		log.Fatalf("Failed to get redis store: %v", err)
	}

	rediStore.Options.Path = "/"
	rediStore.Options.MaxAge = 86400 // 24 hours
	rediStore.Options.Secure = isReleaseMode
	rediStore.Options.HttpOnly = true
	rediStore.Options.SameSite = http.SameSiteLaxMode

	gothic.Store = rediStore
	return store
}
