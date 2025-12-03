package deepcopy_test

import (
	"testing"

	"github.com/animalet/sargantana-go/internal/deepcopy"
)

type BenchmarkSimpleStruct struct {
	Name    string
	Value   int
	Enabled bool
}

type BenchmarkNestedStruct struct {
	Field   string
	Nested  *BenchmarkSimpleStruct
	Servers []string
	Config  map[string]int
	Numbers []int
}

func BenchmarkCopy_SimpleStruct(b *testing.B) {
	original := &BenchmarkSimpleStruct{
		Name:    "test-server",
		Value:   42,
		Enabled: true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = deepcopy.Copy(original)
	}
}

func BenchmarkCopy_NestedStruct(b *testing.B) {
	original := &BenchmarkNestedStruct{
		Field: "main",
		Nested: &BenchmarkSimpleStruct{
			Name:    "nested-server",
			Value:   100,
			Enabled: true,
		},
		Servers: []string{"server1:11211", "server2:11211", "server3:11211"},
		Config:  map[string]int{"timeout": 100, "retries": 3, "max_conns": 10},
		Numbers: []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = deepcopy.Copy(original)
	}
}

func BenchmarkCopy_LargeSlice(b *testing.B) {
	servers := make([]string, 100)
	for i := 0; i < 100; i++ {
		servers[i] = "server:11211"
	}

	original := &BenchmarkNestedStruct{
		Field:   "large",
		Servers: servers,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = deepcopy.Copy(original)
	}
}

func BenchmarkCopy_LargeMap(b *testing.B) {
	config := make(map[string]int, 100)
	for i := 0; i < 100; i++ {
		config["key"] = i
	}

	original := &BenchmarkNestedStruct{
		Field:  "large-map",
		Config: config,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = deepcopy.Copy(original)
	}
}

func BenchmarkCopy_Nil(b *testing.B) {
	var original *BenchmarkSimpleStruct

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = deepcopy.Copy(original)
	}
}
