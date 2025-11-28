//go:build unit

package session

import (
	"github.com/bradfitz/gomemcache/memcache"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Memcached Session Store", func() {
	Context("NewMemcachedSessionStore", func() {
		It("should return error if client is nil", func() {
			_, err := NewMemcachedSessionStore(true, []byte("secret"), nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Memcached client cannot be nil"))
		})

		It("should return error if secret is empty", func() {
			client := memcache.New("localhost:11211")
			_, err := NewMemcachedSessionStore(true, nil, client)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("session secret cannot be empty"))
		})

		It("should create store if valid", func() {
			client := memcache.New("localhost:11211")
			store, err := NewMemcachedSessionStore(true, []byte("secret"), client)
			Expect(err).NotTo(HaveOccurred())
			Expect(store).NotTo(BeNil())
		})

		It("should create store in non-secure mode", func() {
			client := memcache.New("localhost:11211")
			store, err := NewMemcachedSessionStore(false, []byte("secret-key-123"), client)
			Expect(err).NotTo(HaveOccurred())
			Expect(store).NotTo(BeNil())
		})
	})
})
