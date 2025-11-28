//go:build unit

package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rs/zerolog"
)

var _ = Describe("Command-Line Argument Parsing", func() {
	Describe("parseFlags", func() {
		It("should parse valid config flag", func() {
			opts, err := parseFlags([]string{"--config", "/path/to/config.yaml"})
			Expect(err).NotTo(HaveOccurred())
			Expect(opts.configPath).To(Equal("/path/to/config.yaml"))
			Expect(opts.debug).To(BeFalse())
			Expect(opts.showHelp).To(BeFalse())
			Expect(opts.showVersion).To(BeFalse())
		})

		It("should parse config and debug flags", func() {
			opts, err := parseFlags([]string{"--config", "/path/to/config.yaml", "--debug"})
			Expect(err).NotTo(HaveOccurred())
			Expect(opts.configPath).To(Equal("/path/to/config.yaml"))
			Expect(opts.debug).To(BeTrue())
		})

		It("should parse version flag", func() {
			opts, err := parseFlags([]string{"--version"})
			Expect(err).NotTo(HaveOccurred())
			Expect(opts.showVersion).To(BeTrue())
		})

		It("should parse help flag", func() {
			opts, err := parseFlags([]string{"--help"})
			Expect(err).NotTo(HaveOccurred())
			Expect(opts.showHelp).To(BeTrue())
		})

		It("should return error for invalid flag", func() {
			_, err := parseFlags([]string{"--invalid-flag"})
			Expect(err).To(HaveOccurred())
		})

		It("should handle no flags", func() {
			opts, err := parseFlags([]string{})
			Expect(err).NotTo(HaveOccurred())
			Expect(opts.configPath).To(BeEmpty())
		})

		It("should handle -h flag (standard help)", func() {
			opts, err := parseFlags([]string{"-h"})
			Expect(err).NotTo(HaveOccurred())
			Expect(opts.showHelp).To(BeTrue())
		})
	})
})

var _ = Describe("Usage Message", func() {
	It("should print usage to file", func() {
		tmpFile, err := os.CreateTemp("", "usage-*.txt")
		Expect(err).NotTo(HaveOccurred())
		defer func() {
			_ = os.Remove(tmpFile.Name())
		}()
		defer func() {
			_ = tmpFile.Close()
		}()

		printUsage(tmpFile)

		content, err := os.ReadFile(tmpFile.Name())
		Expect(err).NotTo(HaveOccurred())

		output := string(content)
		Expect(output).To(ContainSubstring("Usage: sargantana"))
		Expect(output).To(ContainSubstring("--config"))
		Expect(output).To(ContainSubstring("--debug"))
		Expect(output).To(ContainSubstring("--version"))
		Expect(output).To(ContainSubstring("--help"))
		Expect(output).To(ContainSubstring("github.com/animalet/sargantana-go"))
	})

	It("should handle different output writers", func() {
		tmpFile, err := os.CreateTemp("", "usage-stderr-*.txt")
		Expect(err).NotTo(HaveOccurred())
		defer func() {
			_ = os.Remove(tmpFile.Name())
		}()
		defer func() {
			_ = tmpFile.Close()
		}()

		printUsage(tmpFile)

		content, err := os.ReadFile(tmpFile.Name())
		Expect(err).NotTo(HaveOccurred())
		Expect(len(content)).To(BeNumerically(">", 0))
	})
})

var _ = Describe("Logging Setup", func() {
	It("should set debug level when debug mode is enabled", func() {
		setupLogging(true)
		Expect(zerolog.GlobalLevel()).To(Equal(zerolog.DebugLevel))
	})

	It("should set info level when debug mode is disabled", func() {
		setupLogging(false)
		Expect(zerolog.GlobalLevel()).To(Equal(zerolog.InfoLevel))
	})
})

var _ = Describe("runWithArgs", func() {
	It("should show help message", func() {
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		exitCode := runWithArgs([]string{"--help"})

		_ = w.Close()
		os.Stdout = oldStdout

		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)

		Expect(exitCode).To(Equal(exitSuccess))
		Expect(buf.String()).To(ContainSubstring("Usage: sargantana"))
	})

	It("should show version", func() {
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		exitCode := runWithArgs([]string{"--version"})

		_ = w.Close()
		os.Stdout = oldStdout

		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)

		Expect(exitCode).To(Equal(exitSuccess))
		Expect(buf.String()).To(ContainSubstring("version"))
	})

	It("should fail when config is not provided", func() {
		oldStderr := os.Stderr
		r, w, _ := os.Pipe()
		os.Stderr = w

		exitCode := runWithArgs([]string{})

		_ = w.Close()
		os.Stderr = oldStderr

		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)

		Expect(exitCode).To(Equal(exitError))
		Expect(buf.String()).To(ContainSubstring("--config flag is required"))
	})

	It("should fail with invalid config path", func() {
		exitCode := runWithArgs([]string{"--config", "/nonexistent/config.yaml"})
		Expect(exitCode).To(Equal(exitError))
	})

	It("should fail with invalid flag", func() {
		exitCode := runWithArgs([]string{"--invalid"})
		Expect(exitCode).To(Equal(exitError))
	})
})

var _ = Describe("initServer", func() {
	It("should fail with invalid config file", func() {
		tmpDir := GinkgoT().TempDir()
		configPath := filepath.Join(tmpDir, "invalid.yaml")
		Expect(os.WriteFile(configPath, []byte("invalid: yaml: content: [[["), 0644)).To(Succeed())

		opts := &options{configPath: configPath}
		_, _, err := initServer(opts)
		Expect(err).To(HaveOccurred())
	})

	It("should fail with missing config file", func() {
		opts := &options{configPath: "/nonexistent/path/config.yaml"}
		_, _, err := initServer(opts)
		Expect(err).To(HaveOccurred())
	})

	It("should fail with empty config", func() {
		tmpDir := GinkgoT().TempDir()
		configPath := filepath.Join(tmpDir, "empty.yaml")
		Expect(os.WriteFile(configPath, []byte(""), 0644)).To(Succeed())

		opts := &options{configPath: configPath}
		_, _, err := initServer(opts)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("server configuration is required"))
	})

	It("should fail with invalid server config structure", func() {
		tmpDir := GinkgoT().TempDir()
		configPath := filepath.Join(tmpDir, "invalid-server.yaml")
		Expect(os.WriteFile(configPath, []byte(`sargantana: "this is a string not an object"`), 0644)).To(Succeed())

		opts := &options{configPath: configPath}
		_, _, err := initServer(opts)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("failed to load server configuration"))
	})

	It("should succeed with valid config", func() {
		tmpDir := GinkgoT().TempDir()
		configPath := filepath.Join(tmpDir, "valid.yaml")

		validConfig := `sargantana:
  server:
    address: :9999
    session_name: test_session
    session_secret: a_very_long_secret_key_for_testing_purposes_that_meets_minimum_requirements
  controllers:
    - type: static
      config:
        status: 200
        body: "OK"
`
		Expect(os.WriteFile(configPath, []byte(validConfig), 0644)).To(Succeed())

		opts := &options{configPath: configPath, debug: false}
		srv, closeFunc, err := initServer(opts)
		Expect(err).NotTo(HaveOccurred())
		Expect(srv).NotTo(BeNil())
		Expect(closeFunc).NotTo(BeNil())

		Expect(closeFunc()).To(Succeed())
	})
})

var _ = Describe("runServer", func() {
	It("should fail with invalid config", func() {
		opts := &options{configPath: "/nonexistent/config.yaml"}
		err := runServer(opts)
		Expect(err).To(HaveOccurred())
	})
})

var _ = Describe("registerControllers", func() {
	It("should not panic", func() {
		Expect(func() { registerControllers() }).NotTo(Panic())
	})
})
