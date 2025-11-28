//go:build unit

package config

import (
	"encoding/json"
	"encoding/xml"
	"os"
	"path/filepath"

	"github.com/animalet/sargantana-go/pkg/config/secrets"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

type TestConfigStruct struct {
	Field string `yaml:"field" json:"field" toml:"field" xml:"field"`
}

func (t TestConfigStruct) Validate() error {
	if t.Field == "invalid" {
		return os.ErrInvalid
	}
	return nil
}

// brokenReader simulates an error during reading
type brokenReader struct{}

func (b *brokenReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("broken reader")
}

type ConfigTestStruct struct {
	Host   string `yaml:"host"`
	Port   int    `yaml:"port"`
	Secret string `yaml:"secret"`
}

func (c ConfigTestStruct) Validate() error {
	if c.Port == 0 {
		return errors.New("port is required")
	}
	return nil
}

type MockSecretLoader struct {
	Secrets map[string]string
}

func (m *MockSecretLoader) Resolve(key string) (string, error) {
	if val, ok := m.Secrets[key]; ok {
		return val, nil
	}
	return "", errors.New("secret not found")
}

func (m *MockSecretLoader) Name() string {
	return "mock"
}

var _ = Describe("Config", func() {
	var tempDir string

	BeforeEach(func() {
		var err error
		tempDir, err = os.MkdirTemp("", "config_test")
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		os.RemoveAll(tempDir)
	})

	Context("ReadModular", func() {
		It("should read modular config and allow getting modules", func() {
			path := filepath.Join(tempDir, "test.yaml")
			err := os.WriteFile(path, []byte(`
test:
  field: value
`), 0644)
			Expect(err).NotTo(HaveOccurred())

			cfg, err := NewConfig(path)
			Expect(err).NotTo(HaveOccurred())
			Expect(cfg).NotTo(BeNil())

			// Verify we can get a module
			testCfg, err := Get[TestConfigStruct](cfg, "test")
			Expect(err).NotTo(HaveOccurred())
			Expect(testCfg).NotTo(BeNil())
			Expect(testCfg.Field).To(Equal("value"))
		})

		It("should return error if file does not exist", func() {
			_, err := NewConfig("non_existent_file.yaml")
			Expect(err).To(HaveOccurred())
		})

		It("should return error if file content is invalid", func() {
			path := filepath.Join(tempDir, "invalid.yaml")
			err := os.WriteFile(path, []byte("invalid yaml content: :"), 0644)
			Expect(err).NotTo(HaveOccurred())

			_, err = NewConfig(path)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("Get", func() {
		It("should return nil if module does not exist", func() {
			path := filepath.Join(tempDir, "yaml")
			err := os.WriteFile(path, []byte(`
server:
  host: localhost
  port: 8080
`), 0644)
			Expect(err).NotTo(HaveOccurred())

			cfg, err := NewConfig(path)
			Expect(err).NotTo(HaveOccurred())

			val, err := Get[TestConfigStruct](cfg, "nonexistent")
			Expect(err).NotTo(HaveOccurred())
			Expect(val).To(BeNil())
		})

		It("should return error if validation fails", func() {
			path := filepath.Join(tempDir, "yaml")
			err := os.WriteFile(path, []byte(`
test:
  field: invalid
`), 0644)
			Expect(err).NotTo(HaveOccurred())

			cfg, err := NewConfig(path)
			Expect(err).NotTo(HaveOccurred())

			_, err = Get[TestConfigStruct](cfg, "test")
			Expect(err).To(HaveOccurred())
		})
	})

	Context("Unmarshal", func() {
		It("should return error if unmarshal fails", func() {
			raw := ModuleRawConfig([]byte("invalid: yaml: :"))
			_, err := Unmarshal[TestConfigStruct](raw)
			Expect(err).To(HaveOccurred())
		})

		It("should return error if validation fails", func() {
			raw := ModuleRawConfig([]byte("field: invalid"))
			_, err := Unmarshal[TestConfigStruct](raw)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("configuration is invalid"))
		})
	})

	Context("Formats", func() {
		It("should support JSON format", func() {
			UseFormat(JsonFormat)
			defer UseFormat(YamlFormat)

			raw := ModuleRawConfig([]byte(`{"field": "value"}`))
			cfg, err := Unmarshal[TestConfigStruct](raw)
			Expect(err).NotTo(HaveOccurred())
			Expect(cfg.Field).To(Equal("value"))
		})

		It("should support TOML format", func() {
			UseFormat(TomlFormat)
			defer UseFormat(YamlFormat)

			raw := ModuleRawConfig([]byte(`field = "value"`))
			cfg, err := Unmarshal[TestConfigStruct](raw)
			Expect(err).NotTo(HaveOccurred())
			Expect(cfg.Field).To(Equal("value"))
		})

		It("should support XML format", func() {
			UseFormat(XmlFormat)
			defer UseFormat(YamlFormat)

			raw := ModuleRawConfig([]byte(`<TestConfigStruct><field>value</field></TestConfigStruct>`))
			cfg, err := Unmarshal[TestConfigStruct](raw)
			Expect(err).NotTo(HaveOccurred())
			Expect(cfg.Field).To(Equal("value"))
		})

		It("should return error for unsupported format", func() {
			UseFormat("invalid")
			defer UseFormat(YamlFormat)

			raw := ModuleRawConfig([]byte(`field: value`))
			_, err := Unmarshal[TestConfigStruct](raw)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unsupported format"))
		})
	})

	Context("ModuleRawConfig", func() {
		It("should unmarshal YAML into ModuleRawConfig", func() {
			var m ModuleRawConfig
			err := yaml.Unmarshal([]byte("key: value"), &m)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(m)).To(ContainSubstring("key: value"))
		})

		It("should unmarshal JSON into ModuleRawConfig", func() {
			var m ModuleRawConfig
			err := json.Unmarshal([]byte(`{"key": "value"}`), &m)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(m)).To(Equal(`{"key": "value"}`))
		})

	})

	Context("ModuleRawConfig Unmarshal Errors", func() {
		It("should return error when TOML marshal fails", func() {
			// Create a structure that cannot be marshaled to TOML (e.g., channel)
			type InvalidTOML struct {
				Ch chan int
			}
			data := InvalidTOML{Ch: make(chan int)}

			var m ModuleRawConfig
			err := m.UnmarshalTOML(data)
			Expect(err).To(HaveOccurred())
			// toml.Marshal returns error for unsupported types
		})

		It("should return error when XML decode fails", func() {
			var m ModuleRawConfig
			// Create a decoder with a broken reader
			decoder := xml.NewDecoder(&brokenReader{})
			start := xml.StartElement{Name: xml.Name{Local: "root"}}

			err := m.UnmarshalXML(decoder, start)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("Config Integration (Mocked)", func() {
		var (
			mockLoader *MockSecretLoader
			tempDir    string
		)

		BeforeEach(func() {
			mockLoader = &MockSecretLoader{
				Secrets: map[string]string{
					"my-secret": "super-secret-value",
				},
			}
			secrets.Register("mock", mockLoader)
			var err error
			tempDir, err = os.MkdirTemp("", "config-test")
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			os.RemoveAll(tempDir)
		})

		Describe("Unmarshal", func() {
			It("should unmarshal valid yaml", func() {
				data := []byte(`
host: localhost
port: 8080
secret: ${mock:my-secret}
`)
				cfg, err := Unmarshal[ConfigTestStruct](data)
				Expect(err).NotTo(HaveOccurred())
				Expect(cfg.Host).To(Equal("localhost"))
				Expect(cfg.Port).To(Equal(8080))
				Expect(cfg.Secret).To(Equal("super-secret-value"))
			})

			It("should fail validation", func() {
				data := []byte(`
host: localhost
`)
				_, err := Unmarshal[ConfigTestStruct](data)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("port is required"))
			})
		})

		Describe("ReadModular", func() {
			It("should read modular config", func() {
				path := filepath.Join(tempDir, "yaml")
				err := os.WriteFile(path, []byte(`
server:
  host: localhost
  port: 8080
`), 0644)
				Expect(err).NotTo(HaveOccurred())

				cfg, err := NewConfig(path)
				Expect(err).NotTo(HaveOccurred())

				// We can't use ConfigTestStruct here because the yaml structure doesn't match
				// ConfigTestStruct expects fields at root, but here they are under "server" key?
				// Wait, ReadModular reads the file into a map. "server" key in the map contains the content.
				// The content is:
				// host: localhost
				// port: 8080
				// This matches ConfigTestStruct.

				val, err := Get[ConfigTestStruct](cfg, "server")
				Expect(err).NotTo(HaveOccurred())
				Expect(val).NotTo(BeNil())
				Expect(val.Host).To(Equal("localhost"))
				Expect(val.Port).To(Equal(8080))
			})
		})
	})
})
