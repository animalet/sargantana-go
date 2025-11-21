//go:build unit

package database_test

import (
	"time"

	"github.com/animalet/sargantana-go/pkg/database"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("PostgresConfig", func() {
	Context("Validation", func() {
		It("should validate correct configuration", func() {
			cfg := database.PostgresConfig{
				Host:     "localhost",
				Port:     5432,
				Database: "testdb",
				User:     "user",
				Password: "password",
				SSLMode:  "disable",
			}
			Expect(cfg.Validate()).To(Succeed())
		})

		It("should fail if host is missing", func() {
			cfg := database.PostgresConfig{
				Port:     5432,
				Database: "testdb",
				User:     "user",
				Password: "password",
			}
			Expect(cfg.Validate()).To(HaveOccurred())
		})

		It("should fail if port is missing", func() {
			cfg := database.PostgresConfig{
				Host:     "localhost",
				Database: "testdb",
				User:     "user",
				Password: "password",
			}
			Expect(cfg.Validate()).To(HaveOccurred())
		})

		It("should fail if database is missing", func() {
			cfg := database.PostgresConfig{
				Host:     "localhost",
				Port:     5432,
				User:     "user",
				Password: "password",
			}
			Expect(cfg.Validate()).To(HaveOccurred())
		})

		It("should fail if user is missing", func() {
			cfg := database.PostgresConfig{
				Host:     "localhost",
				Port:     5432,
				Database: "testdb",
				Password: "password",
			}
			Expect(cfg.Validate()).To(HaveOccurred())
		})

		It("should fail if password is missing", func() {
			cfg := database.PostgresConfig{
				Host:     "localhost",
				Port:     5432,
				Database: "testdb",
				User:     "user",
			}
			Expect(cfg.Validate()).To(HaveOccurred())
		})

		It("should fail with invalid ssl mode", func() {
			cfg := database.PostgresConfig{
				Host:     "localhost",
				Port:     5432,
				Database: "testdb",
				User:     "user",
				Password: "password",
				SSLMode:  "invalid",
			}
			Expect(cfg.Validate()).To(HaveOccurred())
		})

		It("should fail with invalid pool settings", func() {
			cfg := database.PostgresConfig{
				Host:     "localhost",
				Port:     5432,
				Database: "testdb",
				User:     "user",
				Password: "password",
				MaxConns: -1,
			}
			Expect(cfg.Validate()).To(HaveOccurred())

			cfg.MaxConns = 10
			cfg.MinConns = 20
			Expect(cfg.Validate()).To(HaveOccurred())
		})

		It("should validate correct pool settings", func() {
			cfg := database.PostgresConfig{
				Host:            "localhost",
				Port:            5432,
				Database:        "testdb",
				User:            "user",
				Password:        "password",
				MaxConns:        20,
				MinConns:        5,
				MaxConnLifetime: time.Hour,
			}
			Expect(cfg.Validate()).To(Succeed())
		})
	})
})
