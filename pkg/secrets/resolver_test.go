//go:build unit

package secrets_test

import (
	"github.com/animalet/sargantana-go/pkg/secrets"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
)

type MockLoader struct {
	Secrets map[string]string
}

func (m *MockLoader) Resolve(key string) (string, error) {
	if val, ok := m.Secrets[key]; ok {
		return val, nil
	}
	return "", errors.New("secret not found")
}

func (m *MockLoader) Name() string {
	return "mock"
}

var _ = Describe("Resolver", func() {
	var mockLoader *MockLoader

	BeforeEach(func() {
		mockLoader = &MockLoader{
			Secrets: map[string]string{
				"key1": "value1",
			},
		}
		secrets.Register("test", mockLoader)
	})

	It("should resolve secret using registered provider", func() {
		val, err := secrets.Resolve("test:key1")
		Expect(err).NotTo(HaveOccurred())
		Expect(val).To(Equal("value1"))
	})

	It("should return error if provider not found", func() {
		_, err := secrets.Resolve("unknown:key1")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("no secret provider registered"))
	})

	It("should return error if secret not found", func() {
		_, err := secrets.Resolve("test:key2")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("failed to resolve secret"))
	})

	It("should default to env provider if no prefix", func() {
		// Assuming env provider is registered by default and we can mock it or set env var
		// But since we can't easily mock the default env provider without replacing it,
		// let's register a mock as "env" for this test if possible.
		// However, secrets.Register("env", ...) might have side effects if parallel tests run.
		// For unit tests, we should be careful.
		// Let's skip this or rely on the fact that we can overwrite "env" provider.
		secrets.Register("env", mockLoader)
		val, err := secrets.Resolve("key1")
		Expect(err).NotTo(HaveOccurred())
		Expect(val).To(Equal("value1"))
	})
})
