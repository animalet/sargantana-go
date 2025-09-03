// Package server provides the core HTTP server implementation for the Sargantana Go web framework.
// It handles server lifecycle management, controller registration, session configuration,
// secrets loading, and graceful shutdown functionality.
package server

import (
	"bytes"
	"context"
	"fmt"
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
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
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

// controllerRegistry holds the mapping of controller type names to their factory functions.
var controllerRegistry = make(map[string]controller.Constructor)
var debug = false

func SetDebug(debugEnabled bool) {
	debug = debugEnabled
	if debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}
}

func GetDebug() bool {
	return debug
}

// NewServer creates a new Server instance by loading configuration from the specified file.
//
// Parameters:
//   - configFile: Path to the configuration file to load settings from
//
// Returns:
//   - *Server: The configured server instance
//   - error: An error if the server could not be created, nil otherwise
func NewServer(configFile string) (*Server, error) {
	c, err := config.Load(configFile)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("failed to load config file: %s", configFile))
	}

	controllers, configurationErrors := configureControllers(c)
	if len(configurationErrors) > 0 {
		log.Error().Msg("Configuration errors encountered, affected controllers have been excluded from bootstrap:")
		for _, configErr := range configurationErrors {
			log.Error().Msgf(" - %v", configErr)
		}
	} else {
		log.Info().Msg("Configuration loaded successfully")
	}

	return &Server{config: c, controllers: controllers}, nil
}

func AddControllerType(typeName string, factory controller.Constructor) {
	log.Info().Msgf("Registering controller type %q", typeName)
	_, exists := controllerRegistry[typeName]
	if exists {
		log.Warn().Msgf("Controller type %q is already registered, overriding", typeName)
	}
	controllerRegistry[typeName] = factory
}

func configureControllers(c *config.Config) (controllers []controller.IController, configErrors []error) {
	for _, binding := range c.ControllerBindings {
		name := "unnamed"
		if binding.Name != "" {
			name = fmt.Sprintf("%q", binding.Name)
		}

		factory, exists := controllerRegistry[binding.TypeName]
		if !exists {
			configErrors = append(configErrors, fmt.Errorf("no configurator found for %s controller type: %q", name, binding.TypeName))
			continue
		}

		newController, err := newController(c, name, binding, factory)
		if err == nil {
			controllers = append(controllers, newController)
		} else {
			configErrors = append(configErrors, fmt.Errorf("error configuring %q controller of type %q: %v", name, binding.TypeName, err))
		}
	}
	return controllers, configErrors
}

func newController(c *config.Config, name string, binding config.ControllerBinding, factory controller.Constructor) (newController controller.IController, err error) {
	log.Info().Msgf("Configuring %s controller of type: %s", name, binding.TypeName)
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic during %q controller configuration, controller was not added: %v", name, r)
			newController = nil
		}
	}()
	if newController, err = factory(binding.ConfigData, c.ServerConfig); err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("failed to configure %s controller of type: %s", name, binding.TypeName))
	}
	return newController, err
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
func (s *Server) Start() error {
	if debug {
		log.Debug().Msg("Debug mode is enabled")
		if s.config.ServerConfig.SecretsDir == "" {
			log.Debug().Msg("No secrets directory configured")
		} else {
			log.Debug().Msgf("Secrets directory: %q", s.config.ServerConfig.SecretsDir)
		}
		log.Debug().Msgf("Listen address: %q", s.config.ServerConfig.Address)
		if s.config.ServerConfig.RedisSessionStore == nil {
			log.Debug().Msg("Use cookies for session storage")
		} else {
			log.Debug().Msgf(
				"Use Redis for session storage at %q, DB: %d, TLS: %+v",
				s.config.ServerConfig.RedisSessionStore.Address,
				s.config.ServerConfig.RedisSessionStore.Database,
				s.config.ServerConfig.RedisSessionStore.TLS)
		}
		log.Debug().Msgf("Session cookie name: %q", s.config.ServerConfig.SessionName)
		if s.config.Vault != nil {
			log.Debug().Msgf("Using Vault for secrets at %q, path: %q, namespace: %q", s.config.Vault.Address, s.config.Vault.Path, s.config.Vault.Namespace)
		} else {
			log.Debug().Msg("Not using Vault for secrets")
		}
		log.Debug().Msg("Expected controllers:")
		for _, binding := range s.config.ControllerBindings {
			log.Debug().Msgf(" - Type: %s, Name: %s, Config Type: %s", binding.TypeName, binding.Name, string(binding.ConfigData))
		}
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	if err := s.bootstrap(); err != nil {
		return err
	}

	return nil
}

func (s *Server) bootstrap() error {
	log.Info().Msg("Bootstrapping server...")

	// Initialize Gin engine
	engine := gin.New()
	isReleaseMode := gin.Mode() == gin.ReleaseMode
	sessionStore, err := s.createSessionStore(isReleaseMode)
	if err != nil {
		return err
	}

	if isReleaseMode {
		log.Info().Msg("Running in release mode")
		err := engine.SetTrustedProxies(nil)
		if err != nil {
			return err
		}
		engine.Use(gin.ErrorLoggerT(gin.ErrorTypePrivate))
	} else {
		engine.Use(bodyLogMiddleware, gin.ErrorLogger())
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

	log.Info().Msgf("Starting server on %s", s.config.ServerConfig.Address)
	s.listenAndServe()

	return nil
}

func (s *Server) createSessionStore(isReleaseMode bool) (sessions.Store, error) {
	var sessionStore sessions.Store
	secret := s.config.ServerConfig.SessionSecret
	if secret == "" {
		return nil, fmt.Errorf("session secret is not set")
	}
	sessionSecret := []byte(secret)
	if s.config.ServerConfig.RedisSessionStore == nil {
		log.Info().Msg("Using cookies for session storage")
		sessionStore = session.NewCookieStore(isReleaseMode, sessionSecret)
	} else {
		log.Info().Msg("Using Redis for session storage")
		pool := database.NewRedisPoolWithConfig(s.config.ServerConfig.RedisSessionStore)
		s.addShutdownHook(func() error { return pool.Close() })
		var err error
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
	log.Info().Msgf("Shutdown signal received (%s)", <-s.shutdownChannel)
	return s.Shutdown()
}

func (s *Server) listenAndServe() {
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal().Msgf("Listen error: %s", err)
		}
	}()
}

func (s *Server) addShutdownHook(f func() error) {
	s.shutdownHooks = append(s.shutdownHooks, f)
}

// Shutdown gracefully shuts down the server, waiting for active connections to complete
// and freeing up resources. It executes registered shutdown hooks in the process.
func (s *Server) Shutdown() error {
	log.Info().Msg("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := s.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("forced shutdown: %s", err)
	}

	// Free up resources used by controllers
	log.Info().Msg("Executing shutdown hooks...")
	for _, hook := range s.shutdownHooks {
		if err := hook(); err != nil {
			log.Error().Msgf("Error during shutdown hook: %s", err)
		}
	}

	log.Info().Msg("Server exited gracefully")
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

func bodyLogMiddleware(c *gin.Context) {
	blw := &bodyLogWriter{body: bytes.NewBufferString(""), ResponseWriter: c.Writer}
	c.Writer = blw
	c.Next()
	log.Debug().Msgf("Response body: %s", blw.body.String())
}
