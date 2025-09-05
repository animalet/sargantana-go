package controller

import (
	"os"

	"github.com/animalet/sargantana-go/config"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

type StaticControllerConfig []StaticMapping

type StaticMapping struct {
	Path string `yaml:"path"`
	Dir  string `yaml:"dir,omitempty"`
	File string `yaml:"file,omitempty"`
}

func (s StaticMapping) Validate() error {
	if s.Dir == "" && s.File == "" || s.Dir != "" && s.File != "" {
		return errors.New("either dir or file must be set and non-empty")
	}

	if s.File != "" {
		if stat, err := os.Stat(s.File); err != nil || stat.IsDir() {
			return errors.Wrap(err, "static file not present or is a directory")
		}
		return nil
	}

	if stat, err := os.Stat(s.Dir); err != nil || !stat.IsDir() {
		return errors.Wrap(err, "statics directory not present or is not a directory")
	}

	if s.Path == "" {
		return errors.New("url_path must be set and non-empty")
	}
	return nil
}

func (s StaticControllerConfig) Validate() error {
	if len(s) == 0 {
		return errors.New("at least one static mapping must be provided")
	}

	errslice := make([]error, 0)
	for _, mapping := range s {
		if err := mapping.Validate(); err != nil {
			errslice = append(errslice, err)
		}
	}
	if len(errslice) > 0 {
		return errors.Errorf("static mappings validation failed: %v", errslice)
	}

	return nil
}

func NewStaticController(configData config.ControllerConfig, _ config.ServerConfig) (IController, error) {
	c, err := config.UnmarshalTo[StaticControllerConfig](configData)
	if err != nil {
		return nil, err
	}

	log.Info().
		Any("mappings", c).
		Msg("Static content configured")

	return &static{
		mappings: *c,
	}, nil
}

// static is a controller that serves static files and HTML templates.
// It provides functionality for serving frontend assets like CSS, JavaScript,
// images, and HTML files, as well as Go template rendering capabilities.
type static struct {
	IController
	mappings []StaticMapping
}

// Bind registers the static controller with the provided Gin engine.
// It sets up routes for serving static files and loading HTML templates from the configured directory.
func (s *static) Bind(engine *gin.Engine, _ gin.HandlerFunc) {
	for _, mapping := range s.mappings {
		if mapping.File != "" {
			engine.StaticFile(mapping.Path, mapping.File)
			continue
		}
		engine.Static(mapping.Path, mapping.Dir)
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
