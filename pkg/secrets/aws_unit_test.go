//go:build unit

package secrets_test

import (
	"github.com/animalet/sargantana-go/pkg/secrets"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("AWS Secrets", func() {
	Context("AWSConfig Validate", func() {
		It("should return error if region is empty", func() {
			cfg := secrets.AWSConfig{
				AccessKeyID:     "key",
				SecretAccessKey: "secret",
			}
			err := cfg.Validate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("AWS region is required"))
		})

		It("should return error if secret name is empty", func() {
			cfg := secrets.AWSConfig{
				Region: "us-east-1",
			}
			err := cfg.Validate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("AWS secret name is required"))
		})

		It("should pass with valid config", func() {
			cfg := secrets.AWSConfig{
				Region:          "us-east-1",
				AccessKeyID:     "key",
				SecretAccessKey: "secret",
				SecretName:      "my-secret",
			}
			err := cfg.Validate()
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
