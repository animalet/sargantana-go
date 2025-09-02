package controller

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/animalet/sargantana-go/config"
	"github.com/animalet/sargantana-go/logger"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

type LoadBalancerControllerConfig struct {
	Auth      bool     `yaml:"auth"`
	Path      string   `yaml:"path"`
	Endpoints []string `yaml:"endpoints"`
}

func NewLoadBalancerController(configData config.ControllerConfig, _ config.ServerConfig) (IController, error) {
	c, err := config.UnmarshalTo[LoadBalancerControllerConfig](configData)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse load balancer controller config")
	}
	auth := c.Auth
	stringEndpoints := c.Endpoints
	endpoints := make([]url.URL, 0, len(stringEndpoints))
	if len(stringEndpoints) == 0 {
		return nil, errors.New("no endpoints provided for load balancing")
	} else {
		logger.Infof("Load balancing path: %q", c.Path)
		logger.Infof("Load balancing authentication: %t", auth)
		logger.Infof("Load balanced endpoints:")
		for _, endpoint := range stringEndpoints {
			u, err := url.Parse(endpoint)
			if err != nil {
				return nil, errors.Wrap(err, fmt.Sprintf("failed to parse load balancer path: %s", c.Path))
			}
			endpoints = append(endpoints, *u)
			logger.Infof(" - %s", u.String())
		}
	}

	httpClient := &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
		},
	}

	return &loadBalancer{
		endpoints:  endpoints,
		httpClient: httpClient,
		path:       strings.TrimSuffix(c.Path, "/") + "/*proxyPath",
		auth:       auth,
	}, nil
}

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

func (l *loadBalancer) Bind(engine *gin.Engine, loginMiddleware gin.HandlerFunc) {
	if len(l.endpoints) == 0 {
		logger.Warn("Load balancer not loaded")
		return
	}

	if l.auth {
		engine.GET(l.path, loginMiddleware, l.forward).
			POST(l.path, loginMiddleware, l.forward).
			PUT(l.path, loginMiddleware, l.forward).
			DELETE(l.path, loginMiddleware, l.forward).
			PATCH(l.path, loginMiddleware, l.forward).
			HEAD(l.path, loginMiddleware, l.forward).
			OPTIONS(l.path, loginMiddleware, l.forward)
	} else {
		engine.GET(l.path, l.forward).
			POST(l.path, l.forward).
			PUT(l.path, l.forward).
			DELETE(l.path, l.forward).
			PATCH(l.path, l.forward).
			HEAD(l.path, l.forward).
			OPTIONS(l.path, l.forward)
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

func (l *loadBalancer) forward(c *gin.Context) {
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
			logger.Errorf("Error closing response body: %v", err)
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
