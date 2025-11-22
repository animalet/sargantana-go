//go:build unit

package secrets_test

import (
	"github.com/animalet/sargantana-go/pkg/secrets"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Vault Secrets", func() {
	Context("VaultConfig Validate", func() {
		It("should return error if address is empty", func() {
			cfg := secrets.VaultConfig{
				Token: "token",
			}
			err := cfg.Validate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Vault address is required"))
		})

		It("should return error if token is empty", func() {
			cfg := secrets.VaultConfig{
				Address: "http://localhost:8200",
			}
			err := cfg.Validate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Vault token is required"))
		})

		It("should pass with valid config", func() {
			cfg := secrets.VaultConfig{
				Address: "http://localhost:8200",
				Token:   "token",
				Path:    "secret/data/myapp",
			}
			err := cfg.Validate()
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
