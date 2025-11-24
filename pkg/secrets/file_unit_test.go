//go:build unit

package secrets

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("FileSecretLoader", func() {
	var tempDir string

	BeforeEach(func() {
		var err error
		tempDir, err = os.MkdirTemp("", "secrets_test")
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		os.RemoveAll(tempDir)
	})

	Context("Validate", func() {
		It("should return error if secrets_dir is empty", func() {
			cfg := FileSecretConfig{}
			err := cfg.Validate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("secrets_dir is required"))
		})

		It("should return error if secrets_dir does not exist", func() {
			cfg := FileSecretConfig{SecretsDir: "/non/existent/dir"}
			err := cfg.Validate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("does not exist"))
		})

		It("should return error if secrets_dir is not a directory", func() {
			path := filepath.Join(tempDir, "file")
			err := os.WriteFile(path, []byte("content"), 0644)
			Expect(err).NotTo(HaveOccurred())

			cfg := FileSecretConfig{SecretsDir: path}
			err = cfg.Validate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("is not a directory"))
		})

		It("should pass if secrets_dir exists and is a directory", func() {
			cfg := FileSecretConfig{SecretsDir: tempDir}
			err := cfg.Validate()
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("CreateClient", func() {
		It("should return error if validation fails", func() {
			cfg := FileSecretConfig{}
			_, err := cfg.CreateClient()
			Expect(err).To(HaveOccurred())
		})

		It("should create client if validation passes", func() {
			cfg := FileSecretConfig{SecretsDir: tempDir}
			client, err := cfg.CreateClient()
			Expect(err).NotTo(HaveOccurred())
			Expect(client).NotTo(BeNil())
		})
	})

	Context("Resolve", func() {
		It("should return error if secrets_dir is not configured", func() {
			loader := NewFileSecretLoader("")
			_, err := loader.Resolve("key")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("no secrets directory configured"))
		})

		It("should return error if key is empty", func() {
			loader := NewFileSecretLoader(tempDir)
			_, err := loader.Resolve("")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("no file specified"))
		})

		It("should return error if file reading fails", func() {
			loader := NewFileSecretLoader(tempDir)
			_, err := loader.Resolve("non_existent_file")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("error reading secret file"))
		})

		It("should return secret content if file exists", func() {
			err := os.WriteFile(filepath.Join(tempDir, "my_secret"), []byte("  secret_value  \n"), 0644)
			Expect(err).NotTo(HaveOccurred())

			loader := NewFileSecretLoader(tempDir)
			val, err := loader.Resolve("my_secret")
			Expect(err).NotTo(HaveOccurred())
			Expect(val).To(Equal("secret_value"))
		})
	})

	Context("Name", func() {
		It("should return File", func() {
			loader := NewFileSecretLoader(tempDir)
			Expect(loader.Name()).To(Equal("File"))
		})
	})
})
