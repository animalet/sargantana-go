package database

import (
	"testing"
	"time"

	"github.com/bradfitz/gomemcache/memcache"
)

// TestMemcachedHealthCheck tests if Memcached service is running
func TestMemcachedHealthCheck(t *testing.T) {
	config := &MemcachedConfig{
		Servers:      []string{"localhost:11211"},
		Timeout:      1 * time.Second,
		MaxIdleConns: 2,
	}

	client, err := config.CreateClient()
	if err != nil {
		t.Fatalf("Failed to connect to Memcached: %v", err)
	}

	// Test basic connectivity with a ping
	if err := client.Ping(); err != nil {
		t.Fatalf("Failed to ping Memcached: %v", err)
	}
}

// TestMemcachedConfig_Validate tests the validation logic
func TestMemcachedConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  MemcachedConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: MemcachedConfig{
				Servers:      []string{"localhost:11211"},
				Timeout:      100 * time.Millisecond,
				MaxIdleConns: 2,
			},
			wantErr: false,
		},
		{
			name: "multiple servers",
			config: MemcachedConfig{
				Servers:      []string{"localhost:11211", "localhost:11212"},
				Timeout:      100 * time.Millisecond,
				MaxIdleConns: 5,
			},
			wantErr: false,
		},
		{
			name: "default timeout and pool size",
			config: MemcachedConfig{
				Servers: []string{"localhost:11211"},
			},
			wantErr: false,
		},
		{
			name: "no servers",
			config: MemcachedConfig{
				Timeout:      100 * time.Millisecond,
				MaxIdleConns: 2,
			},
			wantErr: true,
		},
		{
			name: "empty server address",
			config: MemcachedConfig{
				Servers:      []string{"localhost:11211", ""},
				Timeout:      100 * time.Millisecond,
				MaxIdleConns: 2,
			},
			wantErr: true,
		},
		{
			name: "negative timeout",
			config: MemcachedConfig{
				Servers:      []string{"localhost:11211"},
				Timeout:      -1 * time.Second,
				MaxIdleConns: 2,
			},
			wantErr: true,
		},
		{
			name: "negative max idle conns",
			config: MemcachedConfig{
				Servers:      []string{"localhost:11211"},
				Timeout:      100 * time.Millisecond,
				MaxIdleConns: -1,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("MemcachedConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestMemcachedConfig_CreateClient tests client creation
func TestMemcachedConfig_CreateClient(t *testing.T) {
	tests := []struct {
		name      string
		config    *MemcachedConfig
		wantError bool
	}{
		{
			name: "valid configuration",
			config: &MemcachedConfig{
				Servers:      []string{"localhost:11211"},
				Timeout:      1 * time.Second,
				MaxIdleConns: 2,
			},
			wantError: false,
		},
		{
			name: "invalid configuration - no servers",
			config: &MemcachedConfig{
				Timeout:      1 * time.Second,
				MaxIdleConns: 2,
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := tt.config.CreateClient()
			if tt.wantError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if client == nil {
					t.Error("Expected client but got nil")
				}
			}
		})
	}
}

// TestMemcachedOperations tests basic Memcached operations
func TestMemcachedOperations(t *testing.T) {
	config := &MemcachedConfig{
		Servers:      []string{"localhost:11211"},
		Timeout:      1 * time.Second,
		MaxIdleConns: 2,
	}

	client, err := config.CreateClient()
	if err != nil {
		t.Fatalf("Failed to create Memcached client: %v", err)
	}

	// Test Set operation
	err = client.Set(&memcache.Item{
		Key:   "test-key",
		Value: []byte("test-value"),
	})
	if err != nil {
		t.Fatalf("Failed to set value: %v", err)
	}

	// Test Get operation
	item, err := client.Get("test-key")
	if err != nil {
		t.Fatalf("Failed to get value: %v", err)
	}

	if string(item.Value) != "test-value" {
		t.Errorf("Expected 'test-value', got '%s'", string(item.Value))
	}

	// Test Delete operation
	err = client.Delete("test-key")
	if err != nil {
		t.Fatalf("Failed to delete value: %v", err)
	}

	// Verify deletion
	_, err = client.Get("test-key")
	if err != memcache.ErrCacheMiss {
		t.Errorf("Expected cache miss error, got: %v", err)
	}
}

// BenchmarkMemcachedSet benchmarks Memcached SET operations
func BenchmarkMemcachedSet(b *testing.B) {
	config := &MemcachedConfig{
		Servers:      []string{"localhost:11211"},
		Timeout:      1 * time.Second,
		MaxIdleConns: 10,
	}

	client, err := config.CreateClient()
	if err != nil {
		b.Fatalf("Failed to create Memcached client: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = client.Set(&memcache.Item{
			Key:   "bench-key",
			Value: []byte("bench-value"),
		})
	}
}

// BenchmarkMemcachedGet benchmarks Memcached GET operations
func BenchmarkMemcachedGet(b *testing.B) {
	config := &MemcachedConfig{
		Servers:      []string{"localhost:11211"},
		Timeout:      1 * time.Second,
		MaxIdleConns: 10,
	}

	client, err := config.CreateClient()
	if err != nil {
		b.Fatalf("Failed to create Memcached client: %v", err)
	}

	// Set initial value
	_ = client.Set(&memcache.Item{
		Key:   "bench-key",
		Value: []byte("bench-value"),
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = client.Get("bench-key")
	}
}
