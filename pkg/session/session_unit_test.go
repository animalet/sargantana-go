//go:build unit

package session

import (
	"github.com/bradfitz/gomemcache/memcache"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gomodule/redigo/redis"
	"github.com/jackc/pgx/v5/pgxpool"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.mongodb.org/mongo-driver/mongo"
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

	Context("Memcached", func() {
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
	})

	Context("MongoDB", func() {
		It("should return error if client is nil", func() {
			_, err := NewMongoDBSessionStore(true, []byte("secret"), nil, "db", "coll")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("MongoDB client cannot be nil"))
		})

		It("should return error if secret is empty", func() {
			client := &mongo.Client{}
			_, err := NewMongoDBSessionStore(true, nil, client, "db", "coll")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("session secret cannot be empty"))
		})

		It("should return error if database is empty", func() {
			client := &mongo.Client{}
			_, err := NewMongoDBSessionStore(true, []byte("secret"), client, "", "coll")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("database name cannot be empty"))
		})
	})

	Context("Postgres", func() {
		It("should return error if pool is nil", func() {
			_, err := NewPostgresSessionStore(true, []byte("secret"), nil, "table")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("PostgreSQL pool cannot be nil"))
		})

		It("should return error if secret is empty", func() {
			pool := &pgxpool.Pool{}
			_, err := NewPostgresSessionStore(true, nil, pool, "table")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("session secret cannot be empty"))
		})
	})

	Context("Redis", func() {
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

		It("should return error if pool is nil", func() {
			_, err := NewRedisSessionStore(true, []byte("secret"), nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Redis pool cannot be nil"))
		})
	})
})
