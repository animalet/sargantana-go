package controller

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func setupTestServerWithLB(endpoints []string, lbAuth bool) *gin.Engine {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	parsedEndpoints := []url.URL{}
	for _, e := range endpoints {
		u, _ := url.Parse(e)
		parsedEndpoints = append(parsedEndpoints, *u)
	}
	lb := NewLoadBalancer(parsedEndpoints, "/proxy", lbAuth)
	lb.Bind(r, nil)

	return r
}

func TestForward_Integration(t *testing.T) {
	// Start a real backend server
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("forward integration test"))
	}))
	defer backend.Close()

	u, err := url.Parse(backend.URL)
	if err != nil {
		t.Fatalf("Failed to parse backend URL: %v", err)
	}
	lb := NewLoadBalancer([]url.URL{*u}, "/proxy", false)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/proxy", lb.forward)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/proxy", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "forward integration test") {
		t.Errorf("Expected backend response, got %q", w.Body.String())
	}
}

func TestLoadBalancerRoute_NoEndpoints(t *testing.T) {
	r := setupTestServerWithLB(nil, false)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/proxy", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("Expected 404 for proxy with no endpoints, got %d", w.Code)
	}
}

func TestLoadBalancerRoute_WithEndpoints(t *testing.T) {
	// Start two real backend servers with unique responses
	backend1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("hello from backend1"))
	}))
	defer backend1.Close()
	backend2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("hello from backend2"))
	}))
	defer backend2.Close()

	u1, err := url.Parse(backend1.URL)
	if err != nil {
		t.Fatalf("Failed to parse backend1 URL: %v", err)
	}
	u2, err := url.Parse(backend2.URL)
	if err != nil {
		t.Fatalf("Failed to parse backend2 URL: %v", err)
	}
	endpoints := []string{u1.Scheme + "://" + u1.Host, u2.Scheme + "://" + u2.Host}

	r := setupTestServerWithLB(endpoints, false)

	// First request should hit backend1
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest("GET", "/proxy", nil)
	r.ServeHTTP(w1, req1)
	if w1.Code != http.StatusOK {
		t.Errorf("Expected 200 for proxy with real endpoint, got %d", w1.Code)
	}
	if !strings.Contains(w1.Body.String(), "hello from backend1") {
		t.Errorf("Expected backend1 response, got %q", w1.Body.String())
	}

	// Second request should hit backend2
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/proxy", nil)
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Errorf("Expected 200 for proxy with real endpoint, got %d", w2.Code)
	}
	if !strings.Contains(w2.Body.String(), "hello from backend2") {
		t.Errorf("Expected backend2 response, got %q", w2.Body.String())
	}

	// Third request should hit backend1 again (round robin)
	w3 := httptest.NewRecorder()
	req3, _ := http.NewRequest("GET", "/proxy", nil)
	r.ServeHTTP(w3, req3)
	if w3.Code != http.StatusOK {
		t.Errorf("Expected 200 for proxy with real endpoint, got %d", w3.Code)
	}
	if !strings.Contains(w3.Body.String(), "hello from backend1") {
		t.Errorf("Expected backend1 response, got %q", w3.Body.String())
	}
}
