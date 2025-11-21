//go:build integration

package database_test

import (
	"time"

	"github.com/animalet/sargantana-go/pkg/database"
	"github.com/bradfitz/gomemcache/memcache"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Memcached Integration", func() {
	It("should connect to memcached and perform operations", func() {
		cfg := database.MemcachedConfig{
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
})
