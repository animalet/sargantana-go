//go:build unit

package secrets

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("FileSecretLoader", func() {
	var (
		loader  SecretLoader
		tempDir string
	)

	BeforeEach(func() {
		var err error
		tempDir, err = os.MkdirTemp("", "secrets-test")
		Expect(err).NotTo(HaveOccurred())
		loader = NewFileSecretLoader(tempDir)
	})

	AfterEach(func() {
		os.RemoveAll(tempDir)
	})

	It("should resolve secret from file", func() {
		err := os.WriteFile(filepath.Join(tempDir, "my-secret"), []byte("secret-value"), 0600)
		Expect(err).NotTo(HaveOccurred())

		val, err := loader.Resolve("my-secret")
		Expect(err).NotTo(HaveOccurred())
		Expect(val).To(Equal("secret-value"))
	})

	It("should return error if file does not exist", func() {
		_, err := loader.Resolve("non-existent")
		Expect(err).To(HaveOccurred())
	})

	It("should trim whitespace", func() {
		err := os.WriteFile(filepath.Join(tempDir, "my-secret"), []byte("  secret-value  \n"), 0600)
		Expect(err).NotTo(HaveOccurred())

		val, err := loader.Resolve("my-secret")
		Expect(err).NotTo(HaveOccurred())
		Expect(val).To(Equal("secret-value"))
	})
})
