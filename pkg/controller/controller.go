// Package controller provides the core controller interface and implementations
// for the Sargantana Go web framework. Controllers handle specific aspects of
// web application functionality such as authentication, static file serving,
// and load balancing.
package controller

import (
	"github.com/animalet/sargantana-go/pkg/config"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
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
	//   - loginMiddleware: Pre-configured middleware function for protecting routes that require authentication
	Bind(engine *gin.Engine, loginMiddleware gin.HandlerFunc)

	// Close performs cleanup operations when the controller is being shut down.
	// This method should release any resources held by the controller such as
	// database connections, file handles, or background goroutines.
	//
	// Returns an error if cleanup fails, nil otherwise.
	Close() error
}

// ControllerContext provides runtime dependencies and configuration to controllers
// during instantiation. This separates pure YAML configuration (ServerConfig) from
// runtime dependencies like session stores, databases, or loggers.
//
// This pattern allows controllers to access necessary dependencies without
// coupling to the server's lifecycle methods (Start, Stop, etc.).
type ControllerContext struct {
	// ServerConfig contains the pure YAML configuration from the config file
	ServerConfig config.ServerConfig

	// SessionStore is the session storage implementation (cookie-based or Redis-based)
	// Used by authentication controllers to configure gothic.Store
	SessionStore *sessions.Store

	// Future additions can include:
	// Logger       *zerolog.Logger
	// Metrics      MetricsCollector
	// Database     *sql.DB
}

// Constructor is a factory function that creates a new controller instance.
// It receives the controller-specific configuration and a context with runtime dependencies.
type Constructor func(controllerConfig config.ControllerConfig, ctx ControllerContext) (IController, error)
