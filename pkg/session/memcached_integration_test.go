//go:build integration

package session

import (
	"time"

	"github.com/animalet/sargantana-go/pkg/database"
	"github.com/bradfitz/gomemcache/memcache"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Memcached Session Integration", func() {
	Context("Basic Memcached connection", func() {
		It("should create a session store and connect to Memcached", func() {
			// Connect to Memcached
			cfg := database.MemcachedConfig{
				Servers:      []string{"localhost:11211"},
				Timeout:      500 * time.Millisecond,
				MaxIdleConns: 5,
			}
			client, err := cfg.CreateClient()
			Expect(err).NotTo(HaveOccurred())

			// Test connection
			err = client.Ping()
			Expect(err).NotTo(HaveOccurred())

			// Create Session Store
			store, err := NewMemcachedSessionStore(false, []byte("secret-key-32-bytes-long-123456"), client)
			Expect(err).NotTo(HaveOccurred())
			Expect(store).NotTo(BeNil())
		})

		It("should perform basic set/get operations", func() {
			// Connect to Memcached
			cfg := database.MemcachedConfig{
				Servers:      []string{"localhost:11211"},
				Timeout:      500 * time.Millisecond,
				MaxIdleConns: 5,
			}
			client, err := cfg.CreateClient()
			Expect(err).NotTo(HaveOccurred())

			// Test basic operations
			testItem := &memcache.Item{
				Key:        "test_session_key",
				Value:      []byte("test_session_value"),
				Expiration: 3600,
			}

			err = client.Set(testItem)
			Expect(err).NotTo(HaveOccurred())

			item, err := client.Get("test_session_key")
			Expect(err).NotTo(HaveOccurred())
			Expect(string(item.Value)).To(Equal("test_session_value"))

			// Cleanup
			err = client.Delete("test_session_key")
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("Memcached with multiple servers", func() {
		It("should connect to multiple Memcached servers", func() {
			// Note: For this test to fully work, you'd need multiple Memcached instances
			// For now, we test that the client accepts multiple server addresses
			cfg := database.MemcachedConfig{
				Servers:      []string{"localhost:11211", "localhost:11211"}, // Same server twice for testing
				Timeout:      500 * time.Millisecond,
				MaxIdleConns: 5,
			}
			client, err := cfg.CreateClient()
			Expect(err).NotTo(HaveOccurred())

			// Test connection
			err = client.Ping()
			Expect(err).NotTo(HaveOccurred())

			// Create Session Store
			store, err := NewMemcachedSessionStore(false, []byte("secret-key-32-bytes-long-123456"), client)
			Expect(err).NotTo(HaveOccurred())
			Expect(store).NotTo(BeNil())
		})
	})

	Context("Memcached connection errors", func() {
		It("should handle connection errors gracefully", func() {
			// Connect to invalid Memcached address
			cfg := database.MemcachedConfig{
				Servers:      []string{"localhost:9999"}, // Invalid port
				Timeout:      500 * time.Millisecond,
				MaxIdleConns: 5,
			}
			_, err := cfg.CreateClient()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to connect to Memcached"))
		})

		It("should fail with empty server list", func() {
			// Try to create client with no servers
			cfg := database.MemcachedConfig{
				Servers:      []string{},
				Timeout:      500 * time.Millisecond,
				MaxIdleConns: 5,
			}
			_, err := cfg.CreateClient()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("at least one Memcached server address is required"))
		})
	})

	Context("Memcached timeout configuration", func() {
		It("should respect custom timeout settings", func() {
			// Connect with custom timeout
			cfg := database.MemcachedConfig{
				Servers:      []string{"localhost:11211"},
				Timeout:      2 * time.Second,
				MaxIdleConns: 10,
			}
			client, err := cfg.CreateClient()
			Expect(err).NotTo(HaveOccurred())

			// Verify timeout is set
			Expect(client.Timeout).To(Equal(2 * time.Second))
			Expect(client.MaxIdleConns).To(Equal(10))

			// Test connection
			err = client.Ping()
			Expect(err).NotTo(HaveOccurred())
		})

		It("should use default timeout when not specified", func() {
			// Connect without specifying timeout
			cfg := database.MemcachedConfig{
				Servers: []string{"localhost:11211"},
			}
			client, err := cfg.CreateClient()
			Expect(err).NotTo(HaveOccurred())

			// Verify default timeout is set (100ms)
			Expect(client.Timeout).To(Equal(100 * time.Millisecond))
			Expect(client.MaxIdleConns).To(Equal(2)) // Default max idle conns

			// Test connection works
			err = client.Ping()
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("Memcached session store operations", func() {
		It("should create session store with secure mode", func() {
			// Connect to Memcached
			cfg := database.MemcachedConfig{
				Servers:      []string{"localhost:11211"},
				Timeout:      500 * time.Millisecond,
				MaxIdleConns: 5,
			}
			client, err := cfg.CreateClient()
			Expect(err).NotTo(HaveOccurred())

			// Create Session Store in secure mode
			store, err := NewMemcachedSessionStore(true, []byte("secret-key-32-bytes-long-123456"), client)
			Expect(err).NotTo(HaveOccurred())
			Expect(store).NotTo(BeNil())
		})
	})
})
