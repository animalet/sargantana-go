package controller

import (
	"os"

	"github.com/animalet/sargantana-go/config"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

type StaticControllerConfig struct {
	UrlPath string `yaml:"url_path"`
	Path    string `yaml:"statics_dir"`
}

func (s StaticControllerConfig) Validate() error {
	if stat, err := os.Stat(s.Path); err != nil || !stat.IsDir() {
		return errors.Wrap(err, "statics directory not present or is not a directory")
	}

	if s.UrlPath == "" {
		return errors.New("url_path must be set and non-empty")
	}
	return nil
}

func NewStaticController(configData config.ControllerConfig, _ config.ServerConfig) (IController, error) {
	c, err := config.UnmarshalTo[StaticControllerConfig](configData)
	if err != nil {
		return nil, err
	}

	log.Info().
		Str("path", c.Path).
		Str("url_path", c.UrlPath).
		Msg("Static content configured")

	return &static{
		urlPath: c.UrlPath,
		path:    c.Path,
	}, nil
}

// static is a controller that serves static files and HTML templates.
// It provides functionality for serving frontend assets like CSS, JavaScript,
// images, and HTML files, as well as Go template rendering capabilities.
type static struct {
	IController
	urlPath string
	path    string
}

// Bind registers the static controller with the provided Gin engine.
// It sets up routes for serving static files and loading HTML templates from the configured directory.
func (s *static) Bind(engine *gin.Engine, _ gin.HandlerFunc) {
	if s.urlPath != "" {
		engine.Static("/static", s.path)
		engine.GET("/", func(c *gin.Context) {
			c.Header("Content-Type", "text/html")
			c.File(s.path + "/index.html")
		})
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
