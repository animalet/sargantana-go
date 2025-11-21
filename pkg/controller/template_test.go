//go:build unit

package controller_test

import (
	"os"
	"path/filepath"

	"github.com/animalet/sargantana-go/pkg/config"
	"github.com/animalet/sargantana-go/pkg/controller"
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

			ctx := server.ControllerContext{}
			ctrl, err := controller.NewTemplateController(rawConfig, ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(ctrl).NotTo(BeNil())
		})

		It("should fail if directory does not exist", func() {
			rawConfigMap := map[string]interface{}{
				"path": "/non/existent/path",
			}
			rawConfigBytes, _ := yaml.Marshal(rawConfigMap)
			rawConfig := config.ModuleRawConfig(rawConfigBytes)

			ctx := server.ControllerContext{}
			_, err := controller.NewTemplateController(rawConfig, ctx)
			Expect(err).To(HaveOccurred())
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

			ctx := server.ControllerContext{}
			ctrl, _ := controller.NewTemplateController(rawConfig, ctx)

			engine := gin.New()
			// This might panic if templates are invalid or not found, but we expect it to work
			ctrl.Bind(engine)

			// Verify if we can render (requires a route using it, but Bind just loads them)
			// We can at least ensure Bind didn't panic
		})
	})
})
