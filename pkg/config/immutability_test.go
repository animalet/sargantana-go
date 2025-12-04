package config_test

import (
	"os"
	"path/filepath"

	"github.com/animalet/sargantana-go/pkg/config"
	"github.com/animalet/sargantana-go/pkg/database"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Config Immutability", func() {
	var configFile string
	var cfg *config.Config

	BeforeEach(func() {
		// Create a temporary config file for testing
		configFile = filepath.Join(os.TempDir(), "test-config.yaml")
		configContent := `
memcached:
  servers:
    - server1:11211
    - server2:11211
  timeout: 100ms

redis:
  address: localhost:6379
  max_idle: 10
  idle_timeout: 240s
  tls:
    insecure_skip_verify: false
    cert_file: /path/to/cert.pem
    key_file: /path/to/key.pem
    ca_file: /path/to/ca.pem
`
		err := os.WriteFile(configFile, []byte(configContent), 0600)
		Expect(err).NotTo(HaveOccurred())

		cfg, err = config.NewConfig(configFile)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		_ = os.Remove(configFile)
	})

	Context("Get[T] returns immutable configs", func() {
		It("should not affect original when modifying slice fields", func() {
			// Get config twice
			cfg1, err := config.Get[database.MemcachedConfig](cfg, "memcached")
			Expect(err).NotTo(HaveOccurred())
			Expect(cfg1).NotTo(BeNil())

			cfg2, err := config.Get[database.MemcachedConfig](cfg, "memcached")
			Expect(err).NotTo(HaveOccurred())
			Expect(cfg2).NotTo(BeNil())

			// Verify they start with the same values
			Expect(cfg1.Servers).To(Equal(cfg2.Servers))
			originalFirst := cfg1.Servers[0]

			// Modify first config's slice
			cfg1.Servers[0] = "modified:11211"

			// Verify second config unchanged
			Expect(cfg2.Servers[0]).To(Equal(originalFirst))
			Expect(cfg2.Servers[0]).NotTo(Equal("modified:11211"))
		})

		It("should not affect original when appending to slice fields", func() {
			// Get config twice
			cfg1, err := config.Get[database.MemcachedConfig](cfg, "memcached")
			Expect(err).NotTo(HaveOccurred())

			cfg2, err := config.Get[database.MemcachedConfig](cfg, "memcached")
			Expect(err).NotTo(HaveOccurred())

			// Store original length
			originalLen := len(cfg2.Servers)

			// Append to first config's slice
			cfg1.Servers = append(cfg1.Servers, "server3:11211")

			// Verify second config unchanged
			Expect(len(cfg2.Servers)).To(Equal(originalLen))
			Expect(cfg2.Servers).NotTo(ContainElement("server3:11211"))
		})

		It("should not affect original when modifying nested pointer fields", func() {
			// Get config twice
			cfg1, err := config.Get[database.RedisConfig](cfg, "redis")
			Expect(err).NotTo(HaveOccurred())
			Expect(cfg1).NotTo(BeNil())
			Expect(cfg1.TLS).NotTo(BeNil())

			cfg2, err := config.Get[database.RedisConfig](cfg, "redis")
			Expect(err).NotTo(HaveOccurred())
			Expect(cfg2).NotTo(BeNil())
			Expect(cfg2.TLS).NotTo(BeNil())

			// Store original value
			originalCertFile := cfg2.TLS.CertFile

			// Modify nested pointer in first config
			cfg1.TLS.CertFile = "/modified/cert.pem"

			// Verify second config unchanged
			Expect(cfg2.TLS.CertFile).To(Equal(originalCertFile))
			Expect(cfg2.TLS.CertFile).NotTo(Equal("/modified/cert.pem"))
		})

		It("should not affect original when replacing nested pointer", func() {
			// Get config twice
			cfg1, err := config.Get[database.RedisConfig](cfg, "redis")
			Expect(err).NotTo(HaveOccurred())
			Expect(cfg1).NotTo(BeNil())

			cfg2, err := config.Get[database.RedisConfig](cfg, "redis")
			Expect(err).NotTo(HaveOccurred())
			Expect(cfg2).NotTo(BeNil())

			// Replace entire TLS pointer in first config
			cfg1.TLS = &database.TLSConfig{
				InsecureSkipVerify: true,
				CertFile:           "/new/cert.pem",
			}

			// Verify second config's TLS unchanged
			Expect(cfg2.TLS.InsecureSkipVerify).To(BeFalse())
			Expect(cfg2.TLS.CertFile).To(Equal("/path/to/cert.pem"))
		})

		It("should not affect original when modifying primitive fields", func() {
			// Get config twice
			cfg1, err := config.Get[database.RedisConfig](cfg, "redis")
			Expect(err).NotTo(HaveOccurred())

			cfg2, err := config.Get[database.RedisConfig](cfg, "redis")
			Expect(err).NotTo(HaveOccurred())

			// Store original value
			originalAddress := cfg2.Address

			// Modify primitive field in first config
			cfg1.Address = "modified:6379"

			// Verify second config unchanged
			Expect(cfg2.Address).To(Equal(originalAddress))
			Expect(cfg2.Address).NotTo(Equal("modified:6379"))
		})
	})

	Context("GetClientAndConfig returns immutable config", func() {
		It("should not affect original when modifying returned config", func() {
			// Get client and config twice
			client1, cfg1, err := config.GetClientAndConfig[database.MemcachedConfig](cfg, "memcached")
			Expect(err).NotTo(HaveOccurred())
			Expect(client1).NotTo(BeNil())
			Expect(cfg1).NotTo(BeNil())

			client2, cfg2, err := config.GetClientAndConfig[database.MemcachedConfig](cfg, "memcached")
			Expect(err).NotTo(HaveOccurred())
			Expect(client2).NotTo(BeNil())
			Expect(cfg2).NotTo(BeNil())

			// Store original value
			originalFirst := cfg2.Servers[0]

			// Modify first config
			cfg1.Servers[0] = "modified:11211"

			// Verify second config unchanged
			Expect(cfg2.Servers[0]).To(Equal(originalFirst))
			Expect(cfg2.Servers[0]).NotTo(Equal("modified:11211"))
		})

		It("should return different client instances", func() {
			// Get client twice
			client1, _, err := config.GetClientAndConfig[database.MemcachedConfig](cfg, "memcached")
			Expect(err).NotTo(HaveOccurred())

			client2, _, err := config.GetClientAndConfig[database.MemcachedConfig](cfg, "memcached")
			Expect(err).NotTo(HaveOccurred())

			// Clients should be different instances
			// (they're not the same pointer)
			Expect(client1).NotTo(BeIdenticalTo(client2))
		})
	})

	Context("Nil handling", func() {
		It("should return nil for non-existent configs", func() {
			cfg1, err := config.Get[database.RedisConfig](cfg, "nonexistent")
			Expect(err).NotTo(HaveOccurred())
			Expect(cfg1).To(BeNil())

			// Should be safe to call multiple times
			cfg2, err := config.Get[database.RedisConfig](cfg, "nonexistent")
			Expect(err).NotTo(HaveOccurred())
			Expect(cfg2).To(BeNil())
		})

		It("should return nil client and config for non-existent configs", func() {
			client, configResult, err := config.GetClientAndConfig[database.RedisConfig](cfg, "nonexistent")
			Expect(err).NotTo(HaveOccurred())
			Expect(client).To(BeNil())
			Expect(configResult).To(BeNil())
		})
	})
})
