package session

import (
	"log"
	"net/http"

	"github.com/gin-contrib/sessions"
	redissessions "github.com/gin-contrib/sessions/redis"
	"github.com/gomodule/redigo/redis"
	"github.com/markbates/goth/gothic"
)

func NewRedisSessionStore(isReleaseMode bool, secret []byte, pool *redis.Pool) sessions.Store {
	store, err := redissessions.NewStoreWithPool(pool, secret)
	if err != nil {
		log.Fatalf("Failed to create session store: %v", err)
	}

	rediStore, err := redissessions.GetRedisStore(store)
	if err != nil {
		log.Fatalf("Failed to get redis store: %v", err)
	}

	rediStore.Options.Path = "/"
	rediStore.Options.MaxAge = 86400 // 24 hours
	rediStore.Options.Secure = isReleaseMode
	rediStore.Options.HttpOnly = true
	rediStore.Options.SameSite = http.SameSiteLaxMode

	gothic.Store = rediStore
	return store
}
