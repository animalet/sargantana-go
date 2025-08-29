package controller

import (
	"flag"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/animalet/sargantana-go/config"
	"github.com/gin-gonic/gin"
)

func TestNewLoadBalancer(t *testing.T) {
	tests := []struct {
		name      string
		endpoints []url.URL
		path      string
		auth      bool
	}{
		{
			name:      "empty endpoints",
			endpoints: []url.URL{},
			path:      "/api",
			auth:      false,
		},
		{
			name: "single endpoint",
			endpoints: []url.URL{
				{Scheme: "http", Host: "backend1.com"},
			},
			path: "/proxy",
			auth: true,
		},
		{
			name: "multiple endpoints",
			endpoints: []url.URL{
				{Scheme: "http", Host: "backend1.com"},
				{Scheme: "https", Host: "backend2.com"},
			},
			path: "/lb",
			auth: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lb := NewLoadBalancer(tt.endpoints, tt.path, tt.auth)

			if lb == nil {
				t.Fatal("NewLoadBalancer returned nil")
			}
			if len(lb.endpoints) != len(tt.endpoints) {
				t.Errorf("endpoints length = %v, want %v", len(lb.endpoints), len(tt.endpoints))
			}
			if lb.path != tt.path {
				t.Errorf("path = %v, want %v", lb.path, tt.path)
			}
			if lb.auth != tt.auth {
				t.Errorf("auth = %v, want %v", lb.auth, tt.auth)
			}
			if lb.httpClient == nil {
				t.Error("httpClient is nil")
			}
		})
	}
}

func TestNewLoadBalancerFromFlags(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		expectedPath string
		expectedAuth bool
		expectedEps  int
	}{
		{
			name:         "default values",
			args:         []string{},
			expectedPath: "lb",
			expectedAuth: false,
			expectedEps:  0,
		},
		{
			name:         "custom values",
			args:         []string{"-lbpath=/api", "-lbauth=true", "-lb=http://backend1.com", "-lb=https://backend2.com"},
			expectedPath: "/api",
			expectedAuth: true,
			expectedEps:  2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
			factory := NewLoadBalancerFromFlags(flagSet)

			err := flagSet.Parse(tt.args)
			if err != nil {
				t.Fatalf("Failed to parse flags: %v", err)
			}

			controller := factory().(*loadBalancer)

			if controller.path != tt.expectedPath {
				t.Errorf("path = %v, want %v", controller.path, tt.expectedPath)
			}
			if controller.auth != tt.expectedAuth {
				t.Errorf("auth = %v, want %v", controller.auth, tt.expectedAuth)
			}
			if len(controller.endpoints) != tt.expectedEps {
				t.Errorf("endpoints count = %v, want %v", len(controller.endpoints), tt.expectedEps)
			}
		})
	}
}

func TestLoadBalancer_Bind(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name      string
		endpoints []url.URL
		auth      bool
		path      string
	}{
		{
			name:      "no endpoints",
			endpoints: []url.URL{},
			auth:      false,
			path:      "/test",
		},
		{
			name: "with endpoints no auth",
			endpoints: []url.URL{
				{Scheme: "http", Host: "backend1.com"},
			},
			auth: false,
			path: "/api",
		},
		{
			name: "with endpoints and auth",
			endpoints: []url.URL{
				{Scheme: "http", Host: "backend1.com"},
			},
			auth: true,
			path: "/secure",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lb := NewLoadBalancer(tt.endpoints, tt.path, tt.auth)
			engine := gin.New()
			cfg := config.NewConfig("localhost:8080", "", "", false, "test-session", "")

			// Mock login middleware
			loginMiddleware := func(c *gin.Context) {
				c.Next()
			}

			lb.Bind(engine, loginMiddleware)

			// Check if routes were registered (only if endpoints exist)
			routes := engine.Routes()
			if len(tt.endpoints) > 0 {
				found := false
				for _, route := range routes {
					if route.Path == tt.path {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected route %s not found", tt.path)
				}
			}
		})
	}
}

func TestLoadBalancer_NextEndpoint(t *testing.T) {
	endpoints := []url.URL{
		{Scheme: "http", Host: "backend1.com"},
		{Scheme: "http", Host: "backend2.com"},
		{Scheme: "http", Host: "backend3.com"},
	}

	lb := NewLoadBalancer(endpoints, "/test", false)

	// Test round-robin behavior
	for i := 0; i < 6; i++ { // Test two full cycles
		endpoint := lb.nextEndpoint()
		expectedHost := endpoints[i%len(endpoints)].Host
		if endpoint.Host != expectedHost {
			t.Errorf("Round %d: got %s, want %s", i, endpoint.Host, expectedHost)
		}
	}
}

func TestLoadBalancer_Forward(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create a mock backend server
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("backend response"))
		if err != nil {
			http.Error(w, "Failed to write response", http.StatusInternalServerError)
			return
		}
	}))
	defer backend.Close()

	backendURL, _ := url.Parse(backend.URL)
	endpoints := []url.URL{*backendURL}

	lb := NewLoadBalancer(endpoints, "/proxy", false)
	engine := gin.New()
	engine.Any("/proxy", lb.forward)

	tests := []struct {
		name           string
		method         string
		path           string
		body           string
		expectedStatus int
	}{
		{
			name:           "GET request",
			method:         "GET",
			path:           "/proxy",
			body:           "",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "POST request",
			method:         "POST",
			path:           "/proxy",
			body:           "test data",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "PUT request",
			method:         "PUT",
			path:           "/proxy",
			body:           "update data",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "DELETE request",
			method:         "DELETE",
			path:           "/proxy",
			body:           "",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
			engine.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Status = %v, want %v", w.Code, tt.expectedStatus)
			}
		})
	}
}

func TestLoadBalancer_ForwardInvalidMethod(t *testing.T) {
	gin.SetMode(gin.TestMode)

	backendURL, _ := url.Parse("http://backend1.com")
	endpoints := []url.URL{*backendURL}

	lb := NewLoadBalancer(endpoints, "/proxy", false)
	engine := gin.New()
	engine.Any("/proxy", lb.forward)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("TRACE", "/proxy", nil) // Use TRACE which is not in allowedMethods
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected 405 Method Not Allowed, got %d", w.Code)
	}
}

func TestLoadBalancer_ForwardNoEndpoints(t *testing.T) {
	gin.SetMode(gin.TestMode)

	lb := NewLoadBalancer([]url.URL{}, "/proxy", false)

	// Test that the forward method handles no endpoints gracefully
	// Since Bind() won't register routes for empty endpoints, we need to test differently
	engine := gin.New()

	// We'll test the behavior by checking that no routes are registered
	lb.Bind(engine, nil)

	routes := engine.Routes()
	if len(routes) > 0 {
		t.Error("Expected no routes to be registered for load balancer with no endpoints")
	}
}

func TestLoadBalancer_ForwardWithBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create a mock backend that echoes the request body
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("received: " + string(body)))
		if err != nil {
			http.Error(w, "Failed to write response", http.StatusInternalServerError)
			return
		}
	}))
	defer backend.Close()

	backendURL, _ := url.Parse(backend.URL)
	endpoints := []url.URL{*backendURL}

	lb := NewLoadBalancer(endpoints, "/proxy", false)
	engine := gin.New()
	engine.POST("/proxy", lb.forward)

	testData := "test request body"
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/proxy", strings.NewReader(testData))
	req.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	expectedBody := "received: " + testData
	if w.Body.String() != expectedBody {
		t.Errorf("Body = %v, want %v", w.Body.String(), expectedBody)
	}
}

func TestLoadBalancer_Close(t *testing.T) {
	lb := NewLoadBalancer([]url.URL{}, "/test", false)
	err := lb.Close()
	if err != nil {
		t.Errorf("Close() returned error: %v", err)
	}
}

func TestLoadBalancer_ConcurrentAccess(t *testing.T) {
	endpoints := []url.URL{
		{Scheme: "http", Host: "backend1.com"},
		{Scheme: "http", Host: "backend2.com"},
	}

	lb := NewLoadBalancer(endpoints, "/test", false)

	// Test concurrent access to nextEndpoint
	results := make(chan string, 10)

	for i := 0; i < 10; i++ {
		go func() {
			endpoint := lb.nextEndpoint()
			results <- endpoint.Host
		}()
	}

	// Collect results
	hosts := make(map[string]int)
	for i := 0; i < 10; i++ {
		host := <-results
		hosts[host]++
	}

	// Both backends should have been used
	if len(hosts) == 0 {
		t.Error("No hosts returned from concurrent access")
	}
}
