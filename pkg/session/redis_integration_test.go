//go:build integration

package session

import (
	"github.com/animalet/sargantana-go/pkg/database"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Redis Session Integration", func() {
	It("should create a session store and connect to Redis", func() {
		// Connect to Redis
		cfg := database.RedisConfig{
			Address:  "localhost:6379",
			Database: 0,
			MaxIdle:  10,
		}
		pool, err := cfg.CreateClient()
		Expect(err).NotTo(HaveOccurred())
		defer pool.Close()

		// Test connection
		conn := pool.Get()
		defer conn.Close()
		_, err = conn.Do("PING")
		Expect(err).NotTo(HaveOccurred())

		// Create Session Store
		store, err := NewRedisSessionStore(false, []byte("secret-key-32-bytes-long-123456"), pool)
		Expect(err).NotTo(HaveOccurred())
		Expect(store).NotTo(BeNil())

		// Since we can't easily test the sessions.Store interface without a request/response recorder and gin context,
		// we mainly verify that the NewRedisSessionStore function succeeds and connects.
		// The underlying implementation is from gin-contrib/sessions which is tested there.
		// We just want to cover our wrapper code.
	})

	It("should handle connection errors gracefully", func() {
		// Connect to invalid Redis address
		cfg := database.RedisConfig{
			Address:  "localhost:9999", // Invalid port
			Database: 0,
			MaxIdle:  10,
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
