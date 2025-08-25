package database

import (
	"testing"
	"time"

	"github.com/gomodule/redigo/redis"
)

func TestNewRedisPool(t *testing.T) {
	tests := []struct {
		name    string
		address string
	}{
		{
			name:    "localhost",
			address: "localhost:6379",
		},
		{
			name:    "remote address",
			address: "redis.example.com:6379",
		},
		{
			name:    "custom port",
			address: "localhost:6380",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool := NewRedisPool(tt.address)

			if pool == nil {
				t.Fatal("NewRedisPool returned nil")
			}

			// Verify pool configuration
			if pool.MaxIdle != 10 {
				t.Errorf("MaxIdle = %v, want 10", pool.MaxIdle)
			}

			if pool.IdleTimeout != 240*time.Second {
				t.Errorf("IdleTimeout = %v, want 240s", pool.IdleTimeout)
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

func TestRedisPool_TestOnBorrow(t *testing.T) {
	pool := NewRedisPool("localhost:6379")

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

func TestRedisPool_Dial(t *testing.T) {
	pool := NewRedisPool("invalid:address:format")

	// Test that Dial function is set and can be called
	// Note: This will fail to connect but should not panic
	conn, err := pool.Dial()
	if err == nil {
		// If somehow it succeeds, close the connection
		if conn != nil {
			err = conn.Close()
			if err != nil {
				t.Errorf("Failed to close connection: %v", err)
			}
		}
	}
	// We expect an error for invalid address, but the important thing
	// is that the Dial function doesn't panic
}

func TestRedisPool_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	pool := NewRedisPool("localhost:6379")
	defer func() {
		err := pool.Close()
		if err != nil {
			t.Errorf("Failed to close Redis pool: %v", err)
		}
	}()
	// Try to get a connection
	conn := pool.Get()
	defer func() {
		err := pool.Close()
		if err != nil {
			t.Errorf("Failed to close Redis pool: %v", err)
		}
	}()

	// Test basic Redis operation (will fail if Redis not available)
	_, err := conn.Do("PING")
	if err != nil {
		t.Logf("Redis not available for integration test: %v", err)
		t.Skip("Redis server not available")
	}
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

func (m *mockRedisConn) Do(commandName string, args ...interface{}) (interface{}, error) {
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

func (m *mockRedisConn) Send(commandName string, args ...interface{}) error {
	return nil
}

func (m *mockRedisConn) Flush() error {
	return nil
}

func (m *mockRedisConn) Receive() (interface{}, error) {
	return nil, nil
}
