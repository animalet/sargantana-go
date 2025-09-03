package database

import (
	"testing"
	"time"

	"github.com/gomodule/redigo/redis"
)

func TestNewRedisPoolWithConfig(t *testing.T) {
	tests := []struct {
		name      string
		config    *RedisConfig
		connError bool
	}{
		{
			name: "basic config",
			config: &RedisConfig{
				Address:     "localhost:6379",
				Username:    "redisuser",
				Password:    "redispass",
				MaxIdle:     5,
				IdleTimeout: 120 * time.Second,
			},
		},
		{
			name: "config with wrong password",
			config: &RedisConfig{
				Address:     "localhost:6379",
				Username:    "redisuser",
				Password:    "badpassword",
				MaxIdle:     15,
				IdleTimeout: 300 * time.Second,
			},
			connError: true,
		},
		{
			name: "config with good password",
			config: &RedisConfig{
				Address:     "localhost:6379",
				Username:    "redisuser",
				Password:    "redispass",
				MaxIdle:     15,
				IdleTimeout: 300 * time.Second,
			},
		},
		{
			name: "config with database selection",
			config: &RedisConfig{
				Address:     "localhost:6379",
				Username:    "redisuser",
				Password:    "redispass",
				Database:    1,
				MaxIdle:     8,
				IdleTimeout: 180 * time.Second,
			},
		},
		{
			name: "config with TLS enabled",
			config: &RedisConfig{
				Address:     "localhost:6380",
				Username:    "redisuser",
				Password:    "redispass",
				MaxIdle:     12,
				IdleTimeout: 5 * time.Second,
				TLS: &TLSConfig{
					InsecureSkipVerify: true,
				},
			},
		},
		{
			name: "config with wrong certificates",
			config: &RedisConfig{
				Address:     "localhost:6380",
				Username:    "redisuser",
				Password:    "redispass",
				MaxIdle:     20,
				IdleTimeout: 360 * time.Second,
				TLS: &TLSConfig{
					InsecureSkipVerify: false,
					CertFile:           "/path/to/client.crt",
					KeyFile:            "/path/to/client.key",
					CAFile:             "/path/to/ca.crt",
				},
			},
			connError: true,
		},
		{
			name: "config with wrong address",
			config: &RedisConfig{
				Address:     "secure-redis.example.com:6380",
				Password:    "redispass",
				MaxIdle:     20,
				IdleTimeout: 360 * time.Second,
				TLS: &TLSConfig{
					InsecureSkipVerify: false,
					CertFile:           "/path/to/client.crt",
					KeyFile:            "/path/to/client.key",
					CAFile:             "/path/to/ca.crt",
				},
			},
			connError: true,
		},
		{
			name: "config with TLS and client certificate",
			config: &RedisConfig{
				Address:     "localhost:6380",
				Username:    "redisuser",
				Password:    "redispass",
				MaxIdle:     20,
				IdleTimeout: 360 * time.Second,
				TLS: &TLSConfig{
					InsecureSkipVerify: false,
					CAFile:             "../certs/ca.crt",
					CertFile:           "../certs/client.crt",
					KeyFile:            "../certs/client.key",
				},
			},
		},
		{
			name: "config with TLS and server certificate",
			config: &RedisConfig{
				Address:     "localhost:6380",
				Username:    "redisuser",
				Password:    "redispass",
				MaxIdle:     20,
				IdleTimeout: 360 * time.Second,
				TLS: &TLSConfig{
					InsecureSkipVerify: false,
					CAFile:             "../certs/ca.crt",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool := NewRedisPoolWithConfig(tt.config)
			defer func() {
				err := pool.Close()
				if err != nil {
					t.Errorf("Failed to close Redis pool: %v", err)
				}
			}()
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
			conn := pool.Get()
			defer func() {
				err := pool.Close()
				if err != nil {
					t.Errorf("Failed to close Redis pool: %v", err)
				}
			}()

			_, err := conn.Do("PING")
			switch {
			case err != nil && !tt.connError:
				t.Fatalf("Unexpected connection error state: got %v, want error: %v", err, tt.connError)
			case err != nil && tt.connError:
				t.Logf("Connection failed as expected: %v", err)
				return
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
		})
	}
}

func BenchmarkRedisPool_GetConnection(b *testing.B) {
	benchmarks := []struct {
		name   string
		config *RedisConfig
	}{
		{
			name: "NoTLS_WithAuth",
			config: &RedisConfig{
				Address:     "localhost:6379",
				Username:    "redisuser",
				Password:    "redispass",
				MaxIdle:     10,
				IdleTimeout: 5 * time.Second,
			},
		},
		{
			name: "TLS_NoAuth",
			config: &RedisConfig{
				Address:     "localhost:6380",
				MaxIdle:     10,
				IdleTimeout: 5 * time.Second,
				TLS: &TLSConfig{
					InsecureSkipVerify: true,
				},
			},
		},
		{
			name: "TLS_WithAuth",
			config: &RedisConfig{
				Address:     "localhost:6380",
				Username:    "redisuser",
				Password:    "redispass",
				MaxIdle:     10,
				IdleTimeout: 5 * time.Second,
				TLS: &TLSConfig{
					InsecureSkipVerify: false,
					CAFile:             "../certs/ca.crt",
					CertFile:           "../certs/client.crt",
					KeyFile:            "../certs/client.key",
				},
			},
		},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			pool := NewRedisPoolWithConfig(bm.config)
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
		})
	}
}

func BenchmarkRedisPool_TestOnBorrow(b *testing.B) {
	benchmarks := []struct {
		name   string
		config *RedisConfig
	}{
		{
			name: "NoTLS_WithAuth",
			config: &RedisConfig{
				Address:     "localhost:6379",
				Username:    "redisuser",
				Password:    "redispass",
				MaxIdle:     10,
				IdleTimeout: 5 * time.Second,
			},
		},
		{
			name: "TLS_WithAuthAndServerCert",
			config: &RedisConfig{
				Address:     "localhost:6380",
				Username:    "redisuser",
				Password:    "redispass",
				MaxIdle:     10,
				IdleTimeout: 5 * time.Second,
				TLS: &TLSConfig{
					InsecureSkipVerify: true,
					CAFile:             "../certs/ca.crt",
				},
			},
		},
		{
			name: "TLS_WithAuthAndClientCert",
			config: &RedisConfig{
				Address:     "localhost:6380",
				Username:    "redisuser",
				Password:    "redispass",
				MaxIdle:     10,
				IdleTimeout: 5 * time.Second,
				TLS: &TLSConfig{
					InsecureSkipVerify: false,
					CAFile:             "../certs/ca.crt",
					CertFile:           "../certs/client.crt",
					KeyFile:            "../certs/client.key",
				},
			},
		},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			pool := NewRedisPoolWithConfig(bm.config)
			defer func() {
				err := pool.Close()
				if err != nil {
					b.Errorf("Failed to close Redis pool: %v", err)
				}
			}()

			// Get a connection to test the TestOnBorrow function directly
			conn := pool.Get()
			defer func() {
				_ = conn.Close()
			}()

			// Create a time that's older than 1 minute to trigger TestOnBorrow
			oldTime := time.Now().Add(-2 * time.Minute)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				// Call the TestOnBorrow function directly with an old timestamp
				err := pool.TestOnBorrow(conn, oldTime)
				if err != nil {
					b.Errorf("TestOnBorrow failed: %v", err)
				}
			}
		})
	}
}
