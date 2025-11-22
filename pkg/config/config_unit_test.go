//go:build unit

package config_test

import (
	"encoding/json"
	"encoding/xml"
	"os"
	"path/filepath"

	"github.com/animalet/sargantana-go/pkg/config"
	"github.com/animalet/sargantana-go/pkg/secrets"
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
		It("should return error if file does not exist", func() {
			_, err := config.ReadModular("non_existent_file.yaml")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to read configuration file"))
		})

		It("should return error if unmarshal fails", func() {
			path := filepath.Join(tempDir, "invalid.yaml")
			err := os.WriteFile(path, []byte("invalid: yaml: content: :"), 0644)
			Expect(err).NotTo(HaveOccurred())

			_, err = config.ReadModular(path)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("error marshalling to"))
		})
	})

	Context("ReadFull", func() {
		It("should return error if file does not exist", func() {
			_, err := config.ReadFull[TestConfigStruct]("non_existent_file.yaml")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to read configuration file"))
		})

		It("should return error if unmarshal fails", func() {
			path := filepath.Join(tempDir, "invalid.yaml")
			err := os.WriteFile(path, []byte("invalid: yaml: content: :"), 0644)
			Expect(err).NotTo(HaveOccurred())

			_, err = config.ReadFull[TestConfigStruct](path)
			Expect(err).To(HaveOccurred())
		})

		It("should return error if validation fails", func() {
			path := filepath.Join(tempDir, "invalid_val.yaml")
			err := os.WriteFile(path, []byte("field: invalid"), 0644)
			Expect(err).NotTo(HaveOccurred())

			_, err = config.ReadFull[TestConfigStruct](path)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("configuration is invalid"))
		})
	})

	Context("Load", func() {
		It("should return error if unmarshal fails", func() {
			raw := config.ModuleRawConfig([]byte("invalid: yaml: :"))
			_, err := config.Load[TestConfigStruct](raw)
			Expect(err).To(HaveOccurred())
		})

		It("should return error if validation fails", func() {
			raw := config.ModuleRawConfig([]byte("field: invalid"))
			_, err := config.Load[TestConfigStruct](raw)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("configuration is invalid"))
		})
	})

	Context("Unmarshal", func() {
		It("should return error if unmarshal fails", func() {
			raw := config.ModuleRawConfig([]byte("invalid: yaml: :"))
			_, err := config.Unmarshal[TestConfigStruct](raw)
			Expect(err).To(HaveOccurred())
		})

		It("should return error if validation fails", func() {
			raw := config.ModuleRawConfig([]byte("field: invalid"))
			_, err := config.Unmarshal[TestConfigStruct](raw)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("configuration is invalid"))
		})
	})

	Context("Formats", func() {
		It("should support JSON format", func() {
			config.UseFormat(config.JsonFormat)
			defer config.UseFormat(config.YamlFormat)

			raw := config.ModuleRawConfig([]byte(`{"field": "value"}`))
			cfg, err := config.Unmarshal[TestConfigStruct](raw)
			Expect(err).NotTo(HaveOccurred())
			Expect(cfg.Field).To(Equal("value"))
		})

		It("should support TOML format", func() {
			config.UseFormat(config.TomlFormat)
			defer config.UseFormat(config.YamlFormat)

			raw := config.ModuleRawConfig([]byte(`field = "value"`))
			cfg, err := config.Unmarshal[TestConfigStruct](raw)
			Expect(err).NotTo(HaveOccurred())
			Expect(cfg.Field).To(Equal("value"))
		})

		It("should support XML format", func() {
			config.UseFormat(config.XmlFormat)
			defer config.UseFormat(config.YamlFormat)

			raw := config.ModuleRawConfig([]byte(`<TestConfigStruct><field>value</field></TestConfigStruct>`))
			cfg, err := config.Unmarshal[TestConfigStruct](raw)
			Expect(err).NotTo(HaveOccurred())
			Expect(cfg.Field).To(Equal("value"))
		})

		It("should return error for unsupported format", func() {
			config.UseFormat("invalid")
			defer config.UseFormat(config.YamlFormat)

			raw := config.ModuleRawConfig([]byte(`field: value`))
			_, err := config.Unmarshal[TestConfigStruct](raw)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unsupported format"))
		})
	})

	Context("ModuleRawConfig", func() {
		It("should unmarshal YAML into ModuleRawConfig", func() {
			var m config.ModuleRawConfig
			err := yaml.Unmarshal([]byte("key: value"), &m)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(m)).To(ContainSubstring("key: value"))
		})

		It("should unmarshal JSON into ModuleRawConfig", func() {
			var m config.ModuleRawConfig
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

			var m config.ModuleRawConfig
			err := m.UnmarshalTOML(data)
			Expect(err).To(HaveOccurred())
			// toml.Marshal returns error for unsupported types
		})

		It("should return error when XML decode fails", func() {
			var m config.ModuleRawConfig
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
				cfg, err := config.Unmarshal[ConfigTestStruct](data)
				Expect(err).NotTo(HaveOccurred())
				Expect(cfg.Host).To(Equal("localhost"))
				Expect(cfg.Port).To(Equal(8080))
				Expect(cfg.Secret).To(Equal("super-secret-value"))
			})

			It("should fail validation", func() {
				data := []byte(`
host: localhost
`)
				_, err := config.Unmarshal[ConfigTestStruct](data)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("port is required"))
			})
		})

		Describe("ReadModular", func() {
			It("should read modular config", func() {
				path := filepath.Join(tempDir, "config.yaml")
				err := os.WriteFile(path, []byte(`
server:
  host: localhost
  port: 8080
`), 0644)
				Expect(err).NotTo(HaveOccurred())

				cfg, err := config.ReadModular(path)
				Expect(err).NotTo(HaveOccurred())
				Expect(cfg).To(HaveKey("server"))
			})
		})

		Describe("Load", func() {
			It("should load partial config", func() {
				data := []byte(`
host: localhost
port: 8080
secret: ${mock:my-secret}
`)
				cfg, err := config.Load[ConfigTestStruct](data)
				Expect(err).NotTo(HaveOccurred())
				Expect(cfg.Host).To(Equal("localhost"))
				Expect(cfg.Port).To(Equal(8080))
				Expect(cfg.Secret).To(Equal("super-secret-value"))
			})
		})
	})
})
