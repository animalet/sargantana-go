//go:build integration

package database

import (
	"time"

	"github.com/bradfitz/gomemcache/memcache"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Memcached Integration", func() {
	It("should connect to memcached and perform operations", func() {
		cfg := MemcachedConfig{
			Servers:      []string{"localhost:11211"},
			Timeout:      100 * time.Millisecond,
			MaxIdleConns: 2,
		}

		client, err := cfg.CreateClient()
		Expect(err).NotTo(HaveOccurred())

		err = client.Set(&memcache.Item{Key: "test-key", Value: []byte("test-value")})
		Expect(err).NotTo(HaveOccurred())

		item, err := client.Get("test-key")
		Expect(err).NotTo(HaveOccurred())
		Expect(string(item.Value)).To(Equal("test-value"))
	})

	It("should connect with plaintext (no auth)", func() {
		cfg := MemcachedConfig{
			Servers: []string{"localhost:11211"},
		}
		client, err := cfg.CreateClient()
		Expect(err).NotTo(HaveOccurred())

		err = client.Set(&memcache.Item{Key: "test_ping", Value: []byte("pong")})
		Expect(err).NotTo(HaveOccurred())

		item, err := client.Get("test_ping")
		Expect(err).NotTo(HaveOccurred())
		Expect(string(item.Value)).To(Equal("pong"))
	})
})
