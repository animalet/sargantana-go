package session

import (
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/markbates/goth/gothic"
)

func NewCookieStore(isReleaseMode bool, secret []byte) sessions.Store {
	store := cookie.NewStore(secret)

	store.Options(sessions.Options{
		Path:     "/",
		MaxAge:   86400, // 24 hours
		Secure:   isReleaseMode,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	gothic.Store = store
	return store
}
