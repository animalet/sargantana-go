// Package controller provides the core controller interface and implementations
// for the Sargantana Go web framework. Controllers handle specific aspects of
// web application functionality such as authentication, static file serving,
// and load balancing.
package controller

import (
	"github.com/animalet/sargantana-go/config"
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

// Constructor is a factory function that creates a new controller instance.
type Constructor func(controllerConfig config.ControllerConfig, serverConfig config.ServerConfig) (IController, error)
