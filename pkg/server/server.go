package server

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/animalet/sargantana-go/pkg/config"
	"github.com/animalet/sargantana-go/pkg/server/session"
	"github.com/gin-contrib/secure"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type WebServerConfig struct {
	Address       string          `yaml:"address"`
	SessionName   string          `yaml:"session_name"`
	SessionSecret string          `yaml:"session_secret"`
	Security      *SecurityConfig `yaml:"security,omitempty"`
}

func (c WebServerConfig) Validate() error {
	if c.SessionSecret == "" {
		return errors.New("session_secret must be set and non-empty")
	}

	if c.SessionName == "" {
		return errors.New("session_name must be set and non-empty")
	}

	if c.Address == "" {
		return errors.New("address must be set and non-empty")
	}

	_, err := net.ResolveTCPAddr("tcp", c.Address)
	if err != nil {
		return fmt.Errorf("invalid address: %w", err)
	}

	if c.Security != nil {
		if err := c.Security.Validate(); err != nil {
			return fmt.Errorf("invalid security configuration: %w", err)
		}
	}

	return nil
}

type SecurityConfig struct {
	SSLRedirect               bool              `yaml:"ssl_redirect"`
	SSLTemporaryRedirect      bool              `yaml:"ssl_temporary_redirect"`
	SSLHost                   string            `yaml:"ssl_host"`
	SSLProxyHeaders           map[string]string `yaml:"ssl_proxy_headers"`
	STSSeconds                int64             `yaml:"sts_seconds"`
	STSIncludeSubdomains      bool              `yaml:"sts_include_subdomains"`
	FrameDeny                 bool              `yaml:"frame_deny"`
	CustomFrameOptionsValue   string            `yaml:"custom_frame_options_value"`
	ContentTypeNosniff        bool              `yaml:"content_type_nosniff"`
	BrowserXssFilter          bool              `yaml:"browser_xss_filter"`
	ContentSecurityPolicy     string            `yaml:"content_security_policy"`
	ReferrerPolicy            string            `yaml:"referrer_policy"`
	FeaturePolicy             string            `yaml:"feature_policy"`
	PermissionsPolicy         string            `yaml:"permissions_policy"` // Deprecates FeaturePolicy
	IENoOpen                  bool              `yaml:"ie_no_open"`
	IsDevelopment             bool              `yaml:"is_development"`
	DontRedirectIPV4Hostnames bool              `yaml:"dont_redirect_ipv4_hostnames"`
}

func (c SecurityConfig) Validate() error {
	// Most fields are optional booleans or strings, so basic validation might be minimal.
	// We can add specific checks if needed, e.g., if SSLRedirect is true, maybe check SSLHost?
	// For now, we'll assume gin-contrib/secure handles empty values gracefully (which it does).
	return nil
}

// SargantanaConfig holds the configuration settings for the Sargantana Go server.
type SargantanaConfig struct {
	WebServerConfig    WebServerConfig    `yaml:"server"`
	ControllerBindings ControllerBindings `yaml:"controllers"`
}

func (c SargantanaConfig) Validate() error {
	if err := c.WebServerConfig.Validate(); err != nil {
		return errors.Wrap(err, "server configuration is invalid")
	}

	return c.ControllerBindings.Validate()
}

// Server represents the main HTTP server instance for the Sargantana Go framework.
type Server struct {
	config          SargantanaConfig
	httpServer      *http.Server
	shutdownHooks   []func() error
	shutdownChannel chan os.Signal
	sessionStore    sessions.Store
	authenticator   Authenticator
}

// controllerRegistry holds the mapping of controller type names to their factory functions.
var controllerRegistry = make(map[string]ControllerFactory)

var debug = false

func SetDebug(debugEnabled bool) {
	debug = debugEnabled
	if debugEnabled {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
		gin.SetMode(gin.DebugMode)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
		gin.SetMode(gin.ReleaseMode)
	}
}

func NewServer(cfg SargantanaConfig) *Server {
	return &Server{
		config:        cfg,
		authenticator: NewUnauthorizedAuthenticator(),
	}
}

func (s *Server) SetSessionStore(sessionStore sessions.Store) {
	s.sessionStore = sessionStore
}

// SetAuthenticator configures the authentication provider for the server.
// This allows developers to customize how authentication is handled throughout
// the application. The provided authenticator will be passed to all controllers
// that need authentication middleware.
//
// By default, the server uses UnauthorizedAuthenticator which rejects all
// authenticated requests. To enable authentication, set a proper authenticator:
//
//	server.SetAuthenticator(controller.NewGothAuthenticator())
//
// or provide a custom implementation of the Authenticator interface.
func (s *Server) SetAuthenticator(authenticator Authenticator) {
	s.authenticator = authenticator
}

func addControllerType(typeName string, factory ControllerFactory) {
	log.Info().Msgf("Registering controller type %q", typeName)
	_, exists := controllerRegistry[typeName]
	if exists {
		log.Warn().Msgf("Controller type %q is already registered, overriding", typeName)
	}
	controllerRegistry[typeName] = factory
}

// RegisterController registers a controller factory that takes a typed configuration.
// T must implement config.Validatable.
func RegisterController[T config.Validatable](typeName string, factory func(cfg *T, ctx ControllerContext) (IController, error)) {
	addControllerType(typeName, func(raw config.ModuleRawConfig, ctx ControllerContext) (IController, error) {
		cfg, err := config.Unmarshal[T](raw)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal configuration for controller type %s", typeName)
		}
		return factory(cfg, ctx)
	})
}

func configureControllers(c SargantanaConfig, sessionStore sessions.Store) (controllers []IController, configErrors []error) {
	instanceCounts := make(map[string]int) // Track instances per type for auto-naming

	// Build the controller context with runtime dependencies
	ctx := ControllerContext{
		ServerConfig: c.WebServerConfig,
		SessionStore: sessionStore,
	}

	for _, binding := range c.ControllerBindings {
		// Generate instance name
		instanceName := binding.Name
		if instanceName == "" {
			instanceCounts[binding.TypeName]++
			count := instanceCounts[binding.TypeName]
			if count == 1 {
				instanceName = binding.TypeName
			} else {
				instanceName = fmt.Sprintf("%s-%d", binding.TypeName, count)
			}
		}

		factory, exists := controllerRegistry[binding.TypeName]
		if !exists {
			configErrors = append(configErrors, fmt.Errorf("no factory found for controller type %q (instance: %q)", binding.TypeName, instanceName))
			continue
		}

		newController, err := newController(ctx, instanceName, binding, factory)
		if err == nil {
			controllers = append(controllers, newController)
		} else {
			configErrors = append(configErrors, fmt.Errorf("error configuring controller %q of type %q: %v", instanceName, binding.TypeName, err))
		}
	}
	return controllers, configErrors
}

func newController(ctx ControllerContext, name string, binding ControllerBinding, factory ControllerFactory) (newController IController, err error) {
	log.Info().Msgf("Configuring %s controller of type: %s", name, binding.TypeName)
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic during %s controller configuration, controller was not added: %v", name, r)
			newController = nil
		}
	}()
	if newController, err = factory(binding.Config, ctx); err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("failed to configure %s controller of type: %s", name, binding.TypeName))
	}
	return newController, err
}

func (s *Server) StartAndWaitForSignal() error {
	err := s.Start()
	if err != nil {
		return err
	}
	return s.waitForSignal()
}

func (s *Server) Start() (err error) {
	if debug {
		log.Debug().Msg("Debug mode is enabled")
		log.Debug().Msgf("Listen address: %q", s.config.WebServerConfig.Address)
		log.Debug().Msgf("Session cookie name: %q", s.config.WebServerConfig.SessionName)
		log.Debug().Msg("Expected controllers:")
		for _, binding := range s.config.ControllerBindings {
			log.Debug().Msgf(" - Type: %s, Name: %s\n%s", binding.TypeName, binding.Name, binding.Config)
		}
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	if s.sessionStore == nil {
		log.Debug().Msg("No session store provided, creating one based on configuration")
		s.sessionStore, err = s.createSessionStore(!debug)
		if err != nil {
			return err
		}
	} else {
		log.Debug().Msgf("Using provided session store: %T", s.sessionStore)
	}
	if err := s.bootstrap(); err != nil {
		return err
	}

	s.listenAndServe()

	return nil
}

func (s *Server) bootstrap() error {
	log.Info().Msg("Bootstrapping server...")

	// Configure controllers with session store now that it's available
	controllers, configurationErrors := configureControllers(s.config, s.sessionStore)
	if len(configurationErrors) > 0 {
		log.Error().Msg("Configuration errors encountered, affected controllers have been excluded from bootstrap:")
		for _, configErr := range configurationErrors {
			log.Error().Msgf(" - %v", configErr)
		}
	}

	// Initialize Gin engine
	gin.ForceConsoleColor()
	engine := gin.New()
	if gin.IsDebugging() {
		log.Info().Msg("Running in debug mode")
		engine.Use(bodyLogMiddleware, gin.ErrorLogger())
	} else {
		log.Info().Msg("Running in release mode")
		err := engine.SetTrustedProxies(nil)
		if err != nil {
			return err
		}
		engine.Use(gin.ErrorLoggerT(gin.ErrorTypePrivate))
	}
	engine.Use(
		gin.Logger(),
		gin.Recovery(),
		sessions.Sessions(s.config.WebServerConfig.SessionName, s.sessionStore),
	)

	if s.config.WebServerConfig.Security != nil {
		log.Info().Msg("Applying security middleware")
		secConfig := secure.Config{
			SSLRedirect:               s.config.WebServerConfig.Security.SSLRedirect,
			SSLTemporaryRedirect:      s.config.WebServerConfig.Security.SSLTemporaryRedirect,
			SSLHost:                   s.config.WebServerConfig.Security.SSLHost,
			SSLProxyHeaders:           s.config.WebServerConfig.Security.SSLProxyHeaders,
			STSSeconds:                s.config.WebServerConfig.Security.STSSeconds,
			STSIncludeSubdomains:      s.config.WebServerConfig.Security.STSIncludeSubdomains,
			FrameDeny:                 s.config.WebServerConfig.Security.FrameDeny,
			CustomFrameOptionsValue:   s.config.WebServerConfig.Security.CustomFrameOptionsValue,
			ContentTypeNosniff:        s.config.WebServerConfig.Security.ContentTypeNosniff,
			BrowserXssFilter:          s.config.WebServerConfig.Security.BrowserXssFilter,
			ContentSecurityPolicy:     s.config.WebServerConfig.Security.ContentSecurityPolicy,
			ReferrerPolicy:            s.config.WebServerConfig.Security.ReferrerPolicy,
			FeaturePolicy:             s.config.WebServerConfig.Security.FeaturePolicy,
			IENoOpen:                  s.config.WebServerConfig.Security.IENoOpen,
			IsDevelopment:             s.config.WebServerConfig.Security.IsDevelopment,
			DontRedirectIPV4Hostnames: s.config.WebServerConfig.Security.DontRedirectIPV4Hostnames,
		}
		engine.Use(secure.New(secConfig))
	}

	for _, c := range controllers {
		c.Bind(engine, s.authenticator.Middleware())
		s.addShutdownHook(c.Close)
	}

	s.httpServer = &http.Server{
		Addr:              s.config.WebServerConfig.Address,
		Handler:           engine,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	log.Info().Msgf("Starting server on %s", s.config.WebServerConfig.Address)
	// listenAndServe is now called by Start()

	return nil
}

func (s *Server) createSessionStore(isReleaseMode bool) (sessions.Store, error) {
	secret := s.config.WebServerConfig.SessionSecret
	if secret == "" {
		return nil, fmt.Errorf("session secret is not set")
	}
	sessionSecret := []byte(secret)

	// Default to cookie-based session storage
	// For Redis or other session stores, use SetSessionStore() before calling Start()
	log.Info().Msg("Using default cookie-based session storage")
	sessionStore := session.NewCookieStore(isReleaseMode, sessionSecret)
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

func (s *Server) Shutdown() error {
	log.Info().Msg("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := s.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("forced shutdown: %s", err)
	}

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
