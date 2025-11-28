package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/animalet/sargantana-go/pkg/config"
	"github.com/animalet/sargantana-go/pkg/config/secrets"
	"github.com/animalet/sargantana-go/pkg/controller"
	"github.com/animalet/sargantana-go/pkg/database"
	"github.com/animalet/sargantana-go/pkg/server"
	"github.com/animalet/sargantana-go/pkg/server/session"
	"github.com/pkg/errors"
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
	server.RegisterController("auth", controller.NewAuthController)
	server.RegisterController("load_balancer", controller.NewLoadBalancerController)
	server.RegisterController("static", controller.NewStaticController)
	server.RegisterController("template", controller.NewTemplateController)
	defer func() {
		// Exit gracefully after panicking
		if r := recover(); r != nil {
			log.Fatal().Msgf("Fatal error: %v", r)
			os.Exit(1)
		}
	}()
	cfg := readConfig(*configFile)

	serverCfg, err := config.Get[server.SargantanaConfig](cfg, "server")
	if err != nil {
		panic(errors.Wrap(err, "failed to load server configuration"))
	}
	if serverCfg == nil {
		panic("server configuration is required")
	}
	sargantana := server.NewServer(*serverCfg)
	// Configure authentication using goth (OAuth2)
	sargantana.SetAuthenticator(controller.NewGothAuthenticator())

	// Configure session store based on available database configuration
	// Priority: Redis > MongoDB > PostgreSQL > Memcached
	// Only the first configured store will be used

	redisCfg, err := config.Get[database.RedisConfig](cfg, "redis")
	if err != nil {
		panic(errors.Wrap(err, "failed to load Redis configuration"))
	}

	if redisCfg != nil {
		redisPool, err := redisCfg.CreateClient()
		if err != nil {
			panic(errors.Wrap(err, "failed to create Redis client"))
		}
		defer func() {
			if err := redisPool.Close(); err != nil {
				log.Error().Err(err).Msg("Failed to close Redis pool")
			}
		}()
		store, err := session.NewRedisSessionStore(*debugMode, []byte(serverCfg.WebServerConfig.SessionSecret), redisPool)
		if err != nil {
			log.Fatal().Err(err).Msg("Unable to create Redis session store")
			os.Exit(1)
		}
		sargantana.SetSessionStore(store)
	}

	mongoCfg, err := config.Get[database.MongoDBConfig](cfg, "mongodb")
	if err != nil {
		panic(errors.Wrap(err, "failed to load MongoDB configuration"))
	}

	if mongoCfg != nil {
		mongoClient, err := mongoCfg.CreateClient()
		if err != nil {
			panic(errors.Wrap(err, "failed to create MongoDB client"))
		}
		defer func() {
			if err := mongoClient.Disconnect(context.Background()); err != nil {
				log.Error().Err(err).Msg("Failed to disconnect MongoDB client")
			}
		}()
		store, err := session.NewMongoDBSessionStore(!*debugMode, []byte(serverCfg.WebServerConfig.SessionSecret), mongoClient, mongoCfg.Database, "sessions")
		if err != nil {
			log.Fatal().Err(err).Msg("Unable to create MongoDB session store")
			os.Exit(1)
		}
		sargantana.SetSessionStore(store)
	}

	postgresCfg, err := config.Get[database.PostgresConfig](cfg, "postgres")
	if err != nil {
		panic(errors.Wrap(err, "failed to load PostgreSQL configuration"))
	}

	if postgresCfg != nil {
		pgPool, err := postgresCfg.CreateClient()
		if err != nil {
			panic(errors.Wrap(err, "failed to create PostgreSQL client"))
		}
		defer func() {
			pgPool.Close()
		}()
		store, err := session.NewPostgresSessionStore(!*debugMode, []byte(serverCfg.WebServerConfig.SessionSecret), pgPool, "sessions")
		if err != nil {
			log.Fatal().Err(err).Msg("Unable to create PostgreSQL session store")
			os.Exit(1)
		}
		sargantana.SetSessionStore(store)
	}

	memcachedCfg, err := config.Get[database.MemcachedConfig](cfg, "memcached")
	if err != nil {
		panic(errors.Wrap(err, "failed to load Memcached configuration"))
	}

	if memcachedCfg != nil {
		memcachedClient, err := memcachedCfg.CreateClient()
		if err != nil {
			panic(errors.Wrap(err, "failed to create Memcached client"))
		}
		store, err := session.NewMemcachedSessionStore(!*debugMode, []byte(serverCfg.WebServerConfig.SessionSecret), memcachedClient)
		if err != nil {
			log.Fatal().Err(err).Msg("Unable to create Memcached session store")
			os.Exit(1)
		}
		sargantana.SetSessionStore(store)
	}

	err = sargantana.StartAndWaitForSignal()
	if err != nil {
		log.Fatal().Err(err).Msg("Server error")
	}
}

func readConfig(file string) *config.Config {
	cfg, err := config.NewConfig(file)
	if err != nil {
		panic(err)
	}

	// Register Vault provider if configured
	vaultCfg, err := config.Get[secrets.VaultConfig](cfg, "vault")
	if err != nil {
		panic(errors.Wrap(err, "failed to load Vault configuration"))
	}
	if vaultCfg != nil {
		vaultClient, err := vaultCfg.CreateClient()
		if err != nil {
			panic(errors.Wrap(err, "failed to create Vault client"))
		}
		secrets.Register("vault", secrets.NewVaultSecretLoader(vaultClient, vaultCfg.Path))
	}

	// Register file provider if configured
	fileResolverCfg, err := config.Get[secrets.FileSecretConfig](cfg, "file_resolver")
	if err != nil {
		panic(errors.Wrap(err, "failed to load file secret resolver configuration"))
	}
	if fileResolverCfg != nil {
		fileResolver, err := fileResolverCfg.CreateClient()
		if err != nil {
			panic(errors.Wrap(err, "failed to create file secret provider"))
		}
		secrets.Register("file", fileResolver)
	}

	// Register AWS Secrets Manager provider if configured
	awsCfg, err := config.Get[secrets.AWSConfig](cfg, "aws")
	if err != nil {
		panic(errors.Wrap(err, "failed to load AWS Secrets Manager configuration"))
	}
	if awsCfg != nil {
		awsClient, err := awsCfg.CreateClient()
		if err != nil {
			panic(errors.Wrap(err, "failed to create AWS Secrets Manager client"))
		}
		secrets.Register("aws", secrets.NewAWSSecretLoader(awsClient, awsCfg.SecretName))
	}

	return cfg
}
