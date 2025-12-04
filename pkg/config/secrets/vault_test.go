//go:build unit

package secrets

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Vault Secrets", func() {
	Context("VaultConfig Validate", func() {
		It("should return error if address is empty", func() {
			cfg := VaultConfig{
				Token: "token",
				Path:  "secret/data/myapp",
			}
			err := cfg.Validate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Vault address is required"))
		})

		It("should return error if token is empty", func() {
			cfg := VaultConfig{
				Address: "http://localhost:8200",
				Path:    "secret/data/myapp",
			}
			err := cfg.Validate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Vault token is required"))
		})

		It("should return error if path is empty", func() {
			cfg := VaultConfig{
				Address: "http://localhost:8200",
				Token:   "token",
			}
			err := cfg.Validate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Vault path is required"))
		})

		It("should pass with valid config", func() {
			cfg := VaultConfig{
				Address: "http://localhost:8200",
				Token:   "token",
				Path:    "secret/data/myapp",
			}
			err := cfg.Validate()
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("VaultConfig CreateClient", func() {
		It("should fail with invalid address", func() {
			cfg := VaultConfig{
				Address: "://invalid-url",
				Token:   "token",
				Path:    "secret/data/myapp",
			}
			_, err := cfg.CreateClient()
			Expect(err).To(HaveOccurred())
		})

		It("should create client with namespace", func() {
			cfg := VaultConfig{
				Address:   "http://localhost:8200",
				Token:     "token",
				Path:      "secret/data/myapp",
				Namespace: "myns",
			}
			client, err := cfg.CreateClient()
			Expect(err).NotTo(HaveOccurred())
			Expect(client).NotTo(BeNil())
			// We can't easily verify namespace is set without accessing private fields
			// but we can at least ensure it doesn't error
		})

		It("should create client without namespace", func() {
			cfg := VaultConfig{
				Address: "http://localhost:8200",
				Token:   "token",
				Path:    "secret/data/myapp",
			}
			client, err := cfg.CreateClient()
			Expect(err).NotTo(HaveOccurred())
			Expect(client).NotTo(BeNil())
		})
	})
})
