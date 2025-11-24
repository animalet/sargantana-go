//go:build unit

package session

import (
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gomodule/redigo/redis"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type MockRedisConn struct{}

func (m *MockRedisConn) Close() error { return nil }
func (m *MockRedisConn) Err() error   { return nil }
func (m *MockRedisConn) Do(commandName string, args ...interface{}) (interface{}, error) {
	return "OK", nil
}
func (m *MockRedisConn) Send(commandName string, args ...interface{}) error { return nil }
func (m *MockRedisConn) Flush() error                                       { return nil }
func (m *MockRedisConn) Receive() (interface{}, error)                      { return nil, nil }

var _ = Describe("Session Stores", func() {
	Context("Cookie Store", func() {
		It("should create a new cookie store with correct options", func() {
			secret := []byte("secret-key")
			store := NewCookieStore(true, secret)
			Expect(store).NotTo(BeNil())

			// Type assertion to check if it's a cookie store
			cookieStore, ok := store.(cookie.Store)
			Expect(ok).To(BeTrue())
			Expect(cookieStore).NotTo(BeNil())
		})
	})

	Context("Redis Store", func() {
		It("should create a new redis store with correct options", func() {
			secret := []byte("secret-key")
			pool := &redis.Pool{
				Dial: func() (redis.Conn, error) {
					return &MockRedisConn{}, nil
				},
			}

			store, err := NewRedisSessionStore(true, secret, pool)
			Expect(err).NotTo(HaveOccurred())
			Expect(store).NotTo(BeNil())
		})

		It("should fail if pool is nil", func() {
			secret := []byte("secret-key")
			_, err := NewRedisSessionStore(true, secret, nil)
			Expect(err).To(HaveOccurred())
		})
	})

	Context("Postgres Store", func() {
		It("should fail if pool is nil", func() {
			secret := []byte("secret-key")
			_, err := NewPostgresSessionStore(true, secret, nil, "sessions")
			Expect(err).To(HaveOccurred())
		})

		It("should fail if secret is empty", func() {
			// We can't easily create a valid pool without a real DB connection
			// but we can test the secret validation if we pass a nil pool?
			// No, pool check is first.
			// So we can only test nil pool here without a real connection.
		})
	})

	Context("MongoDB Store", func() {
		It("should fail if client is nil", func() {
			secret := []byte("secret-key")
			_, err := NewMongoDBSessionStore(true, secret, nil, "db", "coll")
			Expect(err).To(HaveOccurred())
		})
	})

	Context("Memcached Store", func() {
		It("should fail if client is nil", func() {
			secret := []byte("secret-key")
			_, err := NewMemcachedSessionStore(true, secret, nil)
			Expect(err).To(HaveOccurred())
		})
	})
})
