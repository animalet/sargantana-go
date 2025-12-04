//go:build unit

package expansion_test

import (
	"github.com/animalet/sargantana-go/internal/expansion"
	"github.com/animalet/sargantana-go/pkg/config/secrets"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// MockSecretProvider for testing expansion
type MockSecretProvider struct{}

func (m *MockSecretProvider) Resolve(key string) (string, error) {
	if key == "test-secret" {
		return "resolved-secret", nil
	}
	return "", nil
}

func (m *MockSecretProvider) Name() string {
	return "mock"
}

var _ = Describe("Expansion", func() {
	BeforeEach(func() {
		secrets.Register("mock", &MockSecretProvider{})
	})

	Context("ExpandVariables", func() {
		It("should expand string variables", func() {
			type TestStruct struct {
				Value string
			}
			s := TestStruct{Value: "${mock:test-secret}"}
			err := expansion.ExpandVariables(&s)
			Expect(err).NotTo(HaveOccurred())
			Expect(s.Value).To(Equal("resolved-secret"))
		})

		It("should expand nested structs", func() {
			type Nested struct {
				Value string
			}
			type TestStruct struct {
				Nested Nested
			}
			s := TestStruct{Nested: Nested{Value: "${mock:test-secret}"}}
			err := expansion.ExpandVariables(&s)
			Expect(err).NotTo(HaveOccurred())
			Expect(s.Nested.Value).To(Equal("resolved-secret"))
		})

		It("should expand pointers", func() {
			type TestStruct struct {
				Value *string
			}
			val := "${mock:test-secret}"
			s := TestStruct{Value: &val}
			err := expansion.ExpandVariables(&s)
			Expect(err).NotTo(HaveOccurred())
			Expect(*s.Value).To(Equal("resolved-secret"))
		})

		It("should expand slices", func() {
			type TestStruct struct {
				Values []string
			}
			s := TestStruct{Values: []string{"${mock:test-secret}", "normal"}}
			err := expansion.ExpandVariables(&s)
			Expect(err).NotTo(HaveOccurred())
			Expect(s.Values[0]).To(Equal("resolved-secret"))
			Expect(s.Values[1]).To(Equal("normal"))
		})

		It("should expand maps", func() {
			type TestStruct struct {
				Values map[string]string
			}
			s := TestStruct{Values: map[string]string{"key": "${mock:test-secret}"}}
			err := expansion.ExpandVariables(&s)
			Expect(err).NotTo(HaveOccurred())
			Expect(s.Values["key"]).To(Equal("resolved-secret"))
		})

		It("should handle nil pointers gracefully", func() {
			type TestStruct struct {
				Value *string
			}
			s := TestStruct{Value: nil}
			err := expansion.ExpandVariables(&s)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should return error on resolution error", func() {
			type TestStruct struct {
				Value string
			}
			// Use an unregistered prefix to trigger an error
			s := TestStruct{Value: "${unregistered:some-key}"}
			err := expansion.ExpandVariables(&s)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("error resolving property"))
		})
	})
})
