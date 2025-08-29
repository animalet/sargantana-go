package controller

import (
	"flag"
	"io/fs"
	"log"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

// static is a controller that serves static files and HTML templates.
// It provides functionality for serving frontend assets like CSS, JavaScript,
// images, and HTML files, as well as Go template rendering capabilities.
type static struct {
	IController
	staticsDir       string
	htmlTemplatesDir string
}

// NewStatic creates a new static controller with the specified directories.
// It sets up the controller to serve static files from staticsDir and
// load HTML templates from htmlTemplatesDir.
//
// Parameters:
//   - staticsDir: Directory path containing static files (CSS, JS, images, etc.)
//   - htmlTemplatesDir: Directory path containing HTML template files
//
// Returns a pointer to the configured static controller.
func NewStatic(staticsDir, htmlTemplatesDir string) IController {
	log.Printf("Statics directory: %q\n", staticsDir)
	log.Printf("Templates directory: %q\n", htmlTemplatesDir)

	return &static{
		staticsDir:       staticsDir,
		htmlTemplatesDir: htmlTemplatesDir,
	}
}

// NewStaticFromFlags creates a static controller factory function that reads
// configuration from command-line flags. This function is designed to be used
// with the server's flag-based initialization system.
//
// The following flags are registered:
//   - frontend: Path to the frontend static content directory (default: "./frontend")
//   - templates: Path to the templates directory (default: "./templates")
//
// Parameters:
//   - flagSet: The flag set to register the static controller flags with
//
// Returns a factory function that creates a static controller when called.
func NewStaticFromFlags(flagSet *flag.FlagSet) func() IController {
	frontend := flagSet.String("frontend", "./frontend", "Path to the frontend static content directory")
	templates := flagSet.String("templates", "./templates", "Path to the templates directory")
	return func() IController { return NewStatic(*frontend, *templates) }
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
func (s *static) Bind(engine *gin.Engine, loginMiddleware gin.HandlerFunc) {
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
