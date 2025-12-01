//go:build unit

package main

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Session Store Configuration", func() {
	Describe("configureSessionStore", func() {
		It("should use cookie-based sessions by default", func() {
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

			opts := &options{configPath: configPath}
			srv, closeFunc, err := initServer(opts)
			Expect(err).NotTo(HaveOccurred())
			Expect(srv).NotTo(BeNil())
			Expect(closeFunc).NotTo(BeNil())

			Expect(closeFunc()).To(Succeed())
		})

		It("should fail with invalid redis config structure", func() {
			tmpDir := GinkgoT().TempDir()
			configPath := filepath.Join(tmpDir, "config-redis-invalid.yaml")

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
redis: "this should be an object"
`
			Expect(os.WriteFile(configPath, []byte(invalidConfig), 0644)).To(Succeed())

			opts := &options{configPath: configPath}
			_, _, err := initServer(opts)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to load or create Redis client"))
		})

		It("should fail with invalid mongodb config structure", func() {
			tmpDir := GinkgoT().TempDir()
			configPath := filepath.Join(tmpDir, "config-mongodb-invalid.yaml")

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
mongodb: "this should be an object"
`
			Expect(os.WriteFile(configPath, []byte(invalidConfig), 0644)).To(Succeed())

			opts := &options{configPath: configPath}
			_, _, err := initServer(opts)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to load or create MongoDB client"))
		})

		It("should fail with invalid postgres config structure", func() {
			tmpDir := GinkgoT().TempDir()
			configPath := filepath.Join(tmpDir, "config-postgres-invalid.yaml")

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
postgres: "this should be an object"
`
			Expect(os.WriteFile(configPath, []byte(invalidConfig), 0644)).To(Succeed())

			opts := &options{configPath: configPath}
			_, _, err := initServer(opts)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to load or create PostgreSQL client"))
		})

		It("should fail with invalid memcached config structure", func() {
			tmpDir := GinkgoT().TempDir()
			configPath := filepath.Join(tmpDir, "config-memcached-invalid.yaml")

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
memcached: "this should be an object"
`
			Expect(os.WriteFile(configPath, []byte(invalidConfig), 0644)).To(Succeed())

			opts := &options{configPath: configPath}
			_, _, err := initServer(opts)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to load or create Memcached client"))
		})

		It("should fail when Redis client creation fails", func() {
			tmpDir := GinkgoT().TempDir()
			configPath := filepath.Join(tmpDir, "config-redis-bad-address.yaml")

			// Invalid Redis address that will fail connection
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
redis:
  address: "invalid:address:99999"
  username: "test"
  password: "test"
  database: 0
`
			Expect(os.WriteFile(configPath, []byte(invalidConfig), 0644)).To(Succeed())

			opts := &options{configPath: configPath}
			_, _, err := initServer(opts)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to create Redis"))
		})

		It("should fail when MongoDB client creation fails", func() {
			tmpDir := GinkgoT().TempDir()
			configPath := filepath.Join(tmpDir, "config-mongodb-bad-uri.yaml")

			// Invalid MongoDB URI
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
mongodb:
  uri: "mongodb://invalid\\host:99999"
  database: "testdb"
`
			Expect(os.WriteFile(configPath, []byte(invalidConfig), 0644)).To(Succeed())

			opts := &options{configPath: configPath}
			_, _, err := initServer(opts)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to load or create MongoDB client"))
		})

		It("should fail when PostgreSQL client creation fails", func() {
			tmpDir := GinkgoT().TempDir()
			configPath := filepath.Join(tmpDir, "config-postgres-bad-dsn.yaml")

			// Invalid PostgreSQL configuration - port out of range
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
postgres:
  host: "nonexistent.host.invalid"
  port: 99999
  username: "test"
  password: "test"
  database: "testdb"
  sslmode: "disable"
`
			Expect(os.WriteFile(configPath, []byte(invalidConfig), 0644)).To(Succeed())

			opts := &options{configPath: configPath}
			_, _, err := initServer(opts)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to load or create PostgreSQL client"))
		})

		It("should succeed with valid Memcached config", func() {
			tmpDir := GinkgoT().TempDir()
			configPath := filepath.Join(tmpDir, "config-memcached-valid.yaml")

			// Valid Memcached configuration
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
memcached:
  servers:
    - "localhost:11211"
`
			Expect(os.WriteFile(configPath, []byte(validConfig), 0644)).To(Succeed())

			opts := &options{configPath: configPath}
			srv, closeFunc, err := initServer(opts)
			Expect(err).NotTo(HaveOccurred())
			Expect(srv).NotTo(BeNil())
			Expect(closeFunc).NotTo(BeNil())
			Expect(closeFunc()).To(Succeed())
		})
	})
})
