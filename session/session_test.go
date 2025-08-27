package session

import (
	"errors"
	"testing"

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
		})
	}
}

func TestNewRedisSessionStore(t *testing.T) {

	tests := []struct {
		name          string
		isReleaseMode bool
		secret        []byte
		poolFunc      func() *redis.Pool
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
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Unexpected panic: %v", r)
				}
			}()

			pool := tt.poolFunc()
			defer func() {
				err := pool.Close()
				if err != nil {
					t.Error("Failed to close Redis pool:", err)
				}
			}()

			store, err := NewRedisSessionStore(tt.isReleaseMode, tt.secret, pool)

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if store == nil {
				t.Fatal("NewRedisSessionStore returned nil")
			}

			// Verify that gothic.Store was set
			if gothic.Store == nil {
				t.Error("gothic.Store was not set")
			}
		})
	}
}

func TestNewRedisSessionStore_WithConnectionError(t *testing.T) {

	// Create a pool that will fail on connection test
	pool := &redis.Pool{
		Dial: func() (redis.Conn, error) {
			return &mockRedisConnWithError{}, nil
		},
	}
	defer func() {
		err := pool.Close()
		if err != nil {
			t.Error("Failed to close Redis pool:", err)
		}
	}()

	// This should still create the store but connection errors will appear later
	store, err := NewRedisSessionStore(false, []byte("test-secret"), pool)
	if err == nil {
		t.Error("Expected error due to connection issues, but got none")
	}

	if store != nil {
		t.Error("NewRedisSessionStore should return nil store on connection error")
	}

	// Verify that gothic.Store was set despite connection issues
	if gothic.Store == nil {
		t.Error("gothic.Store was not set")
	}
}

func TestNewRedisSessionStore_WithInvalidSecret(t *testing.T) {

	pool := &redis.Pool{
		Dial: func() (redis.Conn, error) {
			return &mockRedisConn{}, nil
		},
	}
	defer func() {
		err := pool.Close()
		if err != nil {
			t.Error("Failed to close Redis pool:", err)
		}
	}()

	// Test with empty secret
	store, err := NewRedisSessionStore(false, []byte{}, pool)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if store == nil {
		t.Fatal("NewRedisSessionStore returned nil")
	}
}

func TestRedisSessionStore_AuthenticationBehavior(t *testing.T) {

	tests := []struct {
		name         string
		mockResponse interface{}
		mockError    error
	}{
		{
			name:         "successful auth",
			mockResponse: "OK",
			mockError:    nil,
		},
		{
			name:         "auth error",
			mockResponse: nil,
			mockError:    errors.New("AUTH failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool := &redis.Pool{
				Dial: func() (redis.Conn, error) {
					return &mockRedisConnWithAuth{
						authResponse: tt.mockResponse,
						authError:    tt.mockError,
					}, nil
				},
			}
			defer func() {
				err := pool.Close()
				if err != nil {
					t.Error("Failed to close Redis pool:", err)
				}
			}()

			store, err := NewRedisSessionStore(false, []byte("test-secret"), pool)
			if tt.mockError != nil && err == nil {
				t.Error("Expected error but got none")
			}
			if tt.mockError != nil && store != nil {
				t.Error("Expected error but store was created")
			}
			if tt.mockError == nil && store == nil {
				t.Error("Expected store to be created but got nil")
			}
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

// Additional mock Redis connection with auth support
type mockRedisConnWithAuth struct {
	mockRedisConn
	authResponse interface{}
	authError    error
}

func (m *mockRedisConnWithAuth) Do(commandName string, args ...interface{}) (interface{}, error) {
	switch commandName {
	case "AUTH", "PING":
		return m.authResponse, m.authError
	default:
		return m.mockRedisConn.Do(commandName, args...)
	}
}

// Mock Redis connection that always errors
type mockRedisConnWithError struct {
	mockRedisConn
}

func (m *mockRedisConnWithError) Do(commandName string, args ...interface{}) (interface{}, error) {
	return nil, errors.New("connection error")
}
