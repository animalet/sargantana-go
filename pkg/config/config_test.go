package config

import (
	"reflect"

	"github.com/onsi/ginkgo/v2"
)

type exampleConfig struct {
	key string `yaml:"key"`
}

func (e exampleConfig) Validate() error {
	return nil
}

var _ = ginkgo.Describe("Config", func() {
	cfg := &Config{
		Config: []byte("example_key:\n  key: value"),
	}
	ginkgo.It("should have tests implemented", func() {
		partial, err := Load[exampleConfig](cfg)
		if err != nil {
			ginkgo.Fail("Load failed: " + err.Error())
		}
		expected := &exampleConfig{key: "value"}
		if !reflect.DeepEqual(partial, expected) {
			ginkgo.Fail("Loaded config does not match expected value")
		}
	})
})
