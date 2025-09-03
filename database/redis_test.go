package database

import (
	"testing"
	"time"

	"github.com/gomodule/redigo/redis"
)

func TestNewRedisPoolWithConfig(t *testing.T) {
	tests := []struct {
		name   string
		config *RedisConfig
	}{
		{
			name: "basic config",
			config: &RedisConfig{
				Address:     "localhost:6379",
				MaxIdle:     5,
				IdleTimeout: 120 * time.Second,
			},
		},
		{
			name: "config with password",
			config: &RedisConfig{
				Address:     "localhost:6379",
				Password:    "secret",
				MaxIdle:     15,
				IdleTimeout: 300 * time.Second,
			},
		},
		{
			name: "config with database selection",
			config: &RedisConfig{
				Address:     "localhost:6379",
				Database:    intPtr(1),
				MaxIdle:     8,
				IdleTimeout: 180 * time.Second,
			},
		},
		{
			name: "config with TLS enabled",
			config: &RedisConfig{
				Address:     "localhost:6380",
				MaxIdle:     12,
				IdleTimeout: 240 * time.Second,
				TLS: &TLSConfig{
					InsecureSkipVerify: true,
				},
			},
		},
		{
			name: "config with TLS and certificates",
			config: &RedisConfig{
				Address:     "secure-redis.example.com:6380",
				Password:    "secure-password",
				MaxIdle:     20,
				IdleTimeout: 360 * time.Second,
				TLS: &TLSConfig{
					InsecureSkipVerify: false,
					CertFile:           "/path/to/client.crt",
					KeyFile:            "/path/to/client.key",
					CAFile:             "/path/to/ca.crt",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool := NewRedisPoolWithConfig(tt.config)

			if pool == nil {
				t.Fatal("NewRedisPoolWithConfig returned nil")
			}

			// Verify pool configuration
			if pool.MaxIdle != tt.config.MaxIdle {
				t.Errorf("MaxIdle = %v, want %v", pool.MaxIdle, tt.config.MaxIdle)
			}

			if pool.IdleTimeout != tt.config.IdleTimeout {
				t.Errorf("IdleTimeout = %v, want %v", pool.IdleTimeout, tt.config.IdleTimeout)
			}

			if pool.TestOnBorrow == nil {
				t.Error("TestOnBorrow is nil")
			}

			if pool.Dial == nil {
				t.Error("Dial function is nil")
			}
		})
	}
}

func TestRedisConfig_DefaultValues(t *testing.T) {
	config := &RedisConfig{
		Address: "localhost:6379",
	}

	pool := NewRedisPoolWithConfig(config)

	if pool == nil {
		t.Fatal("NewRedisPoolWithConfig returned nil")
	}

	// Test that defaults are properly handled
	if pool.MaxIdle != 0 {
		t.Errorf("Expected MaxIdle to be 0 when not set, got %v", pool.MaxIdle)
	}

	if pool.IdleTimeout != 0 {
		t.Errorf("Expected IdleTimeout to be 0 when not set, got %v", pool.IdleTimeout)
	}
}

func TestDialRedis_WithDatabase(t *testing.T) {
	config := &RedisConfig{
		Address:  "localhost:6380",
		Database: intPtr(2),
		TLS: &TLSConfig{
			InsecureSkipVerify: true,
		},
	}

	_, err := dialRedis(config)
	t.Logf("dialRedis with database selection returned error (expected in test environment): %v", err)
}

func TestDialRedis_WithTLS(t *testing.T) {
	config := &RedisConfig{
		Address: "localhost:6380",
		TLS: &TLSConfig{
			InsecureSkipVerify: true,
		},
	}

	_, err := dialRedis(config)
	t.Logf("dialRedis with TLS returned error (expected in test environment): %v", err)
}

func TestTLSConfig_WithCertificates(t *testing.T) {
	// Test that dialRedis handles certificate loading gracefully
	config := &RedisConfig{
		Address: "localhost:6380",
		TLS: &TLSConfig{
			InsecureSkipVerify: false,
			CertFile:           "/nonexistent/cert.pem",
			KeyFile:            "/nonexistent/key.pem",
		},
	}

	_, err := dialRedis(config)
	if err == nil {
		t.Error("Expected error when loading nonexistent certificates")
	}
	t.Logf("dialRedis with invalid certificates returned expected error: %v", err)
}

func TestRedisPool_TestOnBorrow(t *testing.T) {
	config := &RedisConfig{
		Address:     "localhost:6379",
		MaxIdle:     10,
		IdleTimeout: 240 * time.Second,
	}
	pool := NewRedisPoolWithConfig(config)

	tests := []struct {
		name     string
		timeAgo  time.Duration
		mockConn *mockRedisConn
		wantErr  bool
	}{
		{
			name:     "recent connection",
			timeAgo:  30 * time.Second,
			mockConn: &mockRedisConn{},
			wantErr:  false,
		},
		{
			name:     "old connection ping success",
			timeAgo:  2 * time.Minute,
			mockConn: &mockRedisConn{pingResponse: "PONG"},
			wantErr:  false,
		},
		{
			name:     "old connection ping failure",
			timeAgo:  2 * time.Minute,
			mockConn: &mockRedisConn{pingError: redis.ErrNil},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testTime := time.Now().Add(-tt.timeAgo)
			err := pool.TestOnBorrow(tt.mockConn, testTime)

			if tt.wantErr && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestRedisPool_Integration(t *testing.T) {
	config := &RedisConfig{
		Address:     "localhost:6380",
		MaxIdle:     10,
		IdleTimeout: 240 * time.Second,
		TLS: &TLSConfig{
			InsecureSkipVerify: true, // For testing with self-signed certificates
			CAFile:             "../certs/ca.crt",
			CertFile:           "../certs/redis.crt",
			KeyFile:            "../certs/redis.key",
		},
	}
	pool := NewRedisPoolWithConfig(config)
	defer func() {
		err := pool.Close()
		if err != nil {
			t.Errorf("Failed to close Redis pool: %v", err)
		}
	}()

	// Try to get a connection
	conn := pool.Get()
	defer func() {
		err := conn.Close()
		if err != nil {
			t.Errorf("Failed to close Redis connection: %v", err)
		}
	}()

	// Test basic Redis operation
	_, err := conn.Do("PING")
	if err != nil {
		// If TLS Redis is not available, try regular Redis as fallback
		regularConfig := &RedisConfig{
			Address:     "localhost:6379",
			MaxIdle:     10,
			IdleTimeout: 240 * time.Second,
		}
		regularPool := NewRedisPoolWithConfig(regularConfig)
		defer func() {
			closeErr := regularPool.Close()
			if closeErr != nil {
				t.Errorf("Failed to close regular Redis pool: %v", closeErr)
			}
		}()

		regularConn := regularPool.Get()
		defer func() {
			closeErr := regularConn.Close()
			if closeErr != nil {
				t.Errorf("Failed to close regular Redis connection: %v", closeErr)
			}
		}()

		_, fallbackErr := regularConn.Do("PING")
		if fallbackErr != nil {
			t.Fatalf("Neither TLS Redis (port 6380) nor regular Redis (port 6379) are available for integration test. TLS error: %v, Regular error: %v", err, fallbackErr)
		}

		// Use regular connection for remaining tests
		conn = regularConn
		pool = regularPool
	}

	// Test basic operations if Redis is available
	_, err = conn.Do("SET", "test:key", "test:value")
	if err != nil {
		t.Errorf("Failed to SET key: %v", err)
	}

	reply, err := conn.Do("GET", "test:key")
	if err != nil {
		t.Errorf("Failed to GET key: %v", err)
	}

	value, err := redis.String(reply, err)
	if err != nil {
		t.Errorf("Failed to convert reply to string: %v", err)
	}

	if value != "test:value" {
		t.Errorf("Expected 'test:value', got '%s'", value)
	}

	// Clean up
	_, err = conn.Do("DEL", "test:key")
	if err != nil {
		t.Errorf("Failed to DELETE key: %v", err)
	}
}

func BenchmarkRedisPool_GetConnectionTLS(b *testing.B) {
	config := &RedisConfig{
		Address:     "localhost:6380",
		MaxIdle:     10,
		IdleTimeout: 240 * time.Second,
		TLS: &TLSConfig{
			InsecureSkipVerify: true, // For testing with self-signed certificates
			CAFile:             "../certs/ca.crt",
			CertFile:           "../certs/redis.crt",
			KeyFile:            "../certs/redis.key",
		},
	}
	pool := NewRedisPoolWithConfig(config)
	defer func() {
		err := pool.Close()
		if err != nil {
			b.Errorf("Failed to close Redis pool: %v", err)
		}
	}()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		conn := pool.Get()
		err := conn.Close()
		if err != nil {
			b.Errorf("Failed to close connection: %v", err)
		}
	}
}

func BenchmarkRedisPool_TestOnBorrowTLS(b *testing.B) {
	config := &RedisConfig{
		Address:     "localhost:6380",
		MaxIdle:     10,
		IdleTimeout: 240 * time.Second,
		TLS: &TLSConfig{
			InsecureSkipVerify: true, // For testing with self-signed certificates
			CAFile:             "../certs/ca.crt",
			CertFile:           "../certs/redis.crt",
			KeyFile:            "../certs/redis.key",
		},
	}
	pool := NewRedisPoolWithConfig(config)
	mockConn := &mockRedisConn{pingResponse: "PONG"}
	testTime := time.Now().Add(-2 * time.Minute)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = pool.TestOnBorrow(mockConn, testTime)
	}
}

// intPtr returns a pointer to the given int value
func intPtr(i int) *int {
	return &i
}

// Mock Redis connection for testing
type mockRedisConn struct {
	pingResponse interface{}
	pingError    error
	closed       bool
}

func (m *mockRedisConn) Close() error {
	m.closed = true
	return nil
}

func (m *mockRedisConn) Err() error {
	return nil
}

func (m *mockRedisConn) Do(commandName string, _ ...interface{}) (interface{}, error) {
	if commandName == "PING" {
		if m.pingError != nil {
			return nil, m.pingError
		}
		if m.pingResponse != nil {
			return m.pingResponse, nil
		}
		return "PONG", nil
	}
	return nil, nil
}

func (m *mockRedisConn) Send(_ string, _ ...interface{}) error {
	return nil
}

func (m *mockRedisConn) Flush() error {
	return nil
}

func (m *mockRedisConn) Receive() (interface{}, error) {
	return nil, nil
}
