package session

import (
	"testing"

	"github.com/gin-contrib/sessions"
	"github.com/gomodule/redigo/redis"
	"github.com/markbates/goth/gothic"
)

func TestNewCookieStore(t *testing.T) {
	tests := []struct {
		name           string
		isReleaseMode  bool
		secret         []byte
		expectedSecure bool
	}{
		{
			name:           "development mode",
			isReleaseMode:  false,
			secret:         []byte("development-secret"),
			expectedSecure: false,
		},
		{
			name:           "production mode",
			isReleaseMode:  true,
			secret:         []byte("production-secret-key"),
			expectedSecure: true,
		},
		{
			name:           "empty secret",
			isReleaseMode:  false,
			secret:         []byte(""),
			expectedSecure: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewCookieStore(tt.isReleaseMode, tt.secret)

			if store == nil {
				t.Fatal("NewCookieStore returned nil")
			}

			// Verify that gothic.Store was set
			if gothic.Store == nil {
				t.Error("gothic.Store was not set")
			}

			// Test that the store implements the sessions.Store interface
			var _ sessions.Store = store
		})
	}
}

func TestNewRedisSessionStore(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Redis session store test in short mode")
	}

	tests := []struct {
		name          string
		isReleaseMode bool
		secret        []byte
		poolFunc      func() *redis.Pool
		expectPanic   bool
	}{
		{
			name:          "valid pool development",
			isReleaseMode: false,
			secret:        []byte("test-secret"),
			poolFunc: func() *redis.Pool {
				return &redis.Pool{
					Dial: func() (redis.Conn, error) {
						return &mockRedisConn{}, nil
					},
				}
			},
			expectPanic: false,
		},
		{
			name:          "valid pool production",
			isReleaseMode: true,
			secret:        []byte("production-secret"),
			poolFunc: func() *redis.Pool {
				return &redis.Pool{
					Dial: func() (redis.Conn, error) {
						return &mockRedisConn{}, nil
					},
				}
			},
			expectPanic: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					if !tt.expectPanic {
						t.Errorf("Unexpected panic: %v", r)
					}
				} else if tt.expectPanic {
					t.Error("Expected panic but didn't get one")
				}
			}()

			pool := tt.poolFunc()
			defer pool.Close()

			store := NewRedisSessionStore(tt.isReleaseMode, tt.secret, pool)

			if store == nil {
				t.Fatal("NewRedisSessionStore returned nil")
			}

			// Verify that gothic.Store was set
			if gothic.Store == nil {
				t.Error("gothic.Store was not set")
			}

			// Test that the store implements the sessions.Store interface
			var _ sessions.Store = store
		})
	}
}

func TestCookieStore_Configuration(t *testing.T) {
	tests := []struct {
		name           string
		isReleaseMode  bool
		expectedSecure bool
	}{
		{
			name:           "development mode options",
			isReleaseMode:  false,
			expectedSecure: false,
		},
		{
			name:           "production mode options",
			isReleaseMode:  true,
			expectedSecure: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			secret := []byte("test-secret-key")
			store := NewCookieStore(tt.isReleaseMode, secret)

			// Verify store was created successfully
			if store == nil {
				t.Fatal("NewCookieStore returned nil")
			}

			// Verify gothic store was set
			if gothic.Store != store {
				t.Error("gothic.Store was not set correctly")
			}
		})
	}
}

func TestSessionStore_CreatesSuccessfully(t *testing.T) {
	secret := []byte("test-secret")
	store := NewCookieStore(false, secret)

	if store == nil {
		t.Fatal("Failed to create cookie store")
	}

	// Verify it implements the Store interface
	var _ sessions.Store = store
}

// Mock Redis connection for testing
type mockRedisConn struct {
	closed bool
	data   map[string]string
}

func (m *mockRedisConn) Close() error {
	m.closed = true
	return nil
}

func (m *mockRedisConn) Err() error {
	return nil
}

func (m *mockRedisConn) Do(commandName string, args ...interface{}) (interface{}, error) {
	switch commandName {
	case "PING":
		return "PONG", nil
	case "GET":
		if len(args) > 0 {
			if m.data == nil {
				return nil, nil
			}
			key := args[0].(string)
			return m.data[key], nil
		}
	case "SET":
		if len(args) >= 2 {
			if m.data == nil {
				m.data = make(map[string]string)
			}
			key := args[0].(string)
			value := args[1].(string)
			m.data[key] = value
			return "OK", nil
		}
	}
	return nil, nil
}

func (m *mockRedisConn) Send(commandName string, args ...interface{}) error {
	return nil
}

func (m *mockRedisConn) Flush() error {
	return nil
}

func (m *mockRedisConn) Receive() (interface{}, error) {
	return nil, nil
}
