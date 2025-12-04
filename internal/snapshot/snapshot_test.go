package snapshot_test

import (
	"testing"

	"github.com/animalet/sargantana-go/internal/snapshot"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestDeepcopy(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Deepcopy Suite")
}

type SimpleStruct struct {
	Name  string
	Value int
}

type NestedStruct struct {
	Field   string
	Nested  *SimpleStruct
	Slice   []string
	Map     map[string]int
	Numbers []int
}

var _ = Describe("DeepCopy", func() {
	Context("Nil handling", func() {
		It("should return nil when copying nil pointer", func() {
			var nilPtr *SimpleStruct
			result, err := snapshot.Copy(nilPtr)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(BeNil())
		})
	})

	Context("Simple structs", func() {
		It("should copy primitive fields correctly", func() {
			original := &SimpleStruct{
				Name:  "test",
				Value: 42,
			}

			copied, err := snapshot.Copy(original)
			Expect(err).NotTo(HaveOccurred())
			Expect(copied).NotTo(BeNil())

			// Values should be equal
			Expect(copied.Name).To(Equal(original.Name))
			Expect(copied.Value).To(Equal(original.Value))

			// Should be different pointers
			Expect(copied).NotTo(BeIdenticalTo(original))

			// Modifying copy shouldn't affect original
			copied.Name = "modified"
			Expect(original.Name).To(Equal("test"))
		})
	})

	Context("Nested pointers", func() {
		It("should deep copy nested pointer fields", func() {
			original := &NestedStruct{
				Field: "outer",
				Nested: &SimpleStruct{
					Name:  "inner",
					Value: 100,
				},
			}

			copied, err := snapshot.Copy(original)
			Expect(err).NotTo(HaveOccurred())
			Expect(copied).NotTo(BeNil())
			Expect(copied.Nested).NotTo(BeNil())

			// Values should be equal
			Expect(copied.Nested.Name).To(Equal(original.Nested.Name))
			Expect(copied.Nested.Value).To(Equal(original.Nested.Value))

			// Should be different pointers
			Expect(copied.Nested).NotTo(BeIdenticalTo(original.Nested))

			// Modifying copy's nested field shouldn't affect original
			copied.Nested.Name = "modified"
			Expect(original.Nested.Name).To(Equal("inner"))
		})

		It("should handle nil nested pointers", func() {
			original := &NestedStruct{
				Field:  "test",
				Nested: nil,
			}

			copied, err := snapshot.Copy(original)
			Expect(err).NotTo(HaveOccurred())
			Expect(copied).NotTo(BeNil())
			Expect(copied.Nested).To(BeNil())
		})
	})

	Context("Slices", func() {
		It("should deep copy string slices", func() {
			original := &NestedStruct{
				Slice: []string{"one", "two", "three"},
			}

			copied, err := snapshot.Copy(original)
			Expect(err).NotTo(HaveOccurred())
			Expect(copied).NotTo(BeNil())

			// Values should be equal
			Expect(copied.Slice).To(Equal(original.Slice))

			// Modifying copy shouldn't affect original
			copied.Slice[0] = "modified"
			Expect(original.Slice[0]).To(Equal("one"))
		})

		It("should deep copy int slices", func() {
			original := &NestedStruct{
				Numbers: []int{1, 2, 3, 4, 5},
			}

			copied, err := snapshot.Copy(original)
			Expect(err).NotTo(HaveOccurred())
			Expect(copied).NotTo(BeNil())

			// Values should be equal
			Expect(copied.Numbers).To(Equal(original.Numbers))

			// Modifying copy shouldn't affect original
			copied.Numbers[0] = 999
			Expect(original.Numbers[0]).To(Equal(1))
		})

		It("should handle appending to copied slice", func() {
			original := &NestedStruct{
				Slice: []string{"one", "two"},
			}

			copied, err := snapshot.Copy(original)
			Expect(err).NotTo(HaveOccurred())

			originalLen := len(original.Slice)

			// Append to copy
			copied.Slice = append(copied.Slice, "three")

			// Original should be unchanged
			Expect(len(original.Slice)).To(Equal(originalLen))
			Expect(original.Slice).NotTo(ContainElement("three"))
		})

		It("should handle nil slices", func() {
			original := &NestedStruct{
				Slice: nil,
			}

			copied, err := snapshot.Copy(original)
			Expect(err).NotTo(HaveOccurred())
			// copier initializes nil slices to empty slices
			Expect(copied.Slice).To(HaveLen(0))
		})

		It("should handle empty slices", func() {
			original := &NestedStruct{
				Slice: []string{},
			}

			copied, err := snapshot.Copy(original)
			Expect(err).NotTo(HaveOccurred())
			Expect(copied.Slice).NotTo(BeNil())
			Expect(len(copied.Slice)).To(Equal(0))
		})
	})

	Context("Maps", func() {
		It("should deep copy maps", func() {
			original := &NestedStruct{
				Map: map[string]int{
					"one":   1,
					"two":   2,
					"three": 3,
				},
			}

			copied, err := snapshot.Copy(original)
			Expect(err).NotTo(HaveOccurred())
			Expect(copied).NotTo(BeNil())

			// Values should be equal
			Expect(copied.Map).To(Equal(original.Map))

			// Modifying copy shouldn't affect original
			copied.Map["one"] = 999
			Expect(original.Map["one"]).To(Equal(1))
		})

		It("should handle adding keys to copied map", func() {
			original := &NestedStruct{
				Map: map[string]int{
					"one": 1,
				},
			}

			copied, err := snapshot.Copy(original)
			Expect(err).NotTo(HaveOccurred())

			// Add key to copy
			copied.Map["two"] = 2

			// Original should not have new key
			Expect(original.Map).NotTo(HaveKey("two"))
		})

		It("should handle nil maps", func() {
			original := &NestedStruct{
				Map: nil,
			}

			copied, err := snapshot.Copy(original)
			Expect(err).NotTo(HaveOccurred())
			// copier initializes nil maps to empty maps
			Expect(copied.Map).To(HaveLen(0))
		})

		It("should handle empty maps", func() {
			original := &NestedStruct{
				Map: map[string]int{},
			}

			copied, err := snapshot.Copy(original)
			Expect(err).NotTo(HaveOccurred())
			Expect(copied.Map).NotTo(BeNil())
			Expect(len(copied.Map)).To(Equal(0))
		})
	})

	Context("Complex nested structures", func() {
		It("should deep copy all levels", func() {
			original := &NestedStruct{
				Field: "top",
				Nested: &SimpleStruct{
					Name:  "nested",
					Value: 42,
				},
				Slice:   []string{"a", "b", "c"},
				Map:     map[string]int{"x": 1, "y": 2},
				Numbers: []int{10, 20, 30},
			}

			copied, err := snapshot.Copy(original)
			Expect(err).NotTo(HaveOccurred())
			Expect(copied).NotTo(BeNil())

			// All values should be equal
			Expect(copied.Field).To(Equal(original.Field))
			Expect(copied.Nested.Name).To(Equal(original.Nested.Name))
			Expect(copied.Slice).To(Equal(original.Slice))
			Expect(copied.Map).To(Equal(original.Map))
			Expect(copied.Numbers).To(Equal(original.Numbers))

			// All should be separate memory
			copied.Field = "mod1"
			copied.Nested.Name = "mod2"
			copied.Slice[0] = "mod3"
			copied.Map["x"] = 999
			copied.Numbers[0] = 888

			// Originals unchanged
			Expect(original.Field).To(Equal("top"))
			Expect(original.Nested.Name).To(Equal("nested"))
			Expect(original.Slice[0]).To(Equal("a"))
			Expect(original.Map["x"]).To(Equal(1))
			Expect(original.Numbers[0]).To(Equal(10))
		})
	})

	Context("MustCopy", func() {
		It("should create immutable copy without returning error", func() {
			original := &SimpleStruct{
				Name:  "test",
				Value: 42,
			}

			copied := snapshot.MustCopy(original)

			Expect(copied).NotTo(BeNil())
			Expect(copied.Name).To(Equal("test"))
			Expect(copied.Value).To(Equal(42))

			// Modify original
			original.Name = "modified"
			original.Value = 99

			// Copy should be unchanged
			Expect(copied.Name).To(Equal("test"))
			Expect(copied.Value).To(Equal(42))
		})

		It("should handle nil input gracefully", func() {
			var original *SimpleStruct
			copied := snapshot.MustCopy(original)
			Expect(copied).To(BeNil())
		})

		It("should work with nested structures", func() {
			original := &NestedStruct{
				Field: "parent",
				Nested: &SimpleStruct{
					Name:  "child",
					Value: 100,
				},
				Slice: []string{"a", "b"},
				Map:   map[string]int{"x": 1},
			}

			copied := snapshot.MustCopy(original)

			// Modify all levels of original
			original.Field = "mod1"
			original.Nested.Name = "mod2"
			original.Slice[0] = "mod3"
			original.Map["x"] = 999

			// Copy should be completely unchanged
			Expect(copied.Field).To(Equal("parent"))
			Expect(copied.Nested.Name).To(Equal("child"))
			Expect(copied.Slice[0]).To(Equal("a"))
			Expect(copied.Map["x"]).To(Equal(1))
		})

		It("should be usable in constructor pattern", func() {
			// Example of typical constructor usage
			type Config struct {
				Address string
				Ports   []int
			}

			type Server struct {
				config Config
			}

			NewServer := func(cfg Config) *Server {
				return &Server{
					config: *snapshot.MustCopy(&cfg),
				}
			}

			cfg := Config{
				Address: "localhost:8080",
				Ports:   []int{8080, 8081},
			}

			server := NewServer(cfg)

			// Modify original config
			cfg.Address = "hacked:9999"
			cfg.Ports[0] = 6666

			// Server's config should be unchanged
			Expect(server.config.Address).To(Equal("localhost:8080"))
			Expect(server.config.Ports[0]).To(Equal(8080))
		})
	})
})
