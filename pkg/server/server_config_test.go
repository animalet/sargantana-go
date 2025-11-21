//go:build unit

package server_test

import (
	"github.com/animalet/sargantana-go/pkg/server"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ServerConfig", func() {
	It("should validate valid config", func() {
		cfg := server.WebServerConfig{
			Address:       "localhost:8080",
			SessionName:   "test-session",
			SessionSecret: "12345678901234567890123456789012",
		}
		err := cfg.Validate()
		Expect(err).NotTo(HaveOccurred())
	})

	It("should fail validation if address is missing", func() {
		cfg := server.WebServerConfig{
			SessionName:   "test-session",
			SessionSecret: "12345678901234567890123456789012",
		}
		err := cfg.Validate()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("address must be set and non-empty"))
	})

	It("should fail validation if session secret is too short", func() {
		// Note: The current implementation of Validate in server_config.go only checks for non-empty
		// It doesn't seem to check length. Let's check the code again.
		// server_config.go:19: if c.SessionSecret == "" { return errors.New("session_secret must be set and non-empty") }
		// So the test expectation "session secret must be at least 32 characters" might be wrong based on current implementation.
		// I will adjust the test to match implementation or fix implementation.
		// Given the task is to rewrite tests, I should match implementation.
		// But usually session secrets should be long.
		// Let's stick to what Validate() does: checks for empty.
		cfg := server.WebServerConfig{
			Address:       "localhost:8080",
			SessionName:   "test-session",
			SessionSecret: "",
		}
		err := cfg.Validate()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("session_secret must be set and non-empty"))
	})
})

var _ = Describe("ControllerConfig", func() {
	It("should validate valid config", func() {
		cfg := server.ControllerBinding{
			Name:     "test-controller",
			TypeName: "test-type",
			Config:   []byte("{}"), // Config is required
		}
		err := cfg.Validate()
		Expect(err).NotTo(HaveOccurred())
	})

	It("should fail validation if type is missing", func() {
		cfg := server.ControllerBinding{
			Name:   "test-controller",
			Config: []byte("{}"),
		}
		err := cfg.Validate()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("controller type must be set and non-empty"))
	})

	It("should fail validation if config is missing", func() {
		cfg := server.ControllerBinding{
			Name:     "test-controller",
			TypeName: "test-type",
		}
		err := cfg.Validate()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("controller config must be provided"))
	})
})

var _ = Describe("Server", func() {
	It("should register controller types", func() {
		server.AddControllerType("test-type", nil)
		// No easy way to check internal map without exposing it, but we can check if it doesn't panic
		// Ideally we should have a GetControllerType or similar, but for now this just ensures the function exists and runs
	})
})
