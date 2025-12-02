package database

import (
	"time"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/pkg/errors"
)

// MemcachedConfig holds configuration for Memcached connection
type MemcachedConfig struct {
	// Servers is a list of Memcached server addresses (host:port)
	// Example: ["localhost:11211", "localhost:11212"]
	Servers []string `yaml:"servers"`

	// Timeout for connecting to Memcached servers
	// Default: 100ms if not specified
	Timeout time.Duration `yaml:"timeout"`

	// MaxIdleConns is the maximum number of idle connections per server
	// Default: 2 if not specified
	MaxIdleConns int `yaml:"max_idle_conns"`
}

// Validate checks if the MemcachedConfig has all required fields set
func (m MemcachedConfig) Validate() error {
	if len(m.Servers) == 0 {
		return errors.New("at least one Memcached server address is required")
	}

	for i, server := range m.Servers {
		if server == "" {
			return errors.Errorf("server address at index %d is empty", i)
		}
	}

	if m.Timeout < 0 {
		return errors.New("timeout cannot be negative")
	}

	if m.MaxIdleConns < 0 {
		return errors.New("max_idle_conns cannot be negative")
	}

	return nil
}

// CreateClient creates and configures a Memcached client from this config.
// Implements the config.ClientFactory[*memcache.Client] interface.
// Returns *memcache.Client on success, or an error if creation fails.

func (m MemcachedConfig) CreateClient() (*memcache.Client, error) {
	client := memcache.New(m.Servers...)

	// Set timeout (default to 100ms if not specified)
	if m.Timeout > 0 {
		client.Timeout = m.Timeout
	} else {
		client.Timeout = 100 * time.Millisecond
	}

	// Set max idle connections (default to 2 if not specified)
	if m.MaxIdleConns > 0 {
		client.MaxIdleConns = m.MaxIdleConns
	} else {
		client.MaxIdleConns = 2
	}

	// Test connection with a simple operation
	if err := client.Ping(); err != nil {
		return nil, errors.Wrap(err, "failed to connect to Memcached")
	}

	return client, nil
}
