//go:build integration

package session_test

import (
	"github.com/animalet/sargantana-go/pkg/database"
	"github.com/animalet/sargantana-go/pkg/session"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Postgres Session Integration", func() {
	It("should create a session store", func() {
		// Connect to Postgres
		cfg := database.PostgresConfig{
			Host:     "localhost",
			Port:     5432,
			User:     "user",
			Password: "password",
			Database: "my_blog_db",
			SSLMode:  "disable",
		}
		pool, err := cfg.CreateClient()
		Expect(err).NotTo(HaveOccurred())
		defer pool.Close()

		// Create Session Store
		store, err := session.NewPostgresSessionStore(false, []byte("secret-key-32-bytes-long-123456"), pool, "sessions")
		Expect(err).NotTo(HaveOccurred())
		Expect(store).NotTo(BeNil())
	})
})
