package controller

import (
	"io/fs"
	"log"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

type Static struct {
	IController
	staticsDir       string
	htmlTemplatesDir string
}

func NewStatic(staticsDir, htmlTemplatesDir string) *Static {
	return &Static{
		staticsDir:       staticsDir,
		htmlTemplatesDir: htmlTemplatesDir,
	}
}

func (s Static) Bind(engine *gin.Engine, _ gin.HandlerFunc) {
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

func (s Static) Close() error {
	return nil
}
