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

type Server struct {
	config          *config.Config
	httpServer      *http.Server
	shutdownHooks   []func() error
	shutdownChannel chan os.Signal
}

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

func NewServerFromFlags() *Server {
	debug := flag.Bool("debug", false, "Enable debug mode")
	secretsDir := flag.String("secrets", "", "Path to the secrets directory")
	host := flag.String("host", "localhost", "Host to listen on")
	port := flag.Int("port", 8080, "port to listen on")
	redis := flag.String("redis", "", "Use the specified Redis address as a session store. It expects <host>[:<port>] format and only TCP is currently supported. Defaults to cookie based session storage")
	sessionName := flag.String("cookiename", "sargantana-go", "Session cookie name. Be aware that this name will be used regardless of the session storage type (cookies or Redis)")

	flag.Parse()

	return NewServer(
		*host, *port,
		*redis,
		*sessionName,
		*debug,
		*secretsDir)
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
			return fmt.Errorf("error reading secret file %s: %v\n", name, err)
		}
		err = os.Setenv(strings.ToUpper(name), strings.TrimSpace(string(content)))
		if err != nil {
			return fmt.Errorf("error setting environment variable %s: %v\n", strings.ToUpper(name), err)
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
