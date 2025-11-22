//go:build unit

package controller_test

import (
	"net/http/httptest"

	"github.com/animalet/sargantana-go/pkg/controller"
	"github.com/animalet/sargantana-go/pkg/server"
	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
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
			cfg := controller.StaticControllerConfig{Dir: "."}
			err := cfg.Validate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("path must be set"))
		})

		It("should return error if neither dir nor file is set", func() {
			cfg := controller.StaticControllerConfig{Path: "/static"}
			err := cfg.Validate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("either dir or file must be set"))
		})

		It("should return error if both dir and file are set", func() {
			cfg := controller.StaticControllerConfig{Path: "/static", Dir: ".", File: "file"}
			err := cfg.Validate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("cannot set both dir and file"))
		})

		It("should return error if file does not exist", func() {
			cfg := controller.StaticControllerConfig{Path: "/static", File: "nonexistent"}
			err := cfg.Validate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("static file not present"))
		})

		It("should return error if dir does not exist", func() {
			cfg := controller.StaticControllerConfig{Path: "/static", Dir: "nonexistent"}
			err := cfg.Validate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("statics directory not present"))
		})
	})

	Context("NewStaticController", func() {
		It("should create a new controller with directory", func() {
			cfg := []byte(`path: /static
dir: ./testdata`)
			ctrl, err := controller.NewStaticController(cfg, server.ControllerContext{})
			Expect(err).NotTo(HaveOccurred())
			Expect(ctrl).NotTo(BeNil())

			ctrl.Bind(engine)
			Expect(ctrl.Close()).To(Succeed())
		})

		It("should create a new controller with file", func() {
			cfg := []byte(`path: /file
file: ./testdata/test.txt`)
			ctrl, err := controller.NewStaticController(cfg, server.ControllerContext{})
			Expect(err).NotTo(HaveOccurred())
			Expect(ctrl).NotTo(BeNil())

			ctrl.Bind(engine)
			Expect(ctrl.Close()).To(Succeed())
		})
	})
})
