package controller

import (
	"flag"
	"io/fs"
	"log"
	"os"
	"path/filepath"

	"github.com/animalet/sargantana-go/config"
	"github.com/gin-gonic/gin"
)

type Static struct {
	IController
	staticsDir       string
	htmlTemplatesDir string
}

func NewStatic(staticsDir, htmlTemplatesDir string) *Static {
	log.Printf("Statics directory: %q\n", staticsDir)
	log.Printf("Templates directory: %q\n", htmlTemplatesDir)

	return &Static{
		staticsDir:       staticsDir,
		htmlTemplatesDir: htmlTemplatesDir,
	}
}

func NewStaticFromFlags() *Static {
	frontend := flag.String("frontend", "./frontend", "Path to the frontend static content directory")
	templates := flag.String("templates", "./templates", "Path to the templates directory")

	return NewStatic(*frontend, *templates)
}

func (s *Static) Bind(server *gin.Engine, config config.Config, loginMiddleware gin.HandlerFunc) {
	server.Static("/static", s.staticsDir)
	server.GET("/", func(c *gin.Context) {
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
			server.LoadHTMLGlob(s.htmlTemplatesDir + "/**")
		} else {
			log.Printf("Templates directory present but no files found, skipping templates.")
		}
	} else {
		log.Printf("Templates directory not present: %v", err)
	}
}

func (s *Static) Close() error {
	return nil
}
