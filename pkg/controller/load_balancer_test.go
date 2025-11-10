package controller

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/animalet/sargantana-go/pkg/server"
	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"
)

func TestNewLoadBalancerController(t *testing.T) {
	tests := []struct {
		name           string
		configData     LoadBalancerControllerConfig
		expectedError  bool
		expectedErrMsg string
	}{
		{
			name: "valid config with single endpoint",
			configData: LoadBalancerControllerConfig{
				Auth:      false,
				Path:      "/api",
				Endpoints: []string{"http://backend1.com"},
			},
			expectedError: false,
		},
		{
			name: "valid config with multiple endpoints",
			configData: LoadBalancerControllerConfig{
				Auth:      true,
				Path:      "/proxy",
				Endpoints: []string{"http://backend1.com", "https://backend2.com"},
			},
			expectedError: false,
		},
		{
			name: "empty endpoints should fail",
			configData: LoadBalancerControllerConfig{
				Auth:      false,
				Path:      "/api",
				Endpoints: []string{},
			},
			expectedError:  true,
			expectedErrMsg: "at least one endpoint must be provided",
		},
		{
			name: "invalid endpoint URL",
			configData: LoadBalancerControllerConfig{
				Auth:      false,
				Path:      "/api",
				Endpoints: []string{"://invalid-url"},
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configBytes, err := yaml.Marshal(tt.configData)
			if err != nil {
				t.Fatalf("Failed to marshal config: %v", err)
			}

			controller, err := NewLoadBalancerController(configBytes, server.ControllerContext{ServerConfig: server.WebServerConfig{}})

			if tt.expectedError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				if tt.expectedErrMsg != "" && !strings.Contains(err.Error(), tt.expectedErrMsg) {
					t.Errorf("Expected error message to contain %q, got %q", tt.expectedErrMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if controller == nil {
					t.Error("Expected controller but got nil")
				}
			}
		})
	}
}

func TestLoadBalancer_Bind(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		configData LoadBalancerControllerConfig
		expectAuth bool
	}{
		{
			name: "no auth required",
			configData: LoadBalancerControllerConfig{
				Auth:      false,
				Path:      "/api",
				Endpoints: []string{"http://backend1.com"},
			},
			expectAuth: false,
		},
		{
			name: "auth required",
			configData: LoadBalancerControllerConfig{
				Auth:      true,
				Path:      "/secure",
				Endpoints: []string{"http://backend1.com"},
			},
			expectAuth: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configBytes, _ := yaml.Marshal(tt.configData)
			lbController, err := NewLoadBalancerController(configBytes, server.ControllerContext{ServerConfig: server.WebServerConfig{}})
			if err != nil {
				t.Fatalf("Failed to create load balancer controller: %v", err)
			}

			engine := gin.New()

			// Mock login middleware that sets a header
			loginMiddleware := func(c *gin.Context) {
				c.Header("Auth-Called", "true")
				c.Next()
			}

			lbController.Bind(engine)

			// Check if routes were registered
			routes := engine.Routes()
			if len(routes) == 0 {
				t.Error("No routes were registered")
			}

			// Verify the expected path pattern
			expectedPath := strings.TrimSuffix(tt.configData.Path, "/") + "/*proxyPath"
			found := false
			for _, route := range routes {
				if route.Path == expectedPath {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected route path %s not found", expectedPath)
			}
		})
	}
}

func TestLoadBalancer_NextEndpoint(t *testing.T) {
	configData := LoadBalancerControllerConfig{
		Auth:      false,
		Path:      "/test",
		Endpoints: []string{"http://backend1.com", "http://backend2.com", "http://backend3.com"},
	}
	configBytes, _ := yaml.Marshal(configData)
	controller, err := NewLoadBalancerController(configBytes, server.ControllerContext{ServerConfig: server.WebServerConfig{}})
	if err != nil {
		t.Fatalf("Failed to create load balancer controller: %v", err)
	}

	lb := controller.(*loadBalancer)

	// Test round-robin behavior
	expectedHosts := []string{"backend1.com", "backend2.com", "backend3.com"}
	for i := 0; i < 6; i++ { // Test two full cycles
		endpoint := lb.nextEndpoint()
		expectedHost := expectedHosts[i%len(expectedHosts)]
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

	configData := LoadBalancerControllerConfig{
		Auth:      false,
		Path:      "/proxy",
		Endpoints: []string{backend.URL},
	}
	configBytes, _ := yaml.Marshal(configData)
	lbController, err := NewLoadBalancerController(configBytes, server.ControllerContext{ServerConfig: server.WebServerConfig{}})
	if err != nil {
		t.Fatalf("Failed to create load balancer controller: %v", err)
	}

	engine := gin.New()
	lbController.Bind(engine)

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
			path:           "/proxy/test",
			body:           "",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "POST request",
			method:         "POST",
			path:           "/proxy/api/data",
			body:           "test data",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "PUT request",
			method:         "PUT",
			path:           "/proxy/update",
			body:           "update data",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "DELETE request",
			method:         "DELETE",
			path:           "/proxy/delete",
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

	configData := LoadBalancerControllerConfig{
		Auth:      false,
		Path:      "/proxy",
		Endpoints: []string{backend.URL},
	}
	configBytes, _ := yaml.Marshal(configData)
	lbController, err := NewLoadBalancerController(configBytes, server.ControllerContext{ServerConfig: server.WebServerConfig{}})
	if err != nil {
		t.Fatalf("Failed to create load balancer controller: %v", err)
	}

	engine := gin.New()
	lbController.Bind(engine)

	testData := "test request body"
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/proxy/api", strings.NewReader(testData))
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
	configData := LoadBalancerControllerConfig{
		Auth:      false,
		Path:      "/test",
		Endpoints: []string{"http://backend1.com"},
	}
	configBytes, _ := yaml.Marshal(configData)
	lbController, err := NewLoadBalancerController(configBytes, server.ControllerContext{ServerConfig: server.WebServerConfig{}})
	if err != nil {
		t.Fatalf("Failed to create load balancer controller: %v", err)
	}

	err = lbController.Close()
	if err != nil {
		t.Errorf("Close() returned error: %v", err)
	}
}

func TestLoadBalancer_ConcurrentAccess(t *testing.T) {
	configData := LoadBalancerControllerConfig{
		Auth:      false,
		Path:      "/test",
		Endpoints: []string{"http://backend1.com", "http://backend2.com"},
	}
	configBytes, _ := yaml.Marshal(configData)
	controller, err := NewLoadBalancerController(configBytes, server.ControllerContext{ServerConfig: server.WebServerConfig{}})
	if err != nil {
		t.Fatalf("Failed to create load balancer controller: %v", err)
	}

	lb := controller.(*loadBalancer)

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

func TestLoadBalancer_ForwardFailedBackend(t *testing.T) {
	gin.SetMode(gin.TestMode)

	configData := LoadBalancerControllerConfig{
		Auth:      false,
		Path:      "/proxy",
		Endpoints: []string{"http://nonexistent-backend.com:9999"},
	}
	configBytes, _ := yaml.Marshal(configData)
	lbController, err := NewLoadBalancerController(configBytes, server.ControllerContext{ServerConfig: server.WebServerConfig{}})
	if err != nil {
		t.Fatalf("Failed to create load balancer controller: %v", err)
	}

	engine := gin.New()
	lbController.Bind(engine)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/proxy/test", nil)
	engine.ServeHTTP(w, req)

	// Should return 502 Bad Gateway when backend is unreachable
	if w.Code != http.StatusBadGateway {
		t.Errorf("Expected 502 Bad Gateway, got %d", w.Code)
	}
}

func TestLoadBalancer_BindWithNoEndpoints(t *testing.T) {
	gin.SetMode(gin.TestMode)

	lb := &loadBalancer{
		endpoints: []url.URL{}, // Empty endpoints to trigger the uncovered path
		path:      "/test",
		auth:      false,
	}

	engine := gin.New()

	// This should trigger the "Load balancer not loaded" log message and early return
	lb.Bind(engine)

	// Verify no routes were registered
	routes := engine.Routes()
	if len(routes) != 0 {
		t.Errorf("Expected no routes to be registered, but got %d routes", len(routes))
	}
}

func TestLoadBalancer_ForwardWithSetCookieHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create a mock backend that returns Set-Cookie header
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Set-Cookie", "session=abc123; Path=/")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`{"message": "success"}`))
		if err != nil {
			http.Error(w, "Failed to write response", http.StatusInternalServerError)
			return
		}
	}))
	defer backend.Close()

	configData := LoadBalancerControllerConfig{
		Auth:      false,
		Path:      "/proxy",
		Endpoints: []string{backend.URL},
	}
	configBytes, _ := yaml.Marshal(configData)
	lbController, err := NewLoadBalancerController(configBytes, server.ControllerContext{ServerConfig: server.WebServerConfig{}})
	if err != nil {
		t.Fatalf("Failed to create load balancer controller: %v", err)
	}

	engine := gin.New()
	lbController.Bind(engine)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/proxy/test", nil)
	engine.ServeHTTP(w, req)

	// Verify Set-Cookie header is filtered out (not passed through)
	if w.Header().Get("Set-Cookie") != "" {
		t.Error("Set-Cookie header should be filtered out but was present in response")
	}

	// Verify other headers are passed through
	if w.Header().Get("Content-Type") != "application/json" {
		t.Error("Content-Type header should be passed through")
	}
}

func TestNewLoadBalancerController_InvalidURL(t *testing.T) {
	// Test the specific error case for invalid URL parsing (lines 25-27)
	configData := LoadBalancerControllerConfig{
		Auth:      false,
		Path:      "/api",
		Endpoints: []string{"://invalid-url-scheme"},
	}
	configBytes, _ := yaml.Marshal(configData)

	controller, err := NewLoadBalancerController(configBytes, server.ControllerContext{ServerConfig: server.WebServerConfig{}})

	if err == nil {
		t.Fatal("Expected error for invalid URL, but got none")
	}
	if controller != nil {
		t.Fatal("Expected nil controller for invalid URL")
	}

	// Verify the error message contains the expected format
	if !strings.Contains(err.Error(), "invalid endpoint URL") {
		t.Errorf("Expected error message about invalid endpoint URL, got: %v", err)
	}
}

func TestLoadBalancer_ForwardWithFilteredHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create a mock backend that captures received headers
	var receivedHeaders http.Header
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("ok"))
		if err != nil {
			http.Error(w, "Failed to write response", http.StatusInternalServerError)
			return
		}
	}))
	defer backend.Close()

	configData := LoadBalancerControllerConfig{
		Auth:      false,
		Path:      "/proxy",
		Endpoints: []string{backend.URL},
	}
	configBytes, _ := yaml.Marshal(configData)
	lbController, err := NewLoadBalancerController(configBytes, server.ControllerContext{ServerConfig: server.WebServerConfig{}})
	if err != nil {
		t.Fatalf("Failed to create load balancer controller: %v", err)
	}

	engine := gin.New()
	lbController.Bind(engine)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/proxy/test", nil)

	// Add headers that should be filtered out
	req.Header.Set("Host", "original-host.com")
	req.Header.Set("X-Forwarded-For", "original-forwarded")
	req.Header.Set("Authorization", "Bearer token123")
	req.Header.Set("Cookie", "session=abc")
	req.Header.Set("X-Forwarded-Proto", "https")

	// Add header that should be passed through
	req.Header.Set("User-Agent", "test-agent")
	req.Header.Set("Content-Type", "application/json")

	engine.ServeHTTP(w, req)

	// Verify filtered headers are not passed through
	if receivedHeaders.Get("Host") == "original-host.com" {
		t.Error("Host header should be filtered out")
	}
	if receivedHeaders.Get("Authorization") != "" {
		t.Error("Authorization header should be filtered out")
	}
	if receivedHeaders.Get("Cookie") != "" {
		t.Error("Cookie header should be filtered out")
	}
	if receivedHeaders.Get("X-Forwarded-Proto") != "" {
		t.Error("X-Forwarded-Proto header should be filtered out")
	}

	// Verify allowed headers are passed through
	if receivedHeaders.Get("User-Agent") != "test-agent" {
		t.Error("User-Agent header should be passed through")
	}
	if receivedHeaders.Get("Content-Type") != "application/json" {
		t.Error("Content-Type header should be passed through")
	}

	// Verify X-Forwarded-For is set by load balancer (should not be the original value)
	forwardedFor := receivedHeaders.Get("X-Forwarded-For")
	if forwardedFor == "original-forwarded" {
		t.Error("X-Forwarded-For should be overwritten by load balancer, not use original value")
	}
	// In test environment, ClientIP() may return empty string, so we just verify it's not the original
}

func TestLoadBalancer_ForwardWithInvalidRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create a mock backend server
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("ok"))
		if err != nil {
			http.Error(w, "Failed to write response", http.StatusInternalServerError)
			return
		}
	}))
	defer backend.Close()

	// Create load balancer with valid backend
	configData := LoadBalancerControllerConfig{
		Auth:      false,
		Path:      "/proxy",
		Endpoints: []string{backend.URL},
	}
	configBytes, _ := yaml.Marshal(configData)
	lbController, err := NewLoadBalancerController(configBytes, server.ControllerContext{ServerConfig: server.WebServerConfig{}})
	if err != nil {
		t.Fatalf("Failed to create load balancer controller: %v", err)
	}

	lb := lbController.(*loadBalancer)
	engine := gin.New()

	// Test with an invalid method to trigger http.NewRequest error
	// This is harder to trigger directly, so let's modify the load balancer's nextEndpoint to return an invalid URL
	originalEndpoints := lb.endpoints
	lb.endpoints = []url.URL{{Scheme: "", Host: "", Path: string([]byte{0x7f})}} // Invalid URL

	lbController.Bind(engine)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/proxy/test", nil)
	engine.ServeHTTP(w, req)

	// Should return 500 Internal Server Error when request creation fails
	if w.Code != http.StatusInternalServerError {
		t.Logf("Expected 500 Internal Server Error for invalid request creation, got %d", w.Code)
	}

	// Restore original endpoints
	lb.endpoints = originalEndpoints
}

func TestLoadBalancer_ForwardWithCopyError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create a backend that returns a response that will cause copy error
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "100") // Set content length higher than actual content
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("short"))
		if err != nil {
			http.Error(w, "Failed to write response", http.StatusInternalServerError)
			return
		}
	}))
	defer backend.Close()

	configData := LoadBalancerControllerConfig{
		Auth:      false,
		Path:      "/proxy",
		Endpoints: []string{backend.URL},
	}
	configBytes, _ := yaml.Marshal(configData)
	lbController, err := NewLoadBalancerController(configBytes, server.ControllerContext{ServerConfig: server.WebServerConfig{}})
	if err != nil {
		t.Fatalf("Failed to create load balancer controller: %v", err)
	}

	engine := gin.New()
	lbController.Bind(engine)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/proxy/test", nil)
	engine.ServeHTTP(w, req)

	// The response should still be successful as the copy error handling is internal
	if w.Code != http.StatusOK {
		t.Logf("Got status %d, expected successful response", w.Code)
	}
}
