package server

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/animalet/sargantana-go/controller"
	"github.com/animalet/sargantana-go/database"
	"github.com/animalet/sargantana-go/session"
	"github.com/fvbock/endless"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

type Server struct {
	host                  string
	port                  int
	cookieBasedSession    bool
	secretsDir            string
	frontendDir           string
	templatesDir          string
	debug                 bool
	loadBalancerPath      string
	loadBalancerAuth      bool
	loadBalancerEndpoints []url.URL
}

func NewServer(
	host string,
	port int,
	cookieBasedSession bool,
	secretsDir, frontendDir, templatesDir string,
	debug bool,
	lbPath string,
	lbAuth bool,
	lbEndpoints []url.URL,
) *Server {
	return &Server{
		host:                  host,
		port:                  port,
		cookieBasedSession:    cookieBasedSession,
		secretsDir:            secretsDir,
		frontendDir:           frontendDir,
		templatesDir:          templatesDir,
		debug:                 debug,
		loadBalancerPath:      lbPath,
		loadBalancerAuth:      lbAuth,
		loadBalancerEndpoints: lbEndpoints,
	}
}

func NewServerFromFlags() *Server {
	debug := flag.Bool("debug", false, "Enable debug mode")
	secretsDir := flag.String("secrets", "/run/secrets", "Path to the secrets directory")
	frontend := flag.String("frontend", "./frontend", "Path to the frontend static content directory")
	templates := flag.String("templates", "./templates", "Path to the frontend static content directory")
	host := flag.String("host", "localhost", "Host to listen on")
	port := flag.Int("port", 8080, "port to listen on")
	useCookies := flag.Bool("cookies", false, "Use cookies for session storage")
	lbPath := flag.String("lbpath", "lb", "Path to use for load balancing")
	lbAuth := flag.Bool("lbauth", false, "Use authentication for load balancing")

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

	return &Server{
		host:                  *host,
		port:                  *port,
		cookieBasedSession:    *useCookies,
		secretsDir:            *secretsDir,
		frontendDir:           *frontend,
		templatesDir:          *templates,
		debug:                 *debug,
		loadBalancerPath:      *lbPath,
		loadBalancerAuth:      *lbAuth,
		loadBalancerEndpoints: lbEndpoints,
	}
}
func (s *Server) Start(appControllers ...controller.IController) error {
	if s.debug {
		log.Printf("Debug mode is enabled\n")
		log.Printf("Secrets loaded from %s\n", s.secretsDir)
		log.Printf("Frontend directory: %s\n", s.frontendDir)
		log.Printf("Templates directory: %s\n", s.templatesDir)
		log.Printf("Host: %s\n", s.host)
		log.Printf("Port: %d\n", s.port)
		log.Printf("Use cookies for session storage: %t\n", s.cookieBasedSession)
		log.Printf("Load balancing path: %s\n", s.loadBalancerPath)
		log.Printf("Load balancing authentication: %t\n", s.loadBalancerAuth)
		log.Printf("Load balancer endpoints: %v\n", s.loadBalancerEndpoints)
	} else {
		os.Setenv("GIN_MODE", "release")
	}

	err := s.loadSecrets()
	if err != nil {
		return err
	}

	callbackEndpoint := os.Getenv("OAUTH_CALLBACK_ENDPOINT") // This covers the case of being behind a reverse proxy
	if callbackEndpoint == "" {
		callbackEndpoint = "http://" + s.host + ":" + strconv.Itoa(s.port)
	}

	controllers := []controller.IController{
		controller.NewStatic(s.frontendDir, s.templatesDir),
		controller.NewAuth(callbackEndpoint),
		controller.NewLoadBalancer(s.loadBalancerEndpoints, s.loadBalancerPath, s.loadBalancerAuth),
	}
	controllers = append(controllers, appControllers...)
	if err = s.doStart(controllers...); err != nil {
		return err
	}

	return nil
}

func (s *Server) loadSecrets() error {
	files, err := os.ReadDir(s.secretsDir)
	if err != nil {
		return fmt.Errorf("error reading secrets directory %s: %v", s.secretsDir, err)
	}
	count := 0
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		name := file.Name()
		content, err := os.ReadFile(filepath.Join(s.secretsDir, name))
		if err != nil {
			return fmt.Errorf("error reading secret file %s: %v\n", name, err)
		}
		err = os.Setenv(strings.ToUpper(name), strings.TrimSpace(string(content)))
		if err != nil {
			return fmt.Errorf("error setting environment variable %s: %v\n", strings.ToUpper(name), err)
		} else {
			count += 1
			if s.debug {
				log.Printf("Set environment variable from secret %s\n", strings.ToUpper(name))
			}
		}
	}
	if s.debug {
		log.Printf("Loaded %d secrets from %s\n", count, s.secretsDir)
	}

	return nil
}

func (s *Server) doStart(controllers ...controller.IController) error {
	engine := gin.New()
	isReleaseMode := gin.Mode() == gin.ReleaseMode

	var sessionStore sessions.Store
	sessionSecret := []byte(os.Getenv("SESSION_SECRET"))
	if s.cookieBasedSession {
		log.Println("Using cookies for session storage")
		sessionStore = session.NewCookieStore(isReleaseMode, sessionSecret)
	} else {
		log.Println("Using Redis for session storage")
		pool := database.NewRedisPool()
		defer pool.Close()
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
	engine.Use(sessions.Sessions("gene", sessionStore))

	for _, c := range controllers {
		c.Bind(engine, controller.LoginFunc)
	}

	addr := s.host + ":" + strconv.Itoa(s.port)
	err := endless.ListenAndServe(addr, engine)

	for _, c := range controllers {
		c.Close()
	}

	if errors.Is(err, http.ErrServerClosed) {
		log.Println("Server closed.")
		return nil
	} else {
		return err
	}
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
