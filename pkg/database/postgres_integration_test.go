//go:build integration

package database_test

import (
	"context"

	"github.com/animalet/sargantana-go/pkg/database"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("PostgreSQL Integration", func() {
	It("should connect to postgres and perform operations", func() {
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

		var result int
		err = pool.QueryRow(context.Background(), "SELECT 1").Scan(&result)
		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal(1))
	})

	It("should connect with default settings (sslmode=prefer)", func() {
		cfg := database.PostgresConfig{
			Host:     "localhost",
			Port:     5432,
			User:     "user",
			Password: "password",
			Database: "my_blog_db",
			SSLMode:  "prefer",
		}
		pool, err := cfg.CreateClient()
		Expect(err).NotTo(HaveOccurred())
		defer pool.Close()

		err = pool.Ping(context.Background())
		Expect(err).NotTo(HaveOccurred())
	})
})
