// Package server provides the core HTTP server implementation for the Sargantana Go web framework.
// It handles server lifecycle management, controller registration, session configuration,
// secrets loading, and graceful shutdown functionality.
package server

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/animalet/sargantana-go/config"
	"github.com/animalet/sargantana-go/controller"
	"github.com/animalet/sargantana-go/database"
	"github.com/animalet/sargantana-go/session"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

// Server represents the main HTTP server instance for the Sargantana Go framework.
// It encapsulates the server configuration, HTTP server instance, shutdown hooks,
// and signal handling for graceful shutdown.
type Server struct {
	config          *config.Config
	httpServer      *http.Server
	shutdownHooks   []func() error
	shutdownChannel chan os.Signal
	controllers     []controller.IController
}

// serverFlags holds all the parsed flag values
type serverFlags struct {
	debug       *bool
	secretsDir  *string
	host        *string
	port        *int
	redis       *string
	sessionName *string
	showVersion *bool
	config      *string
}

// NewServer creates a new Server instance by loading configuration from the specified file.
//
// Parameters:
//   - configFile: Path to the configuration file to load settings from
//
// Returns:
//   - *Server: The configured server instance
func NewServer(configFile string, controllerConfigurators ...controller.IControllerConfigurator) (*Server, error) {
	c, err := config.Load(configFile)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("failed to load config file: %s", configFile))
	}
	configMap, err := registerConfigurators(controllerConfigurators)
	if err != nil {
		return nil, errors.Wrap(err, "failed to register controller configurators")
	}

	var controllers []controller.IController
	for _, binding := range c.ControllerBindings {
		forType := binding.TypeName
		configurator, exists := configMap[forType]
		if !exists {
			return nil, fmt.Errorf("no configurator found for controller type: %s", forType)
		}
		newController := configurator.Configure(binding.ConfigData, c.ServerConfig)
		if newController == nil {
			return nil, fmt.Errorf("failed to configure controller of type: %s", forType)
		}
		controllers = append(controllers, newController)
	}
	log.Println("Configuration loaded successfully")
	return &Server{config: c, controllers: controllers}, nil
}

func registerConfigurators(configurators []controller.IControllerConfigurator) (map[string]controller.IControllerConfigurator, error) {
	var configMap = make(map[string]controller.IControllerConfigurator)
	for _, binding := range configurators {
		forType := binding.ForType()
		if _, exists := configMap[forType]; exists {
			return nil, fmt.Errorf("duplicate controller type: %s", forType)
		}
		configMap[forType] = binding
	}

	return configMap, nil
}

// StartAndWaitForSignal starts the HTTP server and waits for an OS signal to gracefully shut it down.
// It handles initialization, secret loading, and controller registration before starting the server.
// Upon receiving a termination signal, it gracefully shuts down the server, allowing active connections
// to complete and freeing up resources.
func (s *Server) StartAndWaitForSignal() error {
	err := s.Start()
	if err != nil {
		return err
	}
	return s.waitForSignal()
}

// Start initializes the server components and starts listening for incoming HTTP requests.
// It configures the server based on the provided flags, loads secrets, and sets up the router and middleware.
// This function must be called before the server can handle requests.
func (s *Server) Start(controllerInitializers ...controller.IControllerConfigurator) error {
	if s.config.ServerConfig.Debug {
		log.Printf("Debug mode is enabled\n")
		log.Printf("Secrets directory: %q\n", s.config.ServerConfig.SecretsDir)
		log.Printf("Listen address: %q\n", s.config.ServerConfig.Address)
		if s.config.ServerConfig.RedisSessionStore == "" {
			log.Printf("Use cookies for session storage\n")
		} else {
			log.Printf("Use Redis for session storage: %s\n", s.config.ServerConfig.RedisSessionStore)
		}
		log.Printf("Session cookie name: %q\n", s.config.ServerConfig.SessionName)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	if err := s.bootstrap(); err != nil {
		return err
	}

	return nil
}

func (s *Server) bootstrap() error {
	err := s.config.LoadSecrets()
	if err != nil {
		return err
	}

	engine := gin.New()
	isReleaseMode := gin.Mode() == gin.ReleaseMode
	sessionStore, err := s.createSessionStore(isReleaseMode, err)
	if err != nil {
		return err
	}

	if isReleaseMode {
		log.Println("Running in release mode")
		err := engine.SetTrustedProxies(nil)
		if err != nil {
			return err
		}
		engine.Use(gin.ErrorLoggerT(gin.ErrorTypePrivate))
	} else {
		engine.Use(ginBodyLogMiddleware, gin.ErrorLogger())

	}
	engine.Use(
		gin.Logger(),
		gin.Recovery(),
		sessions.Sessions(s.config.ServerConfig.SessionName, sessionStore),
	)

	for _, c := range s.controllers {
		c.Bind(engine, controller.LoginFunc)
		s.addShutdownHook(c.Close)
	}

	s.httpServer = &http.Server{
		Addr:    s.config.ServerConfig.Address,
		Handler: engine,
	}

	log.Println("Starting server on " + s.config.ServerConfig.Address)
	s.listenAndServe()

	return nil
}

func (s *Server) createSessionStore(isReleaseMode bool, err error) (sessions.Store, error) {
	var sessionStore sessions.Store
	secret := s.config.ServerConfig.SessionSecret
	if secret == "" {
		return nil, fmt.Errorf("session secret is not set")
	}
	sessionSecret := []byte(os.ExpandEnv(secret))
	if s.config.ServerConfig.RedisSessionStore == "" {
		log.Println("Using cookies for session storage")
		sessionStore = session.NewCookieStore(isReleaseMode, sessionSecret)
	} else {
		log.Println("Using Redis for session storage")
		pool := database.NewRedisPool(s.config.ServerConfig.RedisSessionStore)
		s.addShutdownHook(func() error { return pool.Close() })
		sessionStore, err = session.NewRedisSessionStore(isReleaseMode, sessionSecret, pool)
		if err != nil {
			return nil, fmt.Errorf("error creating Redis session store: %v", err)
		}
	}
	return sessionStore, nil
}

func (s *Server) waitForSignal() error {
	s.shutdownChannel = make(chan os.Signal, 1)
	// kill (no params) by default sends syscall.SIGTERM
	// kill -2 is syscall.SIGINT
	// kill -9 is syscall.SIGKILL but can't be caught, so don't need to add it
	signal.Notify(s.shutdownChannel, syscall.SIGINT, syscall.SIGTERM)
	log.Printf("Shutdown signal received (%s)\n", <-s.shutdownChannel)
	return s.Shutdown()
}

func (s *Server) listenAndServe() {
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Listen error: %s", err)
		}
	}()
}

func (s *Server) addShutdownHook(f func() error) {
	s.shutdownHooks = append(s.shutdownHooks, f)
}

// Shutdown gracefully shuts down the server, waiting for active connections to complete
// and freeing up resources. It executes registered shutdown hooks in the process.
func (s *Server) Shutdown() error {
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := s.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("forced shutdown: %s", err)
	}

	// Free up resources used by controllers
	log.Println("Executing shutdown hooks...")
	for _, hook := range s.shutdownHooks {
		if err := hook(); err != nil {
			log.Printf("Error during shutdown hook: %s", err)
		}
	}

	log.Println("Server exited gracefully")
	return nil
}

type bodyLogWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w bodyLogWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func ginBodyLogMiddleware(c *gin.Context) {
	blw := &bodyLogWriter{body: bytes.NewBufferString(""), ResponseWriter: c.Writer}
	c.Writer = blw
	c.Next()
	log.Printf("Response body: %s", blw.body.String())
}
