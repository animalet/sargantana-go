//go:build integration

package secrets_test

import (
	"github.com/animalet/sargantana-go/pkg/secrets"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("AWS Secrets Manager Integration", func() {
	It("should retrieve JSON secrets from AWS Secrets Manager", func() {
		cfg := secrets.AWSConfig{
			Region:          "us-east-1",
			AccessKeyID:     "test",
			SecretAccessKey: "test",
			SecretName:      "sargantana/test",
			Endpoint:        "http://localhost:4566",
		}

		client, err := cfg.CreateClient()
		Expect(err).NotTo(HaveOccurred())

		loader := secrets.NewAWSSecretLoader(client, cfg.SecretName)
		val, err := loader.Resolve("GOOGLE_KEY")
		Expect(err).NotTo(HaveOccurred())
		Expect(val).To(Equal("test-google-key"))
	})

	It("should retrieve plain text secrets from AWS Secrets Manager", func() {
		cfg := secrets.AWSConfig{
			Region:          "us-east-1",
			AccessKeyID:     "test",
			SecretAccessKey: "test",
			SecretName:      "sargantana/plain-secret",
			Endpoint:        "http://localhost:4566",
		}

		client, err := cfg.CreateClient()
		Expect(err).NotTo(HaveOccurred())

		loader := secrets.NewAWSSecretLoader(client, cfg.SecretName)
		// For plain text secrets, the key is ignored
		val, err := loader.Resolve("ignored-key")
		Expect(err).NotTo(HaveOccurred())
		Expect(val).To(Equal("plain-text-secret-value"))
	})
})
