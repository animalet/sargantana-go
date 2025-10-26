package main

import (
	"os"

	"github.com/animalet/sargantana-go/examples/blog_example/blog"
	"github.com/animalet/sargantana-go/pkg/config"
	"github.com/animalet/sargantana-go/pkg/controller"
	"github.com/animalet/sargantana-go/pkg/database"
	"github.com/animalet/sargantana-go/pkg/resolver"
	"github.com/animalet/sargantana-go/pkg/server"
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

	// Register property resolvers (must be done before loading config)
	resolver.Register("env", resolver.NewEnvResolver())

	cfg, err := config.ReadConfig("./config.yaml")
	if err != nil {
		panic(err)
	}

	// Register file resolver after loading basic config
	if cfg.ServerConfig.SecretsDir != "" {
		resolver.Register("file", resolver.NewFileResolver(cfg.ServerConfig.SecretsDir))
	}

	// Register Vault resolver if configured
	if cfg.Vault != nil {
		vaultClient, err := cfg.Vault.CreateClient()
		if err != nil {
			panic(errors.Wrap(err, "failed to create Vault client"))
		}
		resolver.Register("vault", resolver.NewVaultResolver(vaultClient, cfg.Vault.Path))
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

	err = sargantana.StartAndWaitForSignal()
	if err != nil {
		panic(err)
	}
}
