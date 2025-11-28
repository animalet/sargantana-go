//go:build unit

package database

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("PostgresConfig", func() {
	Context("Validation", func() {
		It("should validate correct configuration", func() {
			cfg := PostgresConfig{
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
			cfg := PostgresConfig{
				Port:     5432,
				Database: "testdb",
				User:     "user",
				Password: "password",
			}
			Expect(cfg.Validate()).To(HaveOccurred())
		})

		It("should fail if port is missing", func() {
			cfg := PostgresConfig{
				Host:     "localhost",
				Database: "testdb",
				User:     "user",
				Password: "password",
			}
			Expect(cfg.Validate()).To(HaveOccurred())
		})

		It("should fail if database is missing", func() {
			cfg := PostgresConfig{
				Host:     "localhost",
				Port:     5432,
				User:     "user",
				Password: "password",
			}
			Expect(cfg.Validate()).To(HaveOccurred())
		})

		It("should fail if user is missing", func() {
			cfg := PostgresConfig{
				Host:     "localhost",
				Port:     5432,
				Database: "testdb",
				Password: "password",
			}
			Expect(cfg.Validate()).To(HaveOccurred())
		})

		It("should fail if password is missing", func() {
			cfg := PostgresConfig{
				Host:     "localhost",
				Port:     5432,
				Database: "testdb",
				User:     "user",
			}
			Expect(cfg.Validate()).To(HaveOccurred())
		})

		It("should fail with invalid ssl mode", func() {
			cfg := PostgresConfig{
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
			cfg := PostgresConfig{
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

		It("should fail with negative MinConns", func() {
			cfg := PostgresConfig{
				Host:     "localhost",
				Port:     5432,
				Database: "testdb",
				User:     "user",
				Password: "password",
				MinConns: -1,
			}
			Expect(cfg.Validate()).To(HaveOccurred())
			Expect(cfg.Validate().Error()).To(ContainSubstring("min_conns must be non-negative"))
		})

		It("should fail with negative MaxConnLifetime", func() {
			cfg := PostgresConfig{
				Host:            "localhost",
				Port:            5432,
				Database:        "testdb",
				User:            "user",
				Password:        "password",
				MaxConnLifetime: -1 * time.Hour,
			}
			Expect(cfg.Validate()).To(HaveOccurred())
			Expect(cfg.Validate().Error()).To(ContainSubstring("max_conn_lifetime must be non-negative"))
		})

		It("should fail with negative MaxConnIdleTime", func() {
			cfg := PostgresConfig{
				Host:            "localhost",
				Port:            5432,
				Database:        "testdb",
				User:            "user",
				Password:        "password",
				MaxConnIdleTime: -1 * time.Hour,
			}
			Expect(cfg.Validate()).To(HaveOccurred())
			Expect(cfg.Validate().Error()).To(ContainSubstring("max_conn_idle_time must be non-negative"))
		})

		It("should fail with negative HealthCheckPeriod", func() {
			cfg := PostgresConfig{
				Host:              "localhost",
				Port:              5432,
				Database:          "testdb",
				User:              "user",
				Password:          "password",
				HealthCheckPeriod: -1 * time.Hour,
			}
			Expect(cfg.Validate()).To(HaveOccurred())
			Expect(cfg.Validate().Error()).To(ContainSubstring("health_check_period must be non-negative"))
		})

		It("should validate correct pool settings", func() {
			cfg := PostgresConfig{
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

	Context("Connection String Building", func() {
		It("should build connection string with default SSL mode", func() {
			cfg := PostgresConfig{
				Host:     "localhost",
				Port:     5432,
				Database: "testdb",
				User:     "user",
				Password: "password",
			}
			connStr := cfg.buildConnectionString()
			Expect(connStr).To(ContainSubstring("sslmode=prefer"))
		})

		It("should build connection string with specified SSL mode", func() {
			cfg := PostgresConfig{
				Host:     "localhost",
				Port:     5432,
				Database: "testdb",
				User:     "user",
				Password: "password",
				SSLMode:  "require",
			}
			connStr := cfg.buildConnectionString()
			Expect(connStr).To(ContainSubstring("sslmode=require"))
		})
	})
})
