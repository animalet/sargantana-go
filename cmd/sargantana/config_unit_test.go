//go:build unit

package main

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Configuration Loading", func() {
	Describe("loadConfig", func() {
		It("should load valid config file", func() {
			tmpDir := GinkgoT().TempDir()
			configPath := filepath.Join(tmpDir, "config.yaml")

			validConfig := `sargantana:
  server:
    address: :9999
    session_name: test_session
    session_secret: a_very_long_secret_key_for_testing_purposes
  controllers:
    - type: static
      config:
        status: 200
        body: "OK"
`
			Expect(os.WriteFile(configPath, []byte(validConfig), 0644)).To(Succeed())

			cfg, err := loadConfig(configPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(cfg).NotTo(BeNil())
		})

		It("should fail with missing file", func() {
			cfg, err := loadConfig("/nonexistent/path/config.yaml")
			Expect(err).To(HaveOccurred())
			Expect(cfg).To(BeNil())
			Expect(err.Error()).To(ContainSubstring("failed to load configuration file"))
		})

		It("should fail with invalid YAML", func() {
			tmpDir := GinkgoT().TempDir()
			configPath := filepath.Join(tmpDir, "invalid.yaml")
			Expect(os.WriteFile(configPath, []byte("invalid: yaml: [[["), 0644)).To(Succeed())

			cfg, err := loadConfig(configPath)
			Expect(err).To(HaveOccurred())
			Expect(cfg).To(BeNil())
		})

		It("should fail with invalid Vault config", func() {
			tmpDir := GinkgoT().TempDir()
			configPath := filepath.Join(tmpDir, "config-vault-invalid.yaml")

			invalidConfig := `sargantana:
  server:
    address: :9999
    session_name: test_session
    session_secret: a_very_long_secret_key_for_testing_purposes
  controllers:
    - type: static
      config:
        status: 200
        body: "OK"
vault: "this should be an object"
`
			Expect(os.WriteFile(configPath, []byte(invalidConfig), 0644)).To(Succeed())

			cfg, err := loadConfig(configPath)
			Expect(err).To(HaveOccurred())
			Expect(cfg).To(BeNil())
			Expect(err.Error()).To(ContainSubstring("failed to load Vault configuration"))
		})

		It("should fail with invalid file resolver config", func() {
			tmpDir := GinkgoT().TempDir()
			configPath := filepath.Join(tmpDir, "config-file-invalid.yaml")

			invalidConfig := `sargantana:
  server:
    address: :9999
    session_name: test_session
    session_secret: a_very_long_secret_key_for_testing_purposes
  controllers:
    - type: static
      config:
        status: 200
        body: "OK"
file_resolver: "this should be an object"
`
			Expect(os.WriteFile(configPath, []byte(invalidConfig), 0644)).To(Succeed())

			cfg, err := loadConfig(configPath)
			Expect(err).To(HaveOccurred())
			Expect(cfg).To(BeNil())
			Expect(err.Error()).To(ContainSubstring("failed to load file secret resolver configuration"))
		})

		It("should fail with invalid AWS config", func() {
			tmpDir := GinkgoT().TempDir()
			configPath := filepath.Join(tmpDir, "config-aws-invalid.yaml")

			invalidConfig := `sargantana:
  server:
    address: :9999
    session_name: test_session
    session_secret: a_very_long_secret_key_for_testing_purposes
  controllers:
    - type: static
      config:
        status: 200
        body: "OK"
aws: "this should be an object"
`
			Expect(os.WriteFile(configPath, []byte(invalidConfig), 0644)).To(Succeed())

			cfg, err := loadConfig(configPath)
			Expect(err).To(HaveOccurred())
			Expect(cfg).To(BeNil())
			Expect(err.Error()).To(ContainSubstring("failed to load AWS Secrets Manager configuration"))
		})
	})
})
