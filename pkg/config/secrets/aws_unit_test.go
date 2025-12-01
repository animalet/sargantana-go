//go:build unit

package secrets

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("AWS Secrets", func() {
	Context("AWSConfig Validate", func() {
		It("should return error if region is empty", func() {
			cfg := AWSConfig{
				AccessKeyID:     "key",
				SecretAccessKey: "secret",
			}
			err := cfg.Validate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("AWS region is required"))
		})

		It("should return error if secret name is empty", func() {
			cfg := AWSConfig{
				Region: "us-east-1",
			}
			err := cfg.Validate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("AWS secret name is required"))
		})

		It("should pass with valid config", func() {
			cfg := AWSConfig{
				Region:          "us-east-1",
				AccessKeyID:     "key",
				SecretAccessKey: "secret",
				SecretName:      "my-secret",
			}
			err := cfg.Validate()
			Expect(err).NotTo(HaveOccurred())
		})

		It("should pass without credentials (uses default credential chain)", func() {
			cfg := AWSConfig{
				Region:     "us-east-1",
				SecretName: "my-secret",
			}
			err := cfg.Validate()
			Expect(err).NotTo(HaveOccurred())
		})

		It("should pass with custom endpoint", func() {
			cfg := AWSConfig{
				Region:     "us-east-1",
				SecretName: "my-secret",
				Endpoint:   "http://localhost:4566",
			}
			err := cfg.Validate()
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("CreateClient", func() {
		It("should create client with credentials", func() {
			cfg := AWSConfig{
				Region:          "us-east-1",
				AccessKeyID:     "test-key",
				SecretAccessKey: "test-secret",
				SecretName:      "test-secret",
			}
			client, err := cfg.CreateClient()
			// We expect this to succeed locally even without AWS credentials
			// because we're just creating the client, not calling AWS
			Expect(err).NotTo(HaveOccurred())
			Expect(client).NotTo(BeNil())
		})

		It("should create client without credentials", func() {
			cfg := AWSConfig{
				Region:     "us-east-1",
				SecretName: "test-secret",
			}
			client, err := cfg.CreateClient()
			Expect(err).NotTo(HaveOccurred())
			Expect(client).NotTo(BeNil())
		})

		It("should create client with endpoint", func() {
			cfg := AWSConfig{
				Region:     "us-east-1",
				SecretName: "test-secret",
				Endpoint:   "http://localhost:4566",
			}
			client, err := cfg.CreateClient()
			Expect(err).NotTo(HaveOccurred())
			Expect(client).NotTo(BeNil())
		})
	})

	Context("AWSSecretLoader", func() {
		It("should return Name", func() {
			cfg := AWSConfig{
				Region:     "us-east-1",
				SecretName: "test-secret",
			}
			client, _ := cfg.CreateClient()
			loader := NewAWSSecretLoader(client, cfg.SecretName)
			Expect(loader.Name()).To(Equal("AWS Secrets Manager"))
		})
	})
})
