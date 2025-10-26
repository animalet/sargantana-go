// Package database provides database connectivity and integration for the Sargantana Go web framework.
// It includes Redis connection pooling for session storage and caching, as well as Neo4j driver
// configuration for graph database operations.
package database

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"time"

	"github.com/gomodule/redigo/redis"
)

// RedisConfig holds configuration options for Redis connection pool
type RedisConfig struct {
	Address     string        `yaml:"address"`
	Username    string        `yaml:"username,omitempty"`
	Password    string        `yaml:"password,omitempty"`
	Database    int           `yaml:"database,omitempty"`
	MaxIdle     int           `yaml:"max_idle"`
	IdleTimeout time.Duration `yaml:"idle_timeout"`
	TLS         *TLSConfig    `yaml:"tls,omitempty"`
}

func (r RedisConfig) Validate() error {
	if r.Address == "" {
		return fmt.Errorf("redis address must be set and non-empty")
	}
	if r.MaxIdle < 0 {
		return fmt.Errorf("redis max_idle must be non-negative")
	}
	if r.IdleTimeout < 0 {
		return fmt.Errorf("redis idle_timeout must be non-negative")
	}
	if r.Database < 0 {
		return fmt.Errorf("redis database must be non-negative")
	}
	if r.TLS != nil {
		if (r.TLS.CertFile != "" && r.TLS.KeyFile == "") || (r.TLS.CertFile == "" && r.TLS.KeyFile != "") {
			return fmt.Errorf("both cert_file and key_file must be set together in TLS configuration")
		}
	}
	return nil
}

// CreateClient creates and configures a Redis connection pool from this config.
// Implements the config.ClientFactory[*redis.Pool] interface.
// Returns *redis.Pool on success.
func (r *RedisConfig) CreateClient() (*redis.Pool, error) {
	if err := r.Validate(); err != nil {
		return nil, fmt.Errorf("invalid Redis configuration: %w", err)
	}
	return NewRedisPoolWithConfig(r), nil
}

// TLSConfig holds TLS configuration for Redis connections
type TLSConfig struct {
	InsecureSkipVerify bool   `yaml:"insecure_skip_verify"`
	CertFile           string `yaml:"cert_file"`
	KeyFile            string `yaml:"key_file"`
	CAFile             string `yaml:"ca_file"`
}

// NewRedisPoolWithConfig creates a new Redis connection pool with custom configuration.
// This function provides full control over connection parameters including TLS settings.
//
// Parameters:
//   - config: RedisConfig struct containing all connection parameters
//
// Returns a configured Redis connection pool ready for use.
func NewRedisPoolWithConfig(config *RedisConfig) *redis.Pool {
	return &redis.Pool{
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			if time.Since(t) < time.Minute {
				return nil
			}
			_, err := c.Do("PING")
			return err
		},
		MaxIdle:     config.MaxIdle,
		IdleTimeout: config.IdleTimeout,
		Dial: func() (redis.Conn, error) {
			return dialRedis(config)
		},
	}
}

// dialRedis establishes a Redis connection with the given configuration
func dialRedis(config *RedisConfig) (redis.Conn, error) {
	var opts []redis.DialOption

	// Add username if provided
	if config.Username != "" {
		opts = append(opts, redis.DialUsername(config.Username))
	}

	// Add password authentication if provided
	if config.Password != "" {
		opts = append(opts, redis.DialPassword(config.Password))
	}

	// Add database selection (default to 0)
	opts = append(opts, redis.DialDatabase(config.Database))

	// Configure TLS if enabled
	if config.TLS != nil {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: config.TLS.InsecureSkipVerify,
		}

		// Load CA certificate if provided
		if config.TLS.CAFile != "" {
			caCert, err := os.ReadFile(config.TLS.CAFile)
			if err != nil {
				return nil, err
			}
			caCertPool := x509.NewCertPool()
			if !caCertPool.AppendCertsFromPEM(caCert) {
				return nil, fmt.Errorf("failed to parse CA certificate %q", config.TLS.CAFile)
			}
			tlsConfig.RootCAs = caCertPool
		}

		// Load client certificates if provided
		if config.TLS.CertFile != "" && config.TLS.KeyFile != "" {
			cert, err := tls.LoadX509KeyPair(config.TLS.CertFile, config.TLS.KeyFile)
			if err != nil {
				return nil, err
			}
			tlsConfig.Certificates = []tls.Certificate{cert}
		}

		opts = append(opts, redis.DialTLSConfig(tlsConfig))
		opts = append(opts, redis.DialUseTLS(true))
		return redis.Dial("tcp", config.Address, opts...)
	}

	return redis.Dial("tcp", config.Address, opts...)
}
