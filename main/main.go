package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/animalet/sargantana-go/config"
	"github.com/animalet/sargantana-go/controller"
	"github.com/animalet/sargantana-go/database"
	"github.com/animalet/sargantana-go/server"
	"github.com/animalet/sargantana-go/session"
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
	config.RegisterPropertyResolver("env", config.NewEnvResolver())

	// File resolver (if secrets directory is configured)
	if cfg.ServerConfig.SecretsDir != "" {
		config.RegisterPropertyResolver("file", config.NewFileResolver(cfg.ServerConfig.SecretsDir))
		log.Info().Str("secrets_dir", cfg.ServerConfig.SecretsDir).Msg("File resolver registered")
	}

	// Vault resolver (if Vault is configured)
	if cfg.Vault != nil {
		vaultClient, err := config.CreateVaultClient(cfg.Vault)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to create Vault client")
			os.Exit(1)
		}
		config.RegisterPropertyResolver("vault", config.NewVaultResolver(vaultClient, cfg.Vault.Path))
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
		redisPool := database.NewRedisPoolWithConfig(redisCfg)
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
