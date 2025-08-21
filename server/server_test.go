package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/animalet/sargantana-go/config"
	"github.com/animalet/sargantana-go/controller"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

func TestLoadSecrets_SetsEnvVars(t *testing.T) {
	secrets := map[string]string{
		"MY_SECRET1": "supersecret",
		"MY_SECRET2": "123123123123",
		"MY_SECRET3": "asd123asd123",
	}

	dir := t.TempDir()
	for name, value := range secrets {
		secretFile := filepath.Join(dir, name)
		os.WriteFile(secretFile, []byte("   "+value+" \n  \t"), 0644)
		os.Setenv(name, "")
	}

	s := &Server{config: config.NewConfig("", "", dir, false, "")}
	s.loadSecrets()

	// assert that the environment variables are set correctly
	for name, expected := range secrets {
		if got := os.Getenv(name); got != expected {
			t.Errorf("Expected %s=%q, got %s=%q", name, expected, name, got)
		}
	}
}

func TestSessionCookieName_IsCustomizable(t *testing.T) {
	customSessionName, s := testServerWithSession()

	err := s.Start(&sessionController{})
	if err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/session", nil)
	s.httpServer.Handler.ServeHTTP(w, req)

	cookieHeader := w.Header().Get("Set-Cookie")
	if cookieHeader == "" {
		t.Fatalf("No Set-Cookie header found")
	}
	if cookieHeader[:len(customSessionName)] != customSessionName {
		t.Errorf("Expected cookie name %q, got header: %q", customSessionName, cookieHeader)
	}
}

func testServerWithSession() (string, *Server) {
	os.Setenv("SESSION_SECRET", "dummysecret")
	customSessionName := "my-custom-session"
	s := NewServer("localhost", 0, "", ".", true, customSessionName)
	return customSessionName, s
}

type sessionController struct {
	controller.IController
}

func (s sessionController) Bind(server *gin.Engine, _ config.Config, _ gin.HandlerFunc) {
	server.GET("/session", func(c *gin.Context) {
		session := sessions.Default(c)
		session.Set("test_key", "test_value")
		session.Save()
		c.String(http.StatusOK, "Session set")
	})
}
