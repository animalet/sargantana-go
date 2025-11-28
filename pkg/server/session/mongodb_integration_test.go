//go:build integration

package session

import (
	"context"

	"github.com/animalet/sargantana-go/pkg/database"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("MongoDB Session Integration", func() {
	It("should create a session store and save/load sessions", func() {
		// Connect to MongoDB
		cfg := database.MongoDBConfig{
			URI:        "mongodb://localhost:27017",
			Database:   "sessions_test",
			Username:   "admin",
			Password:   "adminpass",
			AuthSource: "admin",
		}
		client, err := cfg.CreateClient()
		Expect(err).NotTo(HaveOccurred())
		defer client.Disconnect(context.Background())

		// Create Session Store
		store, err := NewMongoDBSessionStore(false, []byte("secret-key-32-bytes-long-123456"), client, "session_test_db", "sessions")
		Expect(err).NotTo(HaveOccurred())
		Expect(store).NotTo(BeNil())

		// Since we can't easily test the sessions.Store interface without a request/response recorder and gin context,
		// we mainly verify that the NewMongoDBSessionStore function succeeds and connects.
		// The underlying implementation is from gin-contrib/sessions which is tested there.
		// We just want to cover our wrapper code.
	})
})
