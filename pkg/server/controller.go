package server

import (
	"github.com/animalet/sargantana-go/pkg/config"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

// IController defines the interface that all controllers must implement.
// Controllers are modular components that can be plugged into the Sargantana server
// to provide specific functionality such as authentication, static file serving,
// load balancing, or custom application logic.
type IController interface {
	// Bind registers the controller's routes and middleware with the provided Gin engine.
	// This method is called during server startup to configure the controller's endpoints.
	//
	// Parameters:
	//   - engine: The Gin HTTP router engine to register routes with
	//   - loginMiddleware: Pre-configured middleware function for protecting routes that require authentication.
	//     This can be nil if no authentication middleware is configured.
	Bind(engine *gin.Engine, loginMiddleware gin.HandlerFunc) error

	// Close performs cleanup operations when the controller is being shut down.
	// This method should release any resources held by the controller such as
	// database connections, file handles, or background goroutines not managed externally.
	//
	// Returns an error if cleanup fails, nil otherwise.
	Close() error
}

// ControllerContext provides runtime dependencies and configuration to controllers
// during instantiation. This separates pure YAML configuration (WebServerConfig) from
// runtime dependencies like session stores, databases, or loggers.
//
// This pattern allows controllers to access necessary dependencies without
// coupling to the server's lifecycle methods (Start, Stop, etc.).
type ControllerContext struct {
	// ServerConfig contains the pure YAML configuration from the config file
	ServerConfig WebServerConfig

	// SessionStore is the session storage implementation (cookie-based or Redis-based)
	// Used by authentication controllers to configure gothic.Store
	SessionStore sessions.Store

	// Future additions can include:
	// Logger       *zerolog.Logger
	// Metrics      MetricsCollector
	// Database     *sql.DB
}

// ControllerFactory is a factory function that creates a new controller instance.
// It receives the controller-specific configuration and a context with runtime dependencies.
type ControllerFactory func(controllerConfig config.ModuleRawConfig, ctx ControllerContext) (IController, error)

// ControllerBinding represents the configuration for a single controller.
type ControllerBinding struct {
	Config   config.ModuleRawConfig `yaml:"config"`
	TypeName string                 `yaml:"type"`
	Name     string                 `yaml:"name,omitempty"`
}

type ControllerBindings []ControllerBinding

func (c ControllerBindings) Validate() error {
	var validationErrors []error
	for i, binding := range c {
		if err := binding.Validate(); err != nil {
			validationErrors = append(validationErrors, errors.Wrapf(err, "controller binding at index %d is invalid", i))
		}
	}

	if len(validationErrors) > 0 {
		return errors.Errorf("configuration validation failed: %v", validationErrors)
	}

	return nil
}

// Validate checks if the ControllerBinding has all required fields set.
// Note: Name is optional and will be auto-generated if not provided.
func (c ControllerBinding) Validate() error {
	if c.TypeName == "" {
		return errors.New("controller type must be set and non-empty")
	}
	if c.Config == nil {
		return errors.New("controller config must be provided")
	}
	return nil
}
