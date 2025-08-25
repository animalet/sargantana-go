// Package database provides database connectivity and integration for the Sargantana Go web framework.
// It includes Redis connection pooling for session storage and caching, as well as Neo4j driver
// configuration for graph database operations.
package database

import (
	"time"

	"github.com/gomodule/redigo/redis"
)

// NewRedisPool creates a new Redis connection pool with optimized settings for web applications.
// The pool includes connection health checking, idle connection management, and automatic reconnection.
// This pool is primarily used for session storage but can also be used for general Redis operations.
//
// Parameters:
//   - address: Redis server address in "host:port" format (e.g., "localhost:6379")
//
// Returns a configured Redis connection pool ready for use.
// The pool automatically manages connections and includes:
//   - Health checking with PING commands for connections older than 1 minute
//   - Maximum of 10 idle connections
//   - 240-second idle timeout for unused connections
//   - Automatic TCP connection establishment
func NewRedisPool(address string) *redis.Pool {
	return &redis.Pool{
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			if time.Since(t) < time.Minute {
				return nil
			}
			_, err := c.Do("PING")
			return err
		},
		MaxIdle:     10,
		IdleTimeout: 240 * time.Second,
		Dial:        func() (redis.Conn, error) { return redis.Dial("tcp", address) },
	}
}
