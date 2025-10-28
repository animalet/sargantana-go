package session

import (
	"net/http"

	"github.com/gin-contrib/sessions"
	redissessions "github.com/gin-contrib/sessions/redis"
	"github.com/gomodule/redigo/redis"
)

// NewRedisSessionStore creates a new Redis-based session store with secure default settings.
// This store provides distributed session storage suitable for multi-instance deployments.
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
// Returns an error if Redis store creation or configuration fails.
func NewRedisSessionStore(secure bool, secret []byte, pool *redis.Pool) (sessions.Store, error) {
	store, err := redissessions.NewStoreWithPool(pool, secret)
	if err != nil {
		return nil, err
	}

	rediStore, err := redissessions.GetRedisStore(store)
	if err != nil {
		return nil, err
	}

	rediStore.Options.Path = "/"
	rediStore.Options.MaxAge = 86400 // 24 hours
	rediStore.Options.Secure = secure
	rediStore.Options.HttpOnly = true
	rediStore.Options.SameSite = http.SameSiteLaxMode

	// Set MaxLength to 0 to allow unlimited session data size (stored in Redis, not cookies)
	rediStore.SetMaxLength(0)

	return store, nil
}
