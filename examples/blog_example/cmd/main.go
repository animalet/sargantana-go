package main

import (
	"os"

	"github.com/animalet/sargantana-go/examples/blog_example/blog"
	"github.com/animalet/sargantana-go/pkg/config"
	"github.com/animalet/sargantana-go/pkg/config/secrets"
	"github.com/animalet/sargantana-go/pkg/controller"
	"github.com/animalet/sargantana-go/pkg/database"
	"github.com/animalet/sargantana-go/pkg/server"
	"github.com/animalet/sargantana-go/pkg/server/session"
	"github.com/gomodule/redigo/redis"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{
		Out:        os.Stdout,
		NoColor:    false,
		TimeFormat: "2006-01-02 15:04:05",
	})
	server.SetDebug(true)
	server.RegisterController("auth", controller.NewAuthController)
	server.RegisterController("static", controller.NewStaticController)
	server.RegisterController("template", controller.NewTemplateController)
	cfg := readConfig()
	pool := newPgPool(cfg)
	defer pool.Close()
	server.RegisterController("blog", blog.NewBlogController(pool))

	sargantana, redisPool := newServer(cfg)
	defer func() {
		if err := redisPool.Close(); err != nil {
			log.Error().Err(err).Msg("Failed to close Redis pool")
		}
	}()
	err := sargantana.StartAndWaitForSignal()
	if err != nil {
		panic(err)
	}
}

func readConfig() *config.Config {
	config.UseFormat(config.YamlFormat)
	cfg, err := config.NewConfig("./config.yaml")
	if err != nil {
		panic(err)
	}

	// Register Vault provider if configured
	vaultClient, vaultCfg, err := config.GetClientAndConfig[secrets.VaultConfig](cfg, "vault")
	if err != nil {
		panic(errors.Wrap(err, "failed to load or create Vault client"))
	}
	if vaultClient != nil {
		secrets.Register("vault", secrets.NewVaultSecretLoader(*vaultClient, vaultCfg.Path))
	}

	// Register file provider if configured
	fileResolver, err := config.GetClient[secrets.FileSecretConfig](cfg, "file_resolver")
	if err != nil {
		panic(errors.Wrap(err, "failed to load or create file secret provider"))
	}
	if fileResolver != nil {
		secrets.Register("file", *fileResolver)
	}
	return cfg
}

func newServer(cfg *config.Config) (sargantana *server.Server, redisPool *redis.Pool) {
	serverCfg, err := config.Get[server.SargantanaConfig](cfg, "sargantana")
	if err != nil {
		panic(errors.Wrap(err, "failed to load server configuration"))
	}
	if serverCfg == nil {
		panic("server configuration is required")
	}
	sargantana = server.NewServer(*serverCfg)
	// Configure authentication using goth (OAuth2)
	sargantana.SetAuthenticator(controller.NewGothAuthenticator())

	// Set up Redis session store if configured
	redisPoolPtr, err := config.GetClient[database.RedisConfig](cfg, "redis")
	if err != nil {
		panic(errors.Wrap(err, "failed to load or create Redis client"))
	}
	if redisPoolPtr != nil {
		store, err := session.NewRedisSessionStore(false, []byte(serverCfg.WebServerConfig.SessionSecret), *redisPoolPtr)
		if err != nil {
			panic(errors.Wrap(err, "failed to create Redis session store"))
		}
		sargantana.SetSessionStore(store)
		redisPool = *redisPoolPtr
	}
	return sargantana, redisPool
}

func newPgPool(cfg *config.Config) *pgxpool.Pool {
	pool, err := config.GetClient[database.PostgresConfig](cfg, "database")
	if err != nil {
		panic(errors.Wrap(err, "failed to load or create PostgreSQL client"))
	}
	if pool == nil {
		panic("database configuration is required")
	}
	return *pool
}
