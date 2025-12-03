//go:build unit

package session

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.mongodb.org/mongo-driver/mongo"
)

var _ = Describe("MongoDB Session Store", func() {
	Context("NewMongoDBSessionStore", func() {
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
})
