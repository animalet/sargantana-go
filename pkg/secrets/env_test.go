//go:build unit

package secrets

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("EnvLoader", func() {
	var loader SecretLoader

	BeforeEach(func() {
		loader = NewEnvLoader()
	})

	It("should resolve environment variable", func() {
		os.Setenv("TEST_ENV_VAR", "test-value")
		defer os.Unsetenv("TEST_ENV_VAR")

		val, err := loader.Resolve("TEST_ENV_VAR")
		Expect(err).NotTo(HaveOccurred())
		Expect(val).To(Equal("test-value"))
	})

	It("should return empty string if env var not set", func() {
		val, err := loader.Resolve("NON_EXISTENT_VAR")
		Expect(err).NotTo(HaveOccurred())
		Expect(val).To(Equal(""))
	})

	It("should return correct name", func() {
		Expect(loader.Name()).To(Equal("Environment"))
	})
})
