//go:build integration

package session

import (
	"time"

	"github.com/animalet/sargantana-go/pkg/database"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Redis Session Integration", func() {
	Context("Redis with authentication", func() {
		It("should create a session store with username/password authentication", func() {
			// Connect to Redis with ACL authentication
			cfg := database.RedisConfig{
				Address:     "localhost:6379",
				Username:    "redisuser",
				Password:    "redispass",
				Database:    0,
				MaxIdle:     10,
				IdleTimeout: 240 * time.Second,
			}
			pool, err := cfg.CreateClient()
			Expect(err).NotTo(HaveOccurred())
			defer pool.Close()

			// Test connection with authentication
			conn := pool.Get()
			defer conn.Close()
			_, err = conn.Do("PING")
			Expect(err).NotTo(HaveOccurred())

			// Create Session Store
			store, err := NewRedisSessionStore(false, []byte("secret-key-32-bytes-long-123456"), pool)
			Expect(err).NotTo(HaveOccurred())
			Expect(store).NotTo(BeNil())
		})
	})

	Context("Redis with TLS", func() {
		It("should create a session store with TLS connection", func() {
			// Connect to Redis with TLS using CA certificate
			cfg := database.RedisConfig{
				Address:     "localhost:6380",
				Username:    "redisuser",
				Password:    "redispass",
				Database:    0,
				MaxIdle:     10,
				IdleTimeout: 240 * time.Second,
				TLS: &database.TLSConfig{
					CAFile: "../../../certs/ca.crt", // Use CA cert from docker-compose
				},
			}
			pool, err := cfg.CreateClient()
			Expect(err).NotTo(HaveOccurred())
			defer pool.Close()

			// Test connection with TLS
			conn := pool.Get()
			defer conn.Close()
			_, err = conn.Do("PING")
			Expect(err).NotTo(HaveOccurred())

			// Create Session Store
			store, err := NewRedisSessionStore(true, []byte("secret-key-32-bytes-long-123456"), pool)
			Expect(err).NotTo(HaveOccurred())
			Expect(store).NotTo(BeNil())
		})

		It("should fail to connect with invalid credentials", func() {
			// Connect to Redis with wrong password
			cfg := database.RedisConfig{
				Address:     "localhost:6379",
				Username:    "redisuser",
				Password:    "wrongpass",
				Database:    0,
				MaxIdle:     10,
				IdleTimeout: 240 * time.Second,
			}
			pool, err := cfg.CreateClient()
			Expect(err).NotTo(HaveOccurred())
			defer pool.Close()

			// Test connection should fail with wrong credentials
			conn := pool.Get()
			defer conn.Close()
			_, err = conn.Do("PING")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("WRONGPASS"))
		})
	})

	Context("Redis connection errors", func() {
		It("should handle connection errors gracefully", func() {
			// Connect to invalid Redis address
			cfg := database.RedisConfig{
				Address:     "localhost:9999", // Invalid port
				Database:    0,
				MaxIdle:     10,
				IdleTimeout: 240 * time.Second,
			}
			pool, err := cfg.CreateClient()
			Expect(err).NotTo(HaveOccurred())
			defer pool.Close()

			// Test connection should fail
			conn := pool.Get()
			defer conn.Close()
			_, err = conn.Do("PING")
			Expect(err).To(HaveOccurred())
		})
	})

	Context("Redis database selection", func() {
		It("should connect to different Redis databases", func() {
			// Connect to database 1
			cfg := database.RedisConfig{
				Address:     "localhost:6379",
				Username:    "redisuser",
				Password:    "redispass",
				Database:    1,
				MaxIdle:     10,
				IdleTimeout: 240 * time.Second,
			}
			pool, err := cfg.CreateClient()
			Expect(err).NotTo(HaveOccurred())
			defer pool.Close()

			// Test connection
			conn := pool.Get()
			defer conn.Close()
			_, err = conn.Do("PING")
			Expect(err).NotTo(HaveOccurred())

			// Verify we're on database 1
			// Set a key and verify
			_, err = conn.Do("SET", "test_key", "test_value")
			Expect(err).NotTo(HaveOccurred())

			reply, err := conn.Do("GET", "test_key")
			Expect(err).NotTo(HaveOccurred())
			Expect(string(reply.([]byte))).To(Equal("test_value"))

			// Cleanup
			_, err = conn.Do("DEL", "test_key")
			Expect(err).NotTo(HaveOccurred())

			// Create Session Store
			store, err := NewRedisSessionStore(false, []byte("secret-key-32-bytes-long-123456"), pool)
			Expect(err).NotTo(HaveOccurred())
			Expect(store).NotTo(BeNil())
		})
	})
})
