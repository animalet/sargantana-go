package controller

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/animalet/sargantana-go/pkg/server"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

type LoadBalancerControllerConfig struct {
	Auth      bool     `yaml:"auth"`
	Path      string   `yaml:"path"`
	Endpoints []string `yaml:"endpoints"`
}

func (l LoadBalancerControllerConfig) Validate() error {
	if len(l.Endpoints) == 0 {
		return errors.New("at least one endpoint must be provided")
	}

	if l.Path == "" {
		return errors.New("path must be set and non-empty")
	}

	for _, endpoint := range l.Endpoints {
		if _, err := url.ParseRequestURI(endpoint); err != nil {
			return errors.Wrap(err, fmt.Sprintf("invalid endpoint URL: %s", endpoint))
		}
	}
	return nil
}

func NewLoadBalancerController(c *LoadBalancerControllerConfig, _ server.ControllerContext) (server.IController, error) {
	stringEndpoints := c.Endpoints
	endpoints := make([]url.URL, 0, len(stringEndpoints))
	log.Info().Str("path", c.Path).Msg("Load balancing path configured")
	log.Info().Bool("auth", c.Auth).Msg("Load balancing authentication configured")
	log.Info().Strs("endpoints", stringEndpoints).Msg("Load balancing endpoints configured")
	for _, endpoint := range stringEndpoints {
		u, err := url.ParseRequestURI(endpoint)
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("failed to parse load balancer path: %s", c.Path))
		}
		endpoints = append(endpoints, *u)
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
		auth:       c.Auth,
	}, nil
}

// loadBalancer is a controller that provides round-robin load balancing functionality.
// It distributes incoming requests across multiple backend endpoints and supports
// optional authentication requirements for protected load-balanced routes.
type loadBalancer struct {
	server.IController
	endpoints     []url.URL
	endpointIndex int
	mu            sync.Mutex
	httpClient    *http.Client
	path          string
	auth          bool
}

func (l *loadBalancer) Bind(engine *gin.Engine) {
	if len(l.endpoints) == 0 {
		log.Warn().Msg("Load balancer not loaded: no endpoints configured")
		return
	}

	if l.auth {
		engine.GET(l.path, LoginFunc, l.forward).
			POST(l.path, LoginFunc, l.forward).
			PUT(l.path, LoginFunc, l.forward).
			DELETE(l.path, LoginFunc, l.forward).
			PATCH(l.path, LoginFunc, l.forward).
			HEAD(l.path, LoginFunc, l.forward).
			OPTIONS(l.path, LoginFunc, l.forward)
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
			log.Error().Err(err).Msg("Error closing response body")
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

	_, err = io.Copy(c.Writer, response.Body)
	if err != nil {
		log.Error().Err(err).Msg("Error copying response body")
	}
}
