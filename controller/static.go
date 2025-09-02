package controller

import (
	"io/fs"
	"os"
	"path/filepath"

	"github.com/animalet/sargantana-go/config"
	"github.com/animalet/sargantana-go/logger"
	"github.com/gin-gonic/gin"
)

type StaticControllerConfig struct {
	StaticsDir       string `yaml:"statics_dir,omitempty"`
	HtmlTemplatesDir string `yaml:"templates_dir,omitempty"`
}

func NewStaticController(configData config.ControllerConfig, _ config.ServerConfig) (IController, error) {
	c, err := config.UnmarshalTo[StaticControllerConfig](configData)
	if err != nil {
		return nil, err
	}

	logger.Infof("Statics directory: %q", c.StaticsDir)
	logger.Infof("Templates directory: %q", c.HtmlTemplatesDir)

	// Ensure the statics directory exists
	if stat, err := os.Stat(c.StaticsDir); err != nil || !stat.IsDir() {
		logger.Warnf("Statics directory %q does not exist or is not a directory. Continuing without statics.", c.StaticsDir)
	}

	// Ensure the templates directory exists (if provided)
	if stat, err := os.Stat(c.HtmlTemplatesDir); err != nil || !stat.IsDir() {
		logger.Warnf("Templates directory %q does not exist or is not a directory. Continuing without templates.", c.HtmlTemplatesDir)
	}
	return &static{
		staticsDir:       c.StaticsDir,
		htmlTemplatesDir: c.HtmlTemplatesDir,
	}, nil
}

// static is a controller that serves static files and HTML templates.
// It provides functionality for serving frontend assets like CSS, JavaScript,
// images, and HTML files, as well as Go template rendering capabilities.
type static struct {
	IController
	staticsDir       string
	htmlTemplatesDir string
}

// Bind registers the static file serving routes with the Gin engine.
// It sets up the following routes:
//   - /static/*: Serves static files from the configured static directory
//   - /: Serves the index.html file from the static directory
//
// Additionally, it loads HTML templates from the templates directory if available.
// Templates are loaded using Go's html/template package and can be rendered
// using Gin's template rendering functions.
//
// Parameters:
//   - server: The Gin engine to register routes with
//   - _: Server configuration (unused by this controller)
//   - _: Login middleware function (unused by this controller)
func (s *static) Bind(engine *gin.Engine, _ gin.HandlerFunc) {
	if s.staticsDir != "" {
		engine.Static("/static", s.staticsDir)
		engine.GET("/", func(c *gin.Context) {
			c.Header("Content-Type", "text/html")
			c.File(s.staticsDir + "/index.html")
		})
	}

	if s.htmlTemplatesDir != "" {
		if stat, err := os.Stat(s.htmlTemplatesDir); stat != nil && stat.IsDir() {
			// check if dir is empty
			var found bool
			err = filepath.WalkDir(s.htmlTemplatesDir, func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if !d.IsDir() {
					found = true
				}
				return nil
			})

			if err != nil {
				logger.Warnf("Error walking through templates directory: %v", err)
				return
			}

			if found {
				engine.LoadHTMLGlob(s.htmlTemplatesDir + "/**")
			} else {
				logger.Warn("Templates directory present but no files found, skipping templates.")
			}
		} else {
			logger.Warnf("Templates directory not present: %v", err)
		}
	}
}

// Close performs cleanup for the static controller.
// Since the static controller doesn't hold any persistent resources,
// this method always returns nil.
//
// Returns nil as no cleanup is required.
func (s *static) Close() error {
	return nil
}
