package session

import (
	"github.com/bradfitz/gomemcache/memcache"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/memcached"
	"github.com/pkg/errors"
)

// NewMemcachedSessionStore creates a new Memcached-backed session store.
// It configures the session store with appropriate security settings based on the mode.
//
// Parameters:
//   - secure: Whether to set the Secure flag on session cookies (typically true in release mode)
//   - secret: The secret key used for session encryption (should be at least 32 bytes)
//   - client: Pre-configured Memcached client with connection details
//
// Returns:
//   - sessions.Store: The configured Memcached session store
//   - error: An error if store creation fails
//
// Example usage:
//
//	memcachedClient := memcache.New("localhost:11211")
//	store, err := session.NewMemcachedSessionStore(true, []byte("secret-key"), memcachedClient)
//	if err != nil {
//	    log.Fatal(err)
//	}
func NewMemcachedSessionStore(secure bool, secret []byte, client *memcache.Client) (sessions.Store, error) {
	if client == nil {
		return nil, errors.New("Memcached client cannot be nil")
	}

	if len(secret) == 0 {
		return nil, errors.New("session secret cannot be empty")
	}

	// Create Memcached-backed session store
	// The keyPrefix is used to namespace session keys in Memcached
	store := memcached.NewStore(client, "session_", secret)

	// Configure session options
	store.Options(sessions.Options{
		Path:     "/",
		MaxAge:   86400, // 24 hours
		Secure:   secure,
		HttpOnly: true,
		SameSite: 3, // Lax
	})

	return store, nil
}
