//go:build unit

package config_test

import (
	"os"
	"path/filepath"

	"github.com/animalet/sargantana-go/pkg/config"
	"github.com/animalet/sargantana-go/pkg/secrets"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
)

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
