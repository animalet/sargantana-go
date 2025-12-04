package controller

import (
	"os"

	"github.com/animalet/sargantana-go/internal/deepcopy"
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
	Auth bool   `yaml:"auth,omitempty"` // If true, requires authentication to access static content
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
	// Deep copy the config to enforce immutability
	configCopy := deepcopy.MustCopy(c)

	log.Info().
		Str("path", configCopy.Path).
		Str("dir", configCopy.Dir).
		Str("file", configCopy.File).
		Bool("auth", configCopy.Auth).
		Msg("Static content configured")

	return &static{
		path: configCopy.Path,
		dir:  configCopy.Dir,
		file: configCopy.File,
		auth: configCopy.Auth,
	}, nil
}

// static is a controller that serves static files or directories.
// Each instance handles a single path mapping to either a directory or a file.
// Fields are extracted from configuration at initialization time for immutability.
type static struct {
	server.IController
	path string
	dir  string
	file string
	auth bool
}

// Bind registers the static controller with the provided Gin engine.
// It sets up routes for serving static files or directories from the configured path.
// If authentication is enabled, the loginMiddleware is applied to protect the static content.
func (s *static) Bind(engine *gin.Engine, loginMiddleware gin.HandlerFunc) error {
	isFile := s.file != ""

	log.Info().
		Str("path", s.path).
		Str("file", s.file).
		Str("dir", s.dir).
		Bool("auth", s.auth).
		Msgf("Binding static %s", map[bool]string{true: "file", false: "directory"}[isFile])

	if isFile {
		if s.auth {
			engine.GET(s.path, loginMiddleware, func(c *gin.Context) {
				c.File(s.file)
			})
		} else {
			engine.StaticFile(s.path, s.file)
		}
	} else {
		if s.auth {
			group := engine.Group(s.path, loginMiddleware)
			group.Static("/", s.dir)
		} else {
			engine.Static(s.path, s.dir)
		}
	}
	return nil
}

// Close performs cleanup for the static controller.
// Since the static controller doesn't hold any persistent resources,
// this method always returns nil.
//
// Returns nil as no cleanup is required.
func (s *static) Close() error {
	return nil
}
