package main

import (
	"os"

	"github.com/animalet/sargantana-go/examples/blog_example/blog"
	"github.com/animalet/sargantana-go/pkg/config"
	"github.com/animalet/sargantana-go/pkg/controller"
	"github.com/animalet/sargantana-go/pkg/database"
	"github.com/animalet/sargantana-go/pkg/secrets"
	"github.com/animalet/sargantana-go/pkg/server"
	"github.com/animalet/sargantana-go/pkg/session"
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
	server.AddControllerType("auth", controller.NewAuthController)
	server.AddControllerType("static", controller.NewStaticController)
	server.AddControllerType("template", controller.NewTemplateController)
	cfg := readConfig()
	pool := newPgPool(cfg)
	defer pool.Close()
	server.AddControllerType("blog", blog.NewBlogController(pool))

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

func readConfig() config.Config {
	config.UseFormat(config.YamlFormat)
	cfg, err := config.ReadModular("./config.yaml")
	if err != nil {
		panic(err)
	}

	// Register Vault provider if configured
	vaultCfg, err := config.Load[secrets.VaultConfig](cfg.Get("vault"))
	if err != nil {
		panic(errors.Wrap(err, "failed to load Vault configuration"))
	}
	vaultClient, err := vaultCfg.CreateClient()
	if err != nil {
		panic(errors.Wrap(err, "failed to create Vault client"))
	}
	secrets.Register("vault", secrets.NewVaultSecretLoader(vaultClient, vaultCfg.Path))

	// Register file provider if configured
	fileResolverCfg, err := config.Load[secrets.FileSecretConfig](cfg.Get("file_resolver"))
	if err != nil {
		panic(errors.Wrap(err, "failed to load file secret resolver configuration"))
	}
	fileResolver, err := fileResolverCfg.CreateClient()
	if err != nil {
		panic(errors.Wrap(err, "failed to create file secret provider"))
	}
	secrets.Register("file", fileResolver)
	return cfg
}

func newServer(cfg config.Config) (sargantana *server.Server, redisPool *redis.Pool) {
	serverCfg, err := config.Load[server.SargantanaConfig](cfg.Get("sargantana"))
	if err != nil {
		panic(errors.Wrap(err, "failed to load server configuration"))
	}
	sargantana = server.NewServer(*serverCfg)

	// Set up Redis session store if configured
	redisCfg, err := config.Load[database.RedisConfig](cfg.Get("redis"))
	if err != nil {
		panic(errors.Wrap(err, "failed to load Redis configuration"))
	}
	redisPool, err = redisCfg.CreateClient()
	if err != nil {
		panic(errors.Wrap(err, "failed to create Redis connection pool"))
	}
	store, err := session.NewRedisSessionStore(false, []byte(serverCfg.WebServerConfig.SessionSecret), redisPool)
	if err != nil {
		panic(errors.Wrap(err, "failed to create Redis session store"))
	}
	sargantana.SetSessionStore(store)
	return sargantana, redisPool
}

func newPgPool(cfg config.Config) *pgxpool.Pool {
	postgresCfg, err := config.Load[database.PostgresConfig](cfg.Get("database"))
	if err != nil {
		panic(errors.Wrap(err, "failed to load PostgreSQL configuration"))
	}

	pool, err := postgresCfg.CreateClient()
	if err != nil {
		panic(errors.Wrap(err, "failed to create PostgreSQL connection pool"))
	}
	return pool
}
