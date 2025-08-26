// Package server provides the core HTTP server implementation for the Sargantana Go web framework.
// It handles server lifecycle management, controller registration, session configuration,
// secrets loading, and graceful shutdown functionality.
package server

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/animalet/sargantana-go/config"
	"github.com/animalet/sargantana-go/controller"
	"github.com/animalet/sargantana-go/database"
	"github.com/animalet/sargantana-go/session"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

// Server represents the main HTTP server instance for the Sargantana Go framework.
// It encapsulates the server configuration, HTTP server instance, shutdown hooks,
// and signal handling for graceful shutdown.
type Server struct {
	config          *config.Config
	httpServer      *http.Server
	shutdownHooks   []func() error
	shutdownChannel chan os.Signal
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
}

// parseServerFlags creates and parses all server flags, returning the flagset and parsed values
func parseServerFlags(versionInfo string, flagInitializers ...func(flagSet *flag.FlagSet) func() controller.IController) (*serverFlags, []func() controller.IController) {
	flagSet := flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	// Define all server flags
	flags := &serverFlags{
		debug:       flagSet.Bool("debug", false, "Enable debug mode"),
		secretsDir:  flagSet.String("secrets", "", "Path to the secrets directory"),
		host:        flagSet.String("host", "localhost", "Host to listen on"),
		port:        flagSet.Int("port", 8080, "port to listen on"),
		redis:       flagSet.String("redis", "", "Use the specified Redis address as a session store. It expects <host>[:<port>] format and only TCP is currently supported. Defaults to cookie based session storage"),
		sessionName: flagSet.String("cookiename", "sargantana-go", "Session cookie name. Be aware that this name will be used regardless of the session storage type (cookies or Redis)"),
		showVersion: flagSet.Bool("version", false, "Show version information"),
	}

	// Initialize controller constructors
	var constructors []func() controller.IController
	for _, init := range flagInitializers {
		constructors = append(constructors, init(flagSet))
	}

	// Parse command line arguments
	_ = flagSet.Parse(os.Args[1:])

	// Handle version flag if requested
	if *flags.showVersion {
		fmt.Printf("%s %s\n", "sargantana-go", versionInfo)
		os.Exit(0)
	}

	return flags, constructors
}

// NewServer creates a new Server instance with the specified configuration parameters.
// It initializes the server with basic configuration but does not start it.
//
// Parameters:
//   - host: The hostname or IP address to bind to (e.g., "localhost", "0.0.0.0")
//   - port: The port number to listen on (e.g., 8080)
//   - redis: Redis server address for session storage (empty string uses cookies)
//   - secretsDir: Directory path containing secret files for environment variables
//   - debug: Whether to enable debug mode with detailed logging
//   - sessionName: Name of the session cookie
//
// Returns a pointer to the configured Server instance.
func NewServer(host string, port int, redis, secretsDir string, debug bool, sessionName string) *Server {
	c := config.NewConfig(
		address(host, port),
		redis,
		secretsDir,
		debug,
		sessionName,
	)
	return &Server{config: c}
}

// NewServerFromFlags creates a new Server instance and controllers from command-line flags.
// This is the primary way to initialize a Sargantana Go application with flag-based configuration.
// It registers all controller flags, parses command line arguments, and creates both the server
// and controller instances.
//
// Parameters:
//   - flagInitializers: Variable number of controller factory functions that register their flags
//
// Returns:
//   - *Server: The configured server instance
//   - []controller.IController: List of initialized controllers
func NewServerFromFlags(flagInitializers ...func(flagSet *flag.FlagSet) func() controller.IController) (*Server, []controller.IController) {
	return newServerFromFlagsInternal("unknown", flagInitializers...)
}

// NewServerFromFlagsWithVersion creates a new Server instance and controllers from command-line flags.
// This is the primary way to initialize a Sargantana Go application with flag-based configuration.
// It registers all controller flags, parses command line arguments, and creates both the server
// and controller instances.
//
// Parameters:
//   - versionInfo: Version information to display with --version flag
//   - flagInitializers: Variable number of controller factory functions that register their flags
//
// Returns:
//   - *Server: The configured server instance
//   - []controller.IController: List of initialized controllers
func NewServerFromFlagsWithVersion(versionInfo string, flagInitializers ...func(flagSet *flag.FlagSet) func() controller.IController) (*Server, []controller.IController) {
	return newServerFromFlagsInternal(versionInfo, flagInitializers...)
}

// newServerFromFlagsInternal is the internal implementation that both exported functions use
func newServerFromFlagsInternal(versionInfo string, flagInitializers ...func(flagSet *flag.FlagSet) func() controller.IController) (*Server, []controller.IController) {
	flags, constructors := parseServerFlags(versionInfo, flagInitializers...)

	// Create controller instances
	var controllers []controller.IController
	for _, c := range constructors {
		controllers = append(controllers, c())
	}

	// Create and return server
	return NewServer(
		*flags.host, *flags.port,
		*flags.redis,
		*flags.secretsDir,
		*flags.debug,
		*flags.sessionName), controllers
}

func address(host string, port int) string {
	return host + ":" + strconv.Itoa(port)
}

// StartAndWaitForSignal starts the HTTP server and waits for an OS signal to gracefully shut it down.
// It handles initialization, secret loading, and controller registration before starting the server.
// Upon receiving a termination signal, it gracefully shuts down the server, allowing active connections
// to complete and freeing up resources.
func (s *Server) StartAndWaitForSignal(appControllers ...controller.IController) error {
	err := s.Start(appControllers...)
	if err != nil {
		return err
	}
	return s.waitForSignal()
}

// Start initializes the server components and starts listening for incoming HTTP requests.
// It configures the server based on the provided flags, loads secrets, and sets up the router and middleware.
// This function must be called before the server can handle requests.
func (s *Server) Start(appControllers ...controller.IController) error {
	if s.config.Debug() {
		log.Printf("Debug mode is enabled\n")
		log.Printf("Secrets directory: %q\n", s.config.SecretsDir())
		log.Printf("Listen address: %q\n", s.config.Address())
		if s.config.RedisSessionStore() == "" {
			log.Printf("Use cookies for session storage\n")
		} else {
			log.Printf("Use Redis for session storage: %s\n", s.config.RedisSessionStore())
		}
		log.Printf("Session cookie name: %q\n", s.config.SessionName())
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	if err := s.bootstrap(appControllers...); err != nil {
		return err
	}

	return nil
}

func (s *Server) loadSecrets() error {
	if s.config.SecretsDir() == "" {
		log.Println("No secrets directory configured, skipping secrets loading")
		return nil
	}

	files, err := os.ReadDir(s.config.SecretsDir())
	if err != nil {
		return fmt.Errorf("error reading secrets directory %s: %v", s.config.SecretsDir(), err)
	}
	count := 0
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		name := file.Name()
		content, err := os.ReadFile(filepath.Join(s.config.SecretsDir(), name))
		if err != nil {
			return fmt.Errorf("error reading secret file %s: %v", name, err)
		}
		err = os.Setenv(strings.ToUpper(name), strings.TrimSpace(string(content)))
		if err != nil {
			return fmt.Errorf("error setting environment variable %s: %v", strings.ToUpper(name), err)
		} else {
			count += 1
			if s.config.Debug() {
				log.Printf("Set environment variable from secret %s\n", strings.ToUpper(name))
			}
		}
	}
	if s.config.Debug() {
		log.Printf("Loaded %d secrets from %s\n", count, s.config.SecretsDir())
	}

	return nil
}

func (s *Server) bootstrap(appControllers ...controller.IController) error {
	err := s.loadSecrets()
	if err != nil {
		return err
	}

	engine := gin.New()
	isReleaseMode := gin.Mode() == gin.ReleaseMode
	var sessionStore sessions.Store
	sessionSecret := []byte(os.Getenv("SESSION_SECRET"))
	if s.config.RedisSessionStore() == "" {
		log.Println("Using cookies for session storage")
		sessionStore = session.NewCookieStore(isReleaseMode, sessionSecret)
	} else {
		log.Println("Using Redis for session storage")
		pool := database.NewRedisPool(s.config.RedisSessionStore())
		s.addShutdownHook(func() error { return pool.Close() })
		sessionStore = session.NewRedisSessionStore(isReleaseMode, sessionSecret, pool)
	}

	if isReleaseMode {
		log.Println("Running in release mode")
		err := engine.SetTrustedProxies(nil)
		if err != nil {
			return err
		}
	} else {
		engine.Use(ginBodyLogMiddleware)
	}
	engine.Use(gin.Logger(), gin.Recovery())
	engine.Use(sessions.Sessions(s.config.SessionName(), sessionStore))
	for _, c := range appControllers {
		c.Bind(engine, *s.config, controller.LoginFunc)
		s.addShutdownHook(c.Close)
	}

	s.httpServer = &http.Server{
		Addr:    s.config.Address(),
		Handler: engine,
	}

	log.Println("Starting server on " + s.config.Address())
	s.listenAndServe()

	return nil
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
