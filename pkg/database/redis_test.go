//go:build unit

package database

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("RedisConfig", func() {
	Context("Validation", func() {
		It("should validate correct configuration", func() {
			cfg := RedisConfig{
				Address:     "localhost:6379",
				MaxIdle:     10,
				IdleTimeout: time.Minute,
			}
			Expect(cfg.Validate()).To(Succeed())
		})

		It("should fail if address is missing", func() {
			cfg := RedisConfig{
				MaxIdle: 10,
			}
			Expect(cfg.Validate()).To(HaveOccurred())
		})

		It("should fail if max_idle is negative", func() {
			cfg := RedisConfig{
				Address: "localhost:6379",
				MaxIdle: -1,
			}
			Expect(cfg.Validate()).To(HaveOccurred())
		})

		It("should fail if idle_timeout is negative", func() {
			cfg := RedisConfig{
				Address:     "localhost:6379",
				IdleTimeout: -1,
			}
			Expect(cfg.Validate()).To(HaveOccurred())
		})

		It("should fail if database is negative", func() {
			cfg := RedisConfig{
				Address:  "localhost:6379",
				Database: -1,
			}
			Expect(cfg.Validate()).To(HaveOccurred())
		})

		It("should validate TLS configuration", func() {
			cfg := RedisConfig{
				Address: "localhost:6379",
				TLS: &TLSConfig{
					CertFile: "cert.pem",
					KeyFile:  "key.pem",
				},
			}
			Expect(cfg.Validate()).To(Succeed())
		})

		It("should fail if TLS cert is set without key", func() {
			cfg := RedisConfig{
				Address: "localhost:6379",
				TLS: &TLSConfig{
					CertFile: "cert.pem",
				},
			}
			Expect(cfg.Validate()).To(HaveOccurred())
		})
	})
})
