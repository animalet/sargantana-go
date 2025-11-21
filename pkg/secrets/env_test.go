//go:build unit

package secrets_test

import (
	"os"

	"github.com/animalet/sargantana-go/pkg/secrets"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("EnvLoader", func() {
	var loader secrets.SecretLoader

	BeforeEach(func() {
		loader = secrets.NewEnvLoader()
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
})
