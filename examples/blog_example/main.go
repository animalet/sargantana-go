package main

import (
	"os"

	"github.com/animalet/sargantana-go/examples/blog_example/blog"
	"github.com/animalet/sargantana-go/pkg/config"
	"github.com/animalet/sargantana-go/pkg/controller"
	"github.com/animalet/sargantana-go/pkg/resolver"
	"github.com/animalet/sargantana-go/pkg/server"
	"github.com/jackc/pgx"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type DatabaseConfig struct {
	Host     string `yaml:"host"`
	Port     uint16 `yaml:"port"`
	Database string `yaml:"database"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
}

func (d DatabaseConfig) Validate() error {
	if d.Host == "" {
		return errors.New("host must be set and non-empty")
	}
	if d.Port == 0 {
		return errors.New("port must be set and non-zero")
	}
	if d.Database == "" {
		return errors.New("database must be set and non-empty")
	}
	if d.User == "" {
		return errors.New("user must be set and non-empty")
	}
	if d.Password == "" {
		return errors.New("password must be set and non-empty")
	}
	return nil
}

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
		vaultClient, err := config.CreateVaultClient(cfg.Vault)
		if err != nil {
			panic(errors.Wrap(err, "failed to create Vault client"))
		}
		resolver.Register("vault", resolver.NewVaultResolver(vaultClient, cfg.Vault.Path))
	}

	dbConfig, err := config.LoadConfig[DatabaseConfig]("database", cfg)
	if err != nil {
		panic(err)
	}

	database, err := pgx.Connect(pgx.ConnConfig{
		Host:     dbConfig.Host,
		Port:     dbConfig.Port,
		Database: dbConfig.Database,
		User:     dbConfig.User,
		Password: dbConfig.Password,
	})

	if err != nil {
		panic(err)
	}
	defer func() {
		if err := database.Close(); err != nil {
			log.Printf("Error closing database: %v", err)
		}
	}()

	server.AddControllerType("blog", blog.NewBlogController(database))

	sargantana, err := server.NewServer(cfg)

	if err != nil {
		panic(err)
	}

	err = sargantana.StartAndWaitForSignal()
	if err != nil {
		panic(err)
	}
}
