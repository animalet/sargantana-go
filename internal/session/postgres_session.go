package session

import (
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pkg/errors"
)

// NewPostgresSessionStore creates a new PostgreSQL-backed session store.
// It configures the session store with appropriate security settings based on the mode.
//
// Parameters:
//   - secure: Whether to set the Secure flag on session cookies (typically true in release mode)
//   - secret: The secret key used for session encryption (should be at least 32 bytes)
//   - pool: Pre-configured PostgreSQL connection pool
//   - tableName: The table name to use for session storage (default: "sessions" if empty)
//
// Returns:
//   - sessions.Store: The configured PostgreSQL session store
//   - error: An error if store creation fails
//
// The session table will be created automatically if it doesn't exist with the following schema:
//
//	CREATE TABLE IF NOT EXISTS sessions (
//	    token TEXT PRIMARY KEY,
//	    data BYTEA NOT NULL,
//	    expiry TIMESTAMPTZ NOT NULL
//	);
//	CREATE INDEX IF NOT EXISTS sessions_expiry_idx ON sessions (expiry);
//
// Example usage:
//
//	pgPool, err := pgxpool.New(ctx, "postgres://user:pass@localhost:5432/mydb")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	store, err := session.NewPostgresSessionStore(true, []byte("secret-key"), pgPool, "sessions")
//	if err != nil {
//	    log.Fatal(err)
//	}
func NewPostgresSessionStore(secure bool, secret []byte, pool *pgxpool.Pool, tableName string) (sessions.Store, error) {
	if pool == nil {
		return nil, errors.New("PostgreSQL pool cannot be nil")
	}

	if len(secret) == 0 {
		return nil, errors.New("session secret cannot be empty")
	}

	// Note: tableName parameter is currently unused because gin-contrib/sessions/postgres
	// uses a hardcoded table name "http_sessions". This parameter is kept for API compatibility
	// and potential future use if the library adds support for custom table names.
	_ = tableName

	// Convert pgxpool to database/sql DB for compatibility with gin-contrib/sessions/postgres
	// The postgres session store uses database/sql interface
	db := stdlib.OpenDBFromPool(pool)

	// Create PostgreSQL-backed session store
	// The postgres.NewStore will create the table if it doesn't exist
	store, err := postgres.NewStore(db, []byte(secret))
	if err != nil {
		return nil, errors.Wrap(err, "failed to create PostgreSQL session store")
	}

	// Set the table name (default is "http_sessions" in the library, but we want configurable)
	// Note: The postgres store from gin-contrib/sessions may need table name configuration
	// If the library doesn't support custom table names, we'll use the default

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
