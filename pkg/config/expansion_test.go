//go:build unit

package config

import (
	"reflect"

	"github.com/animalet/sargantana-go/pkg/secrets"
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

	Context("expandVariables", func() {
		It("should expand string variables", func() {
			type TestStruct struct {
				Value string
			}
			s := TestStruct{Value: "${mock:test-secret}"}
			expandVariables(reflect.ValueOf(&s).Elem())
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
			expandVariables(reflect.ValueOf(&s).Elem())
			Expect(s.Nested.Value).To(Equal("resolved-secret"))
		})

		It("should expand pointers", func() {
			type TestStruct struct {
				Value *string
			}
			val := "${mock:test-secret}"
			s := TestStruct{Value: &val}
			expandVariables(reflect.ValueOf(&s).Elem())
			Expect(*s.Value).To(Equal("resolved-secret"))
		})

		It("should expand slices", func() {
			type TestStruct struct {
				Values []string
			}
			s := TestStruct{Values: []string{"${mock:test-secret}", "normal"}}
			expandVariables(reflect.ValueOf(&s).Elem())
			Expect(s.Values[0]).To(Equal("resolved-secret"))
			Expect(s.Values[1]).To(Equal("normal"))
		})

		It("should expand maps", func() {
			type TestStruct struct {
				Values map[string]string
			}
			s := TestStruct{Values: map[string]string{"key": "${mock:test-secret}"}}
			expandVariables(reflect.ValueOf(&s).Elem())
			Expect(s.Values["key"]).To(Equal("resolved-secret"))
		})

		It("should handle nil pointers gracefully", func() {
			type TestStruct struct {
				Value *string
			}
			s := TestStruct{Value: nil}
			Expect(func() {
				expandVariables(reflect.ValueOf(&s).Elem())
			}).NotTo(Panic())
		})
	})
})
