//go:build unit

package session_test

import (
	"github.com/animalet/sargantana-go/pkg/session"
	"github.com/bradfitz/gomemcache/memcache"
	"github.com/jackc/pgx/v5/pgxpool"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.mongodb.org/mongo-driver/mongo"
)

var _ = Describe("Session Stores", func() {
	Context("Memcached", func() {
		It("should return error if client is nil", func() {
			_, err := session.NewMemcachedSessionStore(true, []byte("secret"), nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Memcached client cannot be nil"))
		})

		It("should return error if secret is empty", func() {
			client := memcache.New("localhost:11211")
			_, err := session.NewMemcachedSessionStore(true, nil, client)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("session secret cannot be empty"))
		})

		It("should create store if valid", func() {
			client := memcache.New("localhost:11211")
			store, err := session.NewMemcachedSessionStore(true, []byte("secret"), client)
			Expect(err).NotTo(HaveOccurred())
			Expect(store).NotTo(BeNil())
		})
	})

	Context("MongoDB", func() {
		It("should return error if client is nil", func() {
			_, err := session.NewMongoDBSessionStore(true, []byte("secret"), nil, "db", "coll")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("MongoDB client cannot be nil"))
		})

		It("should return error if secret is empty", func() {
			client := &mongo.Client{}
			_, err := session.NewMongoDBSessionStore(true, nil, client, "db", "coll")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("session secret cannot be empty"))
		})

		It("should return error if database is empty", func() {
			client := &mongo.Client{}
			_, err := session.NewMongoDBSessionStore(true, []byte("secret"), client, "", "coll")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("database name cannot be empty"))
		})
	})

	Context("Postgres", func() {
		It("should return error if pool is nil", func() {
			_, err := session.NewPostgresSessionStore(true, []byte("secret"), nil, "table")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("PostgreSQL pool cannot be nil"))
		})

		It("should return error if secret is empty", func() {
			pool := &pgxpool.Pool{}
			_, err := session.NewPostgresSessionStore(true, nil, pool, "table")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("session secret cannot be empty"))
		})
	})

	Context("Redis", func() {
		It("should return error if pool is nil", func() {
			_, err := session.NewRedisSessionStore(true, []byte("secret"), nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Redis pool cannot be nil"))
		})
	})
})
