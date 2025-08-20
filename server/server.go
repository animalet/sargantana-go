package server

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/animalet/sargantana-go/controller"
	"github.com/animalet/sargantana-go/database"
	"github.com/animalet/sargantana-go/session"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

type config struct {
	address               string
	redisSessionStore     string
	secretsDir            string
	frontendDir           string
	templatesDir          string
	debug                 bool
	loadBalancerPath      string
	loadBalancerAuth      bool
	loadBalancerEndpoints []url.URL
	callbackEndpoint      string
	sessionName           string
}
type Server struct {
	config          *config
	httpServer      *http.Server
	shutdownHooks   []func() error
	shutdownChannel chan os.Signal
}

func NewServer(
	host string,
	port int,
	redis string,
	secretsDir, frontendDir, templatesDir string,
	debug bool,
	lbPath string,
	lbAuth bool,
	lbEndpoints []url.URL,
	sessionName string,
	callbackEndpoint string,
) *Server {
	c := &config{
		address:               address(host, port),
		redisSessionStore:     redis,
		secretsDir:            secretsDir,
		frontendDir:           frontendDir,
		templatesDir:          templatesDir,
		debug:                 debug,
		loadBalancerPath:      lbPath,
		loadBalancerAuth:      lbAuth,
		loadBalancerEndpoints: lbEndpoints,
		sessionName:           sessionName,
		callbackEndpoint:      callbackEndpoint,
	}
	return &Server{config: c}
}

func NewServerFromFlags() *Server {
	debug := flag.Bool("debug", false, "Enable debug mode")
	secretsDir := flag.String("secrets", "", "Path to the secrets directory")
	frontend := flag.String("frontend", "./frontend", "Path to the frontend static content directory")
	templates := flag.String("templates", "./templates", "Path to the frontend static content directory")
	host := flag.String("host", "localhost", "Host to listen on")
	port := flag.Int("port", 8080, "port to listen on")
	redis := flag.String("redis", "", "Use the specified Redis address as a session store. It expects <host>[:<port>] format and only TCP is currently supported. Defaults to cookie based session storage")
	sessionName := flag.String("cookiename", "sargantana-go", "Session cookie name. Be aware that this name will be used regardless of the session storage type (cookies or Redis)")
	lbPath := flag.String("lbpath", "lb", "Path to use for load balancing")
	lbAuth := flag.Bool("lbauth", false, "Use authentication for load balancing")
	callback := flag.String("callback", "", "Callback endpoint for authentication, in case you are behind a reverse proxy or load balancer. If not set, it will default to http://<host>:<port>")

	lbEndpoints := make([]url.URL, 0)
	flag.Func("lb", "Path to use for load balancing", func(s string) error {
		u, err := url.Parse(s)
		if err != nil {
			return err
		}
		lbEndpoints = append(lbEndpoints, *u)
		return nil
	})

	flag.Parse()

	c := &config{
		address:               address(*host, *port),
		redisSessionStore:     *redis,
		sessionName:           *sessionName,
		secretsDir:            *secretsDir,
		frontendDir:           *frontend,
		templatesDir:          *templates,
		debug:                 *debug,
		loadBalancerPath:      *lbPath,
		loadBalancerAuth:      *lbAuth,
		loadBalancerEndpoints: lbEndpoints,
		callbackEndpoint:      *callback,
	}
	return &Server{config: c}
}

func address(host string, port int) string {
	return host + ":" + strconv.Itoa(port)
}

func (s *Server) StartAndWaitForSignal(appControllers ...controller.IController) error {
	err := s.Start(appControllers...)
	if err != nil {
		return err
	}
	return s.waitForSignal()
}

func (s *Server) Start(appControllers ...controller.IController) error {
	if s.config.debug {
		log.Printf("Debug mode is enabled\n")
		log.Printf("Secrets directory: %q\n", s.config.secretsDir)
		log.Printf("Frontend directory: %q\n", s.config.frontendDir)
		log.Printf("Templates directory: %q\n", s.config.templatesDir)
		log.Printf("Listen address: %q\n", s.config.address)
		if s.config.redisSessionStore == "" {
			log.Printf("Use cookies for session storage\n")
		} else {
			log.Printf("Use Redis for session storage: %s\n", s.config.redisSessionStore)
		}
		log.Printf("Session cookie name: %q\n", s.config.sessionName)
		log.Printf("Load balancing path: %q\n", s.config.loadBalancerPath)
		log.Printf("Load balancing authentication: %t\n", s.config.loadBalancerAuth)
		log.Printf("Load balancer endpoints: %v\n", s.config.loadBalancerEndpoints)
		log.Printf("Callback endpoint: %q\n", s.config.callbackEndpoint)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	if err := s.bootstrap(appControllers...); err != nil {
		return err
	}

	return nil
}

func (s *Server) loadSecrets() error {
	if s.config.secretsDir == "" {
		log.Println("No secrets directory configured, skipping secrets loading")
		return nil
	}

	files, err := os.ReadDir(s.config.secretsDir)
	if err != nil {
		return fmt.Errorf("error reading secrets directory %s: %v", s.config.secretsDir, err)
	}
	count := 0
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		name := file.Name()
		content, err := os.ReadFile(filepath.Join(s.config.secretsDir, name))
		if err != nil {
			return fmt.Errorf("error reading secret file %s: %v\n", name, err)
		}
		err = os.Setenv(strings.ToUpper(name), strings.TrimSpace(string(content)))
		if err != nil {
			return fmt.Errorf("error setting environment variable %s: %v\n", strings.ToUpper(name), err)
		} else {
			count += 1
			if s.config.debug {
				log.Printf("Set environment variable from secret %s\n", strings.ToUpper(name))
			}
		}
	}
	if s.config.debug {
		log.Printf("Loaded %d secrets from %s\n", count, s.config.secretsDir)
	}

	return nil
}

func (s *Server) bootstrap(appControllers ...controller.IController) error {
	err := s.loadSecrets()
	if err != nil {
		return err
	}

	if s.config.callbackEndpoint == "" {
		s.config.callbackEndpoint = "http://" + s.config.address
		if u, err := url.Parse(s.config.callbackEndpoint); err != nil {
			return fmt.Errorf("invalid callback endpoint %s: %v", s.config.callbackEndpoint, err)
		} else {
			if u.Hostname() == "0.0.0.0" {
				log.Println("Callback endpoint is set to 0.0.0.0, changing it to localhost")
				s.config.callbackEndpoint = u.Scheme + "://localhost" + ":" + u.Port()
			}
		}
	}

	controllers := []controller.IController{
		controller.NewStatic(s.config.frontendDir, s.config.templatesDir),
		controller.NewAuth(s.config.callbackEndpoint),
		controller.NewLoadBalancer(s.config.loadBalancerEndpoints, s.config.loadBalancerPath, s.config.loadBalancerAuth),
	}
	controllers = append(controllers, appControllers...)

	engine := gin.New()
	isReleaseMode := gin.Mode() == gin.ReleaseMode
	var sessionStore sessions.Store
	sessionSecret := []byte(os.Getenv("SESSION_SECRET"))
	if s.config.redisSessionStore == "" {
		log.Println("Using cookies for session storage")
		sessionStore = session.NewCookieStore(isReleaseMode, sessionSecret)
	} else {
		log.Println("Using Redis for session storage")
		pool := database.NewRedisPool(s.config.redisSessionStore)
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
	engine.Use(sessions.Sessions(s.config.sessionName, sessionStore))

	for _, c := range controllers {
		c.Bind(engine, controller.LoginFunc)
	}

	s.httpServer = &http.Server{
		Addr:    s.config.address,
		Handler: engine,
	}

	for _, c := range controllers {
		s.addShutdownHook(c.Close)
	}
	log.Println("Starting server on " + s.config.address)
	s.listenAndServe()
	return nil
}

func (s *Server) waitForSignal() error {
	s.shutdownChannel = make(chan os.Signal, 1)
	// kill (no params) by default sends syscall.SIGTERM
	// kill -2 is syscall.SIGINT
	// kill -9 is syscall.SIGKILL but can't be caught, so don't need add it
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
