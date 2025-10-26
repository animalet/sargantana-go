package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/animalet/sargantana-go/pkg/config"
	"github.com/animalet/sargantana-go/pkg/controller"
	"github.com/animalet/sargantana-go/pkg/database"
	"github.com/animalet/sargantana-go/pkg/resolver"
	"github.com/animalet/sargantana-go/pkg/server"
	"github.com/animalet/sargantana-go/pkg/session"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Version information set during build
var (
	version = "dev"
)

func main() {
	showVersion := flag.Bool("version", false, "Show version information")
	debugMode := flag.Bool("debug", false, "Enable debug mode")
	configFile := flag.String("config", "", "Path to configuration file")

	flag.Parse()

	if *debugMode {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	if *showVersion {
		fmt.Printf("%s %s\n", "sargantana-go", version)
		os.Exit(0)
	}

	if *configFile == "" {
		n, err := fmt.Fprintln(os.Stderr, "Error: -config flag is required")
		if err != nil || n <= 0 {
			panic("Failed to print error message")
		}
		os.Exit(1)
	}

	log.Logger = log.Output(zerolog.ConsoleWriter{
		Out:        os.Stdout,
		NoColor:    false,
		TimeFormat: "2006-01-02 15:04:05",
	})
	server.SetDebug(*debugMode)
	server.AddControllerType("auth", controller.NewAuthController)
	server.AddControllerType("load_balancer", controller.NewLoadBalancerController)
	server.AddControllerType("static", controller.NewStaticController)
	server.AddControllerType("template", controller.NewTemplateController)

	cfg, err := config.ReadConfig(*configFile)
	if err != nil {
		log.Fatal().Err(err).Msg("Unable to read configuration file")
		os.Exit(1)
	}

	// Register property resolvers
	// Environment resolver (default - always register first)
	resolver.Register("env", resolver.NewEnvResolver())

	// File resolver (if secrets directory is configured)
	if cfg.ServerConfig.SecretsDir != "" {
		resolver.Register("file", resolver.NewFileResolver(cfg.ServerConfig.SecretsDir))
		log.Info().Str("secrets_dir", cfg.ServerConfig.SecretsDir).Msg("File resolver registered")
	}

	// Vault resolver (if Vault is configured)
	if cfg.Vault != nil {
		vaultClient, err := cfg.Vault.CreateClient()
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to create Vault client")
			os.Exit(1)
		}
		resolver.Register("vault", resolver.NewVaultResolver(vaultClient, cfg.Vault.Path))
		log.Info().Msg("Vault resolver registered")
	}

	sargantana, err := server.NewServer(cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create server")
		os.Exit(1)
	}

	redisCfg, err := config.LoadConfig[database.RedisConfig]("redis", cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("Unable to load Redis configuration")
		os.Exit(1)
	}

	if redisCfg != nil {
		redisPool, err := redisCfg.CreateClient()
		if err != nil {
			log.Fatal().Err(err).Msg("Unable to create Redis connection pool")
			os.Exit(1)
		}
		store, err := session.NewRedisSessionStore(*debugMode, []byte(cfg.ServerConfig.SessionSecret), redisPool)
		if err != nil {
			log.Fatal().Err(err).Msg("Unable to create Redis session store")
			os.Exit(1)
		}
		sargantana.SetSessionStore(&store)
	}

	err = sargantana.StartAndWaitForSignal()
	if err != nil {
		log.Fatal().Err(err).Msg("Server error")
	}
}
