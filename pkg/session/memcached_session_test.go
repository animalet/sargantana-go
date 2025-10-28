package session

import (
	"testing"
	"time"

	"github.com/bradfitz/gomemcache/memcache"
)

// TestNewMemcachedSessionStore tests creating a new Memcached session store
func TestNewMemcachedSessionStore(t *testing.T) {
	client := memcache.New("localhost:11211")
	secret := []byte("test-secret-key-that-is-long-enough")

	tests := []struct {
		name    string
		secure  bool
		secret  []byte
		client  *memcache.Client
		wantErr bool
	}{
		{
			name:    "valid configuration secure",
			secure:  true,
			secret:  secret,
			client:  client,
			wantErr: false,
		},
		{
			name:    "valid configuration insecure",
			secure:  false,
			secret:  secret,
			client:  client,
			wantErr: false,
		},
		{
			name:    "nil client",
			secure:  true,
			secret:  secret,
			client:  nil,
			wantErr: true,
		},
		{
			name:    "empty secret",
			secure:  true,
			secret:  []byte{},
			client:  client,
			wantErr: true,
		},
		{
			name:    "nil secret",
			secure:  true,
			secret:  nil,
			client:  client,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, err := NewMemcachedSessionStore(tt.secure, tt.secret, tt.client)
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

// TestMemcachedSessionStore_Integration tests the session store with actual Memcached
func TestMemcachedSessionStore_Integration(t *testing.T) {
	client := memcache.New("localhost:11211")

	// Test connectivity
	if err := client.Ping(); err != nil {
		t.Skipf("Memcached not available: %v", err)
	}

	secret := []byte("test-secret-key-that-is-long-enough")
	store, err := NewMemcachedSessionStore(false, secret, client)
	if err != nil {
		t.Fatalf("Failed to create session store: %v", err)
	}

	if store == nil {
		t.Fatal("Expected non-nil store")
	}

	// Basic validation that the store was created with proper options
	// The gin-contrib/sessions API doesn't expose Options() directly,
	// but we've configured them in the constructor
}

// TestMemcachedSessionStore_MultipleClients tests with multiple Memcached servers
func TestMemcachedSessionStore_MultipleClients(t *testing.T) {
	// Note: This test assumes a single Memcached server
	// In production, you might have multiple servers for redundancy
	client := memcache.New("localhost:11211")

	if err := client.Ping(); err != nil {
		t.Skipf("Memcached not available: %v", err)
	}

	secret := []byte("test-secret-key-that-is-long-enough")
	store, err := NewMemcachedSessionStore(true, secret, client)
	if err != nil {
		t.Fatalf("Failed to create session store: %v", err)
	}

	if store == nil {
		t.Fatal("Expected non-nil store")
	}
}

// TestMemcachedSessionStore_Timeout tests client with custom timeout
func TestMemcachedSessionStore_Timeout(t *testing.T) {
	client := memcache.New("localhost:11211")
	client.Timeout = 500 * time.Millisecond
	client.MaxIdleConns = 5

	if err := client.Ping(); err != nil {
		t.Skipf("Memcached not available: %v", err)
	}

	secret := []byte("test-secret-key-that-is-long-enough")
	store, err := NewMemcachedSessionStore(false, secret, client)
	if err != nil {
		t.Fatalf("Failed to create session store: %v", err)
	}

	if store == nil {
		t.Fatal("Expected non-nil store")
	}
}
