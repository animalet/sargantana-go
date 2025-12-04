//go:build unit

package database

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("MemcachedConfig", func() {
	Context("Validation", func() {
		It("should validate correct configuration", func() {
			cfg := MemcachedConfig{
				Servers:      []string{"localhost:11211"},
				Timeout:      100 * time.Millisecond,
				MaxIdleConns: 2,
			}
			Expect(cfg.Validate()).To(Succeed())
		})

		It("should fail if servers list is empty", func() {
			cfg := MemcachedConfig{
				Servers: []string{},
			}
			Expect(cfg.Validate()).To(HaveOccurred())
		})

		It("should fail if any server address is empty", func() {
			cfg := MemcachedConfig{
				Servers: []string{"localhost:11211", ""},
			}
			Expect(cfg.Validate()).To(HaveOccurred())
		})

		It("should fail if timeout is negative", func() {
			cfg := MemcachedConfig{
				Servers: []string{"localhost:11211"},
				Timeout: -1,
			}
			Expect(cfg.Validate()).To(HaveOccurred())
		})

		It("should fail if max_idle_conns is negative", func() {
			cfg := MemcachedConfig{
				Servers:      []string{"localhost:11211"},
				MaxIdleConns: -1,
			}
			Expect(cfg.Validate()).To(HaveOccurred())
		})
	})
})
