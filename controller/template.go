package controller

import (
	"io/fs"
	"os"
	"path/filepath"

	"github.com/animalet/sargantana-go/config"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

type TemplateControllerConfig struct {
	Path string `yaml:"path"`
}

func NewTemplateController(configData config.ControllerConfig, _ ControllerContext) (IController, error) {
	c, err := config.UnmarshalTo[TemplateControllerConfig](configData)
	if err != nil {
		return nil, err
	}

	log.Info().
		Str("path", c.Path).
		Msg("Templates directory configured")

	return &template{
		path: c.Path,
	}, nil
}

func (c TemplateControllerConfig) Validate() error {
	if stat, err := os.Stat(c.Path); err != nil || !stat.IsDir() {
		return errors.Wrap(err, "templates directory not present or is not a directory")
	}
	return nil
}

// static is a controller that serves static files and HTML templates.
// It provides functionality for serving frontend assets like CSS, JavaScript,
// images, and HTML files, as well as Go template rendering capabilities.
type template struct {
	IController
	path string
}

// Bind registers the template controller with the provided Gin engine.
// It sets up the HTML template rendering by loading templates from the configured directory.
func (t *template) Bind(engine *gin.Engine, _ gin.HandlerFunc) {
	if stat, err := os.Stat(t.path); err == nil && stat.IsDir() {
		var found bool
		err = filepath.WalkDir(t.path, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() {
				found = true
			}
			return nil
		})

		if err != nil {
			panic(errors.Wrap(err, "error walking through templates directory"))
		}

		if found {
			engine.LoadHTMLGlob(t.path + "/**")
		} else {
			log.Warn().Msg("Templates directory present but no files found, skipping templates.")
		}
	}
}

// Close performs cleanup for the static controller.
//
// Returns nil as no cleanup is required.
func (t *template) Close() error {
	return nil
}
