package controller

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/animalet/sargantana-go/config"
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
			expectedErrMsg: "no endpoints provided for load balancing",
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

			controller, err := NewLoadBalancerController(configBytes, config.ServerConfig{})

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
			lbController, err := NewLoadBalancerController(configBytes, config.ServerConfig{})
			if err != nil {
				t.Fatalf("Failed to create load balancer controller: %v", err)
			}

			engine := gin.New()

			// Mock login middleware that sets a header
			loginMiddleware := func(c *gin.Context) {
				c.Header("Auth-Called", "true")
				c.Next()
			}

			lbController.Bind(engine, loginMiddleware)

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
	controller, err := NewLoadBalancerController(configBytes, config.ServerConfig{})
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
	lbController, err := NewLoadBalancerController(configBytes, config.ServerConfig{})
	if err != nil {
		t.Fatalf("Failed to create load balancer controller: %v", err)
	}

	engine := gin.New()
	lbController.Bind(engine, nil)

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
	lbController, err := NewLoadBalancerController(configBytes, config.ServerConfig{})
	if err != nil {
		t.Fatalf("Failed to create load balancer controller: %v", err)
	}

	engine := gin.New()
	lbController.Bind(engine, nil)

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
	lbController, err := NewLoadBalancerController(configBytes, config.ServerConfig{})
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
	controller, err := NewLoadBalancerController(configBytes, config.ServerConfig{})
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
	lbController, err := NewLoadBalancerController(configBytes, config.ServerConfig{})
	if err != nil {
		t.Fatalf("Failed to create load balancer controller: %v", err)
	}

	engine := gin.New()
	lbController.Bind(engine, nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/proxy/test", nil)
	engine.ServeHTTP(w, req)

	// Should return 502 Bad Gateway when backend is unreachable
	if w.Code != http.StatusBadGateway {
		t.Errorf("Expected 502 Bad Gateway, got %d", w.Code)
	}
}
