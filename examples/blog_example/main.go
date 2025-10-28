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

	// Register secret providers (must be done before loading config)
	secrets.Register("env", secrets.NewEnvLoader())

	cfg, err := config.ReadConfig("./config.yaml")
	if err != nil {
		panic(err)
	}

	// Register file provider if configured
	fileResolverCfg, err := config.LoadConfig[secrets.FileSecretConfig]("file_resolver", cfg)
	if err == nil {
		fileResolver, err := fileResolverCfg.CreateClient()
		if err != nil {
			panic(errors.Wrap(err, "failed to create file secret provider"))
		}
		secrets.Register("file", fileResolver)
	}

	// Register Vault provider if configured
	vaultCfg, err := config.LoadConfig[secrets.VaultConfig]("vault", cfg)
	if err == nil {
		vaultClient, err := vaultCfg.CreateClient()
		if err != nil {
			panic(errors.Wrap(err, "failed to create Vault client"))
		}
		secrets.Register("vault", secrets.NewVaultSecretLoader(vaultClient, vaultCfg.Path))
	}

	postgresCfg, err := config.LoadConfig[database.PostgresConfig]("database", cfg)
	if err != nil {
		panic(errors.Wrap(err, "failed to load PostgreSQL configuration"))
	}

	pool, err := postgresCfg.CreateClient()
	if err != nil {
		panic(errors.Wrap(err, "failed to create PostgreSQL connection pool"))
	}
	defer pool.Close()

	server.AddControllerType("blog", blog.NewBlogController(pool))

	sargantana, err := server.NewServer(cfg)
	if err != nil {
		panic(err)
	}

	// Set up Redis session store if configured
	redisCfg, err := config.LoadConfig[database.RedisConfig]("redis", cfg)
	if err == nil {
		redisPool, err := redisCfg.CreateClient()
		if err != nil {
			panic(errors.Wrap(err, "failed to create Redis connection pool"))
		}
		store, err := session.NewRedisSessionStore(false, []byte(cfg.ServerConfig.SessionSecret), redisPool)
		if err != nil {
			panic(errors.Wrap(err, "failed to create Redis session store"))
		}
		sargantana.SetSessionStore(&store)
		log.Info().Msg("Redis session store configured")
	}

	err = sargantana.StartAndWaitForSignal()
	if err != nil {
		panic(err)
	}
}
