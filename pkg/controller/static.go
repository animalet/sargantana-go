package controller

import (
	"os"

	"github.com/animalet/sargantana-go/pkg/server"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

// StaticControllerConfig defines the configuration for a static file controller.
// Each controller instance serves a single path with either a directory or a single file.
type StaticControllerConfig struct {
	Path string `yaml:"path"`
	Dir  string `yaml:"dir,omitempty"`
	File string `yaml:"file,omitempty"`
}

func (s StaticControllerConfig) Validate() error {
	if s.Path == "" {
		return errors.New("path must be set and non-empty")
	}

	if s.Dir == "" && s.File == "" {
		return errors.New("either dir or file must be set and non-empty")
	}

	if s.Dir != "" && s.File != "" {
		return errors.New("cannot set both dir and file, choose one")
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

	return nil
}

func NewStaticController(c *StaticControllerConfig, _ server.ControllerContext) (server.IController, error) {

	log.Info().
		Str("path", c.Path).
		Str("dir", c.Dir).
		Str("file", c.File).
		Msg("Static content configured")

	return &static{
		config: *c,
	}, nil
}

// static is a controller that serves static files or directories.
// Each instance handles a single path mapping to either a directory or a file.
type static struct {
	server.IController
	config StaticControllerConfig
}

// Bind registers the static controller with the provided Gin engine.
// It sets up routes for serving static files or directories from the configured path.
func (s *static) Bind(engine *gin.Engine) {
	if s.config.File != "" {
		log.Info().Str("path", s.config.Path).Str("file", s.config.File).Msg("Binding static file")
		engine.StaticFile(s.config.Path, s.config.File)
	} else {
		log.Info().Str("path", s.config.Path).Str("dir", s.config.Dir).Msg("Binding static directory")
		engine.Static(s.config.Path, s.config.Dir)
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
