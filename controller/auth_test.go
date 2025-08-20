package controller

import (
	"encoding/gob"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/markbates/goth"
)

func setupAuthTestServer() *gin.Engine {
	gob.Register(UserObject{})
	gin.SetMode(gin.TestMode)
	r := gin.New()
	store := cookie.NewStore([]byte("secret-key"))
	r.Use(sessions.Sessions("mysession", store))
	auth := NewAuth("callback")
	auth.Bind(r, LoginFunc)
	return r
}

func TestAuthUser_NoUser(t *testing.T) {
	r := setupAuthTestServer()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/auth/user", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Errorf("Expected 403 Forbidden, got %d", w.Code)
	}
}

func TestAuthUser_ValidUser(t *testing.T) {
	r := setupAuthTestServer()
	// Set user in session
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/setuser", nil)
	r.GET("/setuser", func(c *gin.Context) {
		sess := sessions.Default(c)
		onehour := time.Now().Add(time.Hour)
		sess.Set("user", UserObject{
			Id:   "1",
			User: goth.User{ExpiresAt: onehour},
		})
		if err := sess.Save(); err != nil {
			c.String(500, "session save error")
			return
		}
		c.String(200, "ok")
	})
	r.ServeHTTP(w, req)
	cookieHeader := w.Header().Get("Set-Cookie")

	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/auth/user", nil)
	req2.Header.Set("Cookie", cookieHeader)
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Errorf("Expected 200 OK, got %d", w2.Code)
	}
	if !strings.Contains(w2.Body.String(), "1") {
		t.Errorf("Expected user ID in response, got %q", w2.Body.String())
	}
}

func TestAuthUser_ExpiredUser(t *testing.T) {
	r := setupAuthTestServer()
	// Set expired user in session
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/setuser", nil)
	r.GET("/setuser", func(c *gin.Context) {
		sess := sessions.Default(c)
		sess.Set("user", UserObject{
			Id:   "1",
			User: goth.User{ExpiresAt: time.Now().Add(-time.Hour)},
		})
		if err := sess.Save(); err != nil {
			c.String(500, "session save error")
			return
		}
		c.String(200, "ok")
	})
	r.ServeHTTP(w, req)
	cookieHeader := w.Header().Get("Set-Cookie")

	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/auth/user", nil)
	req2.Header.Set("Cookie", cookieHeader)
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401 Unauthorized, got %d", w2.Code)
	}
}
