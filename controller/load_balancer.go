package controller

import (
	"flag"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
)

// loadBalancer is a controller that provides round-robin load balancing functionality.
// It distributes incoming requests across multiple backend endpoints and supports
// optional authentication requirements for protected load-balanced routes.
type loadBalancer struct {
	IController
	endpoints     []url.URL
	endpointIndex int
	mu            sync.Mutex
	httpClient    *http.Client
	path          string
	auth          bool
}

// NewLoadBalancer creates a new loadBalancer controller with the specified configuration.
// It sets up round-robin load balancing across the provided endpoints and configures
// an optimized HTTP client for backend communication.
//
// Parameters:
//   - endpoints: List of backend server URLs to load balance across
//   - path: URL path where the load balancer will be accessible (e.g., "api" for /api/*)
//   - auth: Whether authentication is required to access load-balanced routes
//
// Returns a pointer to the configured loadBalancer controller.
func NewLoadBalancer(endpoints []url.URL, path string, auth bool) IController {
	if len(endpoints) == 0 {
		log.Printf("No endpoints provided for load balancing")
	} else {
		log.Printf("Load balanced endpoints:")
		for _, endpoint := range endpoints {
			log.Printf("%v", endpoint.String())
		}
	}

	log.Printf("Load balancing path: %q\n", path)
	log.Printf("Load balancing authentication: %t\n", auth)
	log.Printf("Load balancer endpoints: %v\n", endpoints)

	httpClient := &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
		},
	}

	return &loadBalancer{
		endpoints:  endpoints,
		httpClient: httpClient,
		path:       path,
		auth:       auth,
	}
}

// NewLoadBalancerFromFlags creates a loadBalancer controller factory function that reads
// configuration from command-line flags. This function is designed to be used
// with the server's flag-based initialization system.
//
// The following flags are registered:
//   - lbpath: Path to use for load balancing (default: "lb")
//   - lbauth: Use authentication for load balancing (default: false)
//   - lb: Backend endpoint URLs (can be specified multiple times)
//
// Parameters:
//   - flagSet: The flag set to register the load balancer flags with
//
// Returns a factory function that creates a loadBalancer controller when called.
func NewLoadBalancerFromFlags(flagSet *flag.FlagSet) func() IController {
	lbPath := flagSet.String("lbpath", "lb", "Path to use for load balancing")
	lbAuth := flagSet.Bool("lbauth", false, "Use authentication for load balancing")

	lbEndpoints := make([]url.URL, 0)
	flagSet.Func("lb", "Path to use for load balancing", func(s string) error {
		u, err := url.Parse(s)
		if err != nil {
			return err
		}
		lbEndpoints = append(lbEndpoints, *u)
		return nil
	})
	return func() IController { return NewLoadBalancer(lbEndpoints, *lbPath, *lbAuth) }
}

func (l *loadBalancer) Bind(engine *gin.Engine, loginMiddleware gin.HandlerFunc) {
	if len(l.endpoints) == 0 {
		log.Printf("Load balancer not loaded")
		return
	}

	if l.auth {
		engine.Any(l.path, loginMiddleware, l.forward)
	} else {
		engine.Any(l.path, l.forward)
	}
}

func (l *loadBalancer) Close() error {
	return nil
}

func (l *loadBalancer) nextEndpoint() url.URL {
	l.mu.Lock()
	defer func() {
		l.endpointIndex = (l.endpointIndex + 1) % len(l.endpoints)
		l.mu.Unlock()
	}()
	return l.endpoints[l.endpointIndex]
}

var allowedMethods = map[string]bool{"GET": true, "POST": true, "PUT": true, "DELETE": true, "PATCH": true, "HEAD": true, "OPTIONS": true}

func (l *loadBalancer) forward(c *gin.Context) {
	// Only allow safe HTTP methods
	if !allowedMethods[c.Request.Method] {
		c.AbortWithStatus(http.StatusMethodNotAllowed)
		return
	}

	endpoint := l.nextEndpoint()
	// Build the target URL using only path and raw query
	targetUrl := url.URL{
		Scheme:   endpoint.Scheme,
		Host:     endpoint.Host,
		Path:     c.Request.URL.Path,
		RawQuery: c.Request.URL.RawQuery,
	}

	// Create the new request
	request, err := http.NewRequest(c.Request.Method, targetUrl.String(), c.Request.Body)
	if err != nil {
		_ = c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	// Filter headers to avoid leaking sensitive data
	for k, v := range c.Request.Header {
		// Skip Host, X-Forwarded-For, Authorization, Cookie, etc.
		if strings.EqualFold(k, "Host") || strings.HasPrefix(strings.ToLower(k), "x-forwarded-") || strings.EqualFold(k, "Authorization") || strings.EqualFold(k, "Cookie") {
			continue
		}
		for _, vv := range v {
			request.Header.Add(k, vv)
		}
	}

	request.Header.Set("X-Forwarded-For", c.ClientIP())

	response, err := l.httpClient.Do(request)
	if err != nil {
		_ = c.AbortWithError(http.StatusBadGateway, err)
		return
	}
	defer func() {
		err = response.Body.Close()
		if err != nil {
			log.Printf("Error closing response body: %v", err)
		}
	}()

	c.Status(response.StatusCode)
	for k, v := range response.Header {
		if strings.EqualFold(k, "Set-Cookie") {
			continue // avoid leaking backend cookies
		}
		for _, vv := range v {
			c.Writer.Header().Add(k, vv)
		}
	}
	// Copy response body
	if _, err := io.Copy(c.Writer, response.Body); err != nil {
		_ = c.AbortWithError(http.StatusInternalServerError, err)
	}
}
