//go:build unit

package controller

import (
	"os"
	"path/filepath"

	"github.com/animalet/sargantana-go/pkg/config"
	"github.com/animalet/sargantana-go/pkg/server"
	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v3"
)

var _ = Describe("TemplateController", func() {
	var (
		tempDir string
	)

	BeforeEach(func() {
		var err error
		tempDir, err = os.MkdirTemp("", "templates")
		Expect(err).NotTo(HaveOccurred())
		gin.SetMode(gin.TestMode)
	})

	AfterEach(func() {
		os.RemoveAll(tempDir)
	})

	Context("Configuration", func() {
		It("should create a new template controller", func() {
			rawConfigMap := map[string]interface{}{
				"path": tempDir,
			}
			rawConfigBytes, _ := yaml.Marshal(rawConfigMap)
			rawConfig := config.ModuleRawConfig(rawConfigBytes)

			var tmplCfg TemplateControllerConfig
			err := yaml.Unmarshal(rawConfig, &tmplCfg)
			Expect(err).NotTo(HaveOccurred())

			ctx := server.ControllerContext{}
			ctrl, err := NewTemplateController(&tmplCfg, ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(ctrl).NotTo(BeNil())
		})

		It("should fail if directory does not exist", func() {
			rawConfigMap := map[string]interface{}{
				"path": "/non/existent/path",
			}
			rawConfigBytes, _ := yaml.Marshal(rawConfigMap)
			rawConfig := config.ModuleRawConfig(rawConfigBytes)

			var tmplCfg TemplateControllerConfig
			err := yaml.Unmarshal(rawConfig, &tmplCfg)
			Expect(err).NotTo(HaveOccurred())

			err = tmplCfg.Validate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("templates directory not present"))
		})

		It("should fail if path is not a directory", func() {
			// Create a file instead of a directory
			tempFile := filepath.Join(tempDir, "notadir")
			err := os.WriteFile(tempFile, []byte("test"), 0644)
			Expect(err).NotTo(HaveOccurred())

			tmplCfg := TemplateControllerConfig{
				Path: tempFile,
			}

			err = tmplCfg.Validate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("templates path is not a directory"))
		})
	})

	Context("Binding", func() {
		It("should load templates", func() {
			// Create a dummy template file
			err := os.WriteFile(filepath.Join(tempDir, "index.html"), []byte("<h1>Hello</h1>"), 0644)
			Expect(err).NotTo(HaveOccurred())

			rawConfigMap := map[string]interface{}{
				"path": tempDir,
			}
			rawConfigBytes, _ := yaml.Marshal(rawConfigMap)
			rawConfig := config.ModuleRawConfig(rawConfigBytes)

			var tmplCfg TemplateControllerConfig
			err = yaml.Unmarshal(rawConfig, &tmplCfg)
			Expect(err).NotTo(HaveOccurred())

			ctx := server.ControllerContext{}
			ctrl, _ := NewTemplateController(&tmplCfg, ctx)

			engine := gin.New()
			// This might panic if templates are invalid or not found, but we expect it to work
			ctrl.Bind(engine, nil)

			// Verify if we can render (requires a route using it, but Bind just loads them)
			// We can at least ensure Bind didn't panic
		})

		It("should handle empty templates directory", func() {
			// Create an empty directory with no template files
			emptyDir := filepath.Join(tempDir, "empty")
			err := os.Mkdir(emptyDir, 0755)
			Expect(err).NotTo(HaveOccurred())

			tmplCfg := TemplateControllerConfig{
				Path: emptyDir,
			}

			ctx := server.ControllerContext{}
			ctrl, err := NewTemplateController(&tmplCfg, ctx)
			Expect(err).NotTo(HaveOccurred())

			engine := gin.New()
			// Should log a warning but not panic
			ctrl.Bind(engine, nil)
		})

		It("should handle Close without error", func() {
			tmplCfg := TemplateControllerConfig{
				Path: tempDir,
			}

			ctx := server.ControllerContext{}
			ctrl, _ := NewTemplateController(&tmplCfg, ctx)

			err := ctrl.Close()
			Expect(err).NotTo(HaveOccurred())
		})

		It("should handle non-existent path in Bind gracefully", func() {
			tmplCfg := TemplateControllerConfig{
				Path: "/non/existent/path",
			}

			ctx := server.ControllerContext{}
			ctrl, _ := NewTemplateController(&tmplCfg, ctx)

			engine := gin.New()
			// Should not panic even if path doesn't exist (Bind checks)
			ctrl.Bind(engine, nil)
		})
	})
})
