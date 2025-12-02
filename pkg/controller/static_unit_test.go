//go:build unit

package controller

import (
	"net/http/httptest"

	"github.com/animalet/sargantana-go/pkg/server"
	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v3"
)

var _ = Describe("Static Controller", func() {
	var (
		engine *gin.Engine
		w      *httptest.ResponseRecorder
	)

	BeforeEach(func() {
		gin.SetMode(gin.TestMode)
		w = httptest.NewRecorder()
		_, engine = gin.CreateTestContext(w)
	})

	Context("StaticControllerConfig Validate", func() {
		It("should return error if path is empty", func() {
			cfg := StaticControllerConfig{Dir: "."}
			err := cfg.Validate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("path must be set"))
		})

		It("should return error if neither dir nor file is set", func() {
			cfg := StaticControllerConfig{Path: "/static"}
			err := cfg.Validate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("either dir or file must be set"))
		})

		It("should return error if both dir and file are set", func() {
			cfg := StaticControllerConfig{Path: "/static", Dir: ".", File: "file"}
			err := cfg.Validate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("cannot set both dir and file"))
		})

		It("should return error if file does not exist", func() {
			cfg := StaticControllerConfig{Path: "/static", File: "nonexistent"}
			err := cfg.Validate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("static file not present"))
		})

		It("should return error if dir does not exist", func() {
			cfg := StaticControllerConfig{Path: "/static", Dir: "nonexistent"}
			err := cfg.Validate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("statics directory not present"))
		})

		It("should pass validation with valid file", func() {
			cfg := StaticControllerConfig{Path: "/file", File: "./testdata/test.txt"}
			err := cfg.Validate()
			Expect(err).NotTo(HaveOccurred())
		})

		It("should pass validation with valid directory", func() {
			cfg := StaticControllerConfig{Path: "/static", Dir: "./testdata"}
			err := cfg.Validate()
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("NewStaticController", func() {
		It("should create a new controller with directory", func() {
			cfg := []byte(`path: /static
dir: ./testdata`)
			var staticCfg StaticControllerConfig
			err := yaml.Unmarshal(cfg, &staticCfg)
			Expect(err).NotTo(HaveOccurred())

			ctrl, err := NewStaticController(&staticCfg, server.ControllerContext{})
			Expect(err).NotTo(HaveOccurred())
			Expect(ctrl).NotTo(BeNil())

			ctrl.Bind(engine, nil)
			Expect(ctrl.Close()).To(Succeed())
		})

		It("should create a new controller with file", func() {
			cfg := []byte(`path: /file
file: ./testdata/test.txt`)
			var staticCfg StaticControllerConfig
			err := yaml.Unmarshal(cfg, &staticCfg)
			Expect(err).NotTo(HaveOccurred())

			ctrl, err := NewStaticController(&staticCfg, server.ControllerContext{})
			Expect(err).NotTo(HaveOccurred())
			Expect(ctrl).NotTo(BeNil())

			ctrl.Bind(engine, nil)
			Expect(ctrl.Close()).To(Succeed())
		})
	})
})
