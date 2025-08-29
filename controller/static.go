package controller

import (
	"io/fs"
	"log"
	"os"
	"path/filepath"

	"github.com/animalet/sargantana-go/config"
	"github.com/gin-gonic/gin"
)

type staticConfigurator struct {
}

func (s staticConfigurator) ForType() string {
	return "static"
}

func (s staticConfigurator) Configure(controllerConfig config.ControllerConfig, _ config.ServerConfig) (IController, error) {
	staticsDir := controllerConfig["statics_dir"].(string)
	htmlTemplatesDir := controllerConfig["templates_dir"].(string)
	log.Printf("Statics directory: %q\n", staticsDir)
	log.Printf("Templates directory: %q\n", htmlTemplatesDir)

	return &static{
		staticsDir:       staticsDir,
		htmlTemplatesDir: htmlTemplatesDir,
	}, nil
}

func NewStaticConfigurator() IControllerConfigurator {
	return &staticConfigurator{}
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
	engine.Static("/static", s.staticsDir)
	engine.GET("/", func(c *gin.Context) {
		c.Header("Content-Type", "text/html")
		c.File(s.staticsDir + "/index.html")
	})

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
			log.Printf("Error walking through templates directory: %v", err)
			return
		}

		if found {
			engine.LoadHTMLGlob(s.htmlTemplatesDir + "/**")
		} else {
			log.Printf("Templates directory present but no files found, skipping templates.")
		}
	} else {
		log.Printf("Templates directory not present: %v", err)
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
