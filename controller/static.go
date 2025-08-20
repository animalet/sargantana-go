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
		// Only load templates if there are any .html files
		var found bool
		filepath.WalkDir(s.htmlTemplatesDir, func(path string, d fs.DirEntry, err error) error {
			if !d.IsDir() && filepath.Ext(d.Name()) == ".html" {
				found = true
			}
			return nil
		})
		if found {
			engine.LoadHTMLGlob(s.htmlTemplatesDir + "/**")
		} else {
			log.Printf("Templates directory present but no .html files found, skipping LoadHTMLGlob")
		}
	} else {
		log.Printf("Templates directory not present: %v", err)
	}
}

func (s Static) Close() {
}
