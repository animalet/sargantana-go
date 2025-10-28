package session

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// TestNewPostgresSessionStore tests creating a new PostgreSQL session store
func TestNewPostgresSessionStore(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	connStr := "postgres://user:password@localhost:5432/my_blog_db?sslmode=disable"
	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		t.Skipf("PostgreSQL not available: %v", err)
	}
	defer pool.Close()

	// Test connection
	if err := pool.Ping(ctx); err != nil {
		t.Skipf("PostgreSQL not available: %v", err)
	}

	secret := []byte("test-secret-key-that-is-long-enough")

	tests := []struct {
		name      string
		secure    bool
		secret    []byte
		pool      *pgxpool.Pool
		tableName string
		wantErr   bool
	}{
		{
			name:      "valid configuration with default table",
			secure:    true,
			secret:    secret,
			pool:      pool,
			tableName: "",
			wantErr:   false,
		},
		{
			name:      "valid configuration with custom table",
			secure:    false,
			secret:    secret,
			pool:      pool,
			tableName: "custom_sessions",
			wantErr:   false,
		},
		{
			name:      "nil pool",
			secure:    true,
			secret:    secret,
			pool:      nil,
			tableName: "sessions",
			wantErr:   true,
		},
		{
			name:      "empty secret",
			secure:    true,
			secret:    []byte{},
			pool:      pool,
			tableName: "sessions",
			wantErr:   true,
		},
		{
			name:      "nil secret",
			secure:    true,
			secret:    nil,
			pool:      pool,
			tableName: "sessions",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, err := NewPostgresSessionStore(tt.secure, tt.secret, tt.pool, tt.tableName)
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				if store != nil {
					t.Error("Expected nil store but got non-nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if store == nil {
					t.Error("Expected store but got nil")
				}
			}
		})
	}
}

// TestPostgresSessionStore_Integration tests the session store with actual PostgreSQL
func TestPostgresSessionStore_Integration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	connStr := "postgres://user:password@localhost:5432/my_blog_db?sslmode=disable"
	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		t.Skipf("PostgreSQL not available: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		t.Skipf("PostgreSQL not available: %v", err)
	}

	secret := []byte("test-secret-key-that-is-long-enough")
	store, err := NewPostgresSessionStore(false, secret, pool, "test_sessions")
	if err != nil {
		t.Fatalf("Failed to create session store: %v", err)
	}

	if store == nil {
		t.Fatal("Expected non-nil store")
	}

	// Note: The gin-contrib/sessions/postgres package will create the table automatically
	// We could verify table existence here if needed

	// Clean up test table
	_, _ = pool.Exec(ctx, "DROP TABLE IF EXISTS test_sessions")
}

// TestPostgresSessionStore_DefaultTable tests using default table name
func TestPostgresSessionStore_DefaultTable(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	connStr := "postgres://user:password@localhost:5432/my_blog_db?sslmode=disable"
	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		t.Skipf("PostgreSQL not available: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		t.Skipf("PostgreSQL not available: %v", err)
	}

	secret := []byte("test-secret-key-that-is-long-enough")

	// Pass empty string for table name to use default
	store, err := NewPostgresSessionStore(true, secret, pool, "")
	if err != nil {
		t.Fatalf("Failed to create session store with default table: %v", err)
	}

	if store == nil {
		t.Fatal("Expected non-nil store")
	}
}

// TestPostgresSessionStore_ConnectionPooling tests with different pool configurations
func TestPostgresSessionStore_ConnectionPooling(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create pool with custom configuration
	config, err := pgxpool.ParseConfig("postgres://user:password@localhost:5432/my_blog_db?sslmode=disable")
	if err != nil {
		t.Fatalf("Failed to parse config: %v", err)
	}

	config.MaxConns = 5
	config.MinConns = 1

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Skipf("PostgreSQL not available: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		t.Skipf("PostgreSQL not available: %v", err)
	}

	secret := []byte("test-secret-key-that-is-long-enough")
	store, err := NewPostgresSessionStore(false, secret, pool, "pooled_sessions")
	if err != nil {
		t.Fatalf("Failed to create session store with custom pool: %v", err)
	}

	if store == nil {
		t.Fatal("Expected non-nil store")
	}

	// Clean up
	_, _ = pool.Exec(ctx, "DROP TABLE IF EXISTS pooled_sessions")
}

// TestPostgresSessionStore_SSL tests connection with SSL
func TestPostgresSessionStore_SSL(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Try with SSL mode prefer (will work even if SSL is not available)
	connStr := "postgres://user:password@localhost:5432/my_blog_db?sslmode=prefer"
	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		t.Skipf("PostgreSQL not available: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		t.Skipf("PostgreSQL not available: %v", err)
	}

	secret := []byte("test-secret-key-that-is-long-enough")
	store, err := NewPostgresSessionStore(true, secret, pool, "ssl_sessions")
	if err != nil {
		t.Fatalf("Failed to create session store with SSL: %v", err)
	}

	if store == nil {
		t.Fatal("Expected non-nil store")
	}

	// Clean up
	_, _ = pool.Exec(ctx, "DROP TABLE IF EXISTS ssl_sessions")
}
