//go:build integration

package secrets_test

import (
	"github.com/animalet/sargantana-go/pkg/secrets"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Vault Integration", func() {
	It("should retrieve secrets from Vault (KV v2)", func() {
		cfg := secrets.VaultConfig{
			Address: "http://localhost:8200",
			Token:   "dev-root-token",
			Path:    "secret/data/sargantana",
		}

		client, err := cfg.CreateClient()
		Expect(err).NotTo(HaveOccurred())

		loader := secrets.NewVaultSecretLoader(client, cfg.Path)
		val, err := loader.Resolve("GOOGLE_KEY")
		Expect(err).NotTo(HaveOccurred())
		Expect(val).To(Equal("test-google-key"))
	})

	It("should retrieve secrets from Vault (KV v1)", func() {
		cfg := secrets.VaultConfig{
			Address: "http://localhost:8200",
			Token:   "dev-root-token",
			Path:    "secret-v1/sargantana",
		}

		client, err := cfg.CreateClient()
		Expect(err).NotTo(HaveOccurred())

		loader := secrets.NewVaultSecretLoader(client, cfg.Path)
		val, err := loader.Resolve("GOOGLE_KEY")
		Expect(err).NotTo(HaveOccurred())
		Expect(val).To(Equal("test-google-key"))
	})
})
