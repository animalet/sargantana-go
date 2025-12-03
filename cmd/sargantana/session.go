package main

import (
	"context"

	"github.com/animalet/sargantana-go/internal/session"
	"github.com/animalet/sargantana-go/pkg/config"
	"github.com/animalet/sargantana-go/pkg/database"
	"github.com/animalet/sargantana-go/pkg/server"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

// sessionStoreCloser is a function that cleans up session store resources
type sessionStoreCloser func() error

// configureSessionStore sets up the session store based on available database configuration.
// Priority: Redis > Memcached > PostgreSQL > MongoDB > Cookie (default)
// Returns a closer function that should be deferred to clean up resources
func configureSessionStore(cfg *config.Config, srv *server.Server, sessionSecret []byte, debugMode bool) (sessionStoreCloser, error) {
	// Try Redis first
	if closer, err := configureRedisStore(cfg, srv, sessionSecret, debugMode); closer != nil || err != nil {
		return closer, err
	}

	// Try Memcached second
	if closer, err := configureMemcachedStore(cfg, srv, sessionSecret, debugMode); closer != nil || err != nil {
		return closer, err
	}

	// Try PostgreSQL third
	if closer, err := configurePostgresStore(cfg, srv, sessionSecret, debugMode); closer != nil || err != nil {
		return closer, err
	}

	// Try MongoDB fourth
	if closer, err := configureMongoDBStore(cfg, srv, sessionSecret, debugMode); closer != nil || err != nil {
		return closer, err
	}

	// Default: Cookie-based sessions (no cleanup needed)
	return func() error { return nil }, nil
}

func configureRedisStore(cfg *config.Config, srv *server.Server, sessionSecret []byte, debugMode bool) (sessionStoreCloser, error) {
	redisPool, err := config.GetClient[database.RedisConfig](cfg, "redis")
	if err != nil {
		return nil, errors.Wrap(err, "failed to load or create Redis client")
	}
	if redisPool == nil {
		return nil, nil
	}

	store, err := session.NewRedisSessionStore(debugMode, sessionSecret, *redisPool)
	if err != nil {
		_ = (*redisPool).Close()
		return nil, errors.Wrap(err, "failed to create Redis session store")
	}

	srv.SetSessionStore(store)
	log.Info().Msg("Using Redis session store")

	return func() error {
		if err := (*redisPool).Close(); err != nil {
			log.Error().Err(err).Msg("Failed to close Redis pool")
			return err
		}
		return nil
	}, nil
}

func configureMongoDBStore(cfg *config.Config, srv *server.Server, sessionSecret []byte, debugMode bool) (sessionStoreCloser, error) {
	mongoClient, mongoCfg, err := config.GetClientAndConfig[database.MongoDBConfig](cfg, "mongodb")
	if err != nil {
		return nil, errors.Wrap(err, "failed to load or create MongoDB client")
	}
	if mongoClient == nil {
		return nil, nil
	}

	store, err := session.NewMongoDBSessionStore(!debugMode, sessionSecret, *mongoClient, mongoCfg.Database, "sessions")
	if err != nil {
		_ = (*mongoClient).Disconnect(context.Background())
		return nil, errors.Wrap(err, "failed to create MongoDB session store")
	}

	srv.SetSessionStore(store)
	log.Info().Msg("Using MongoDB session store")

	return func() error {
		if err := (*mongoClient).Disconnect(context.Background()); err != nil {
			log.Error().Err(err).Msg("Failed to disconnect MongoDB client")
			return err
		}
		return nil
	}, nil
}

func configurePostgresStore(cfg *config.Config, srv *server.Server, sessionSecret []byte, debugMode bool) (sessionStoreCloser, error) {
	pgPool, err := config.GetClient[database.PostgresConfig](cfg, "postgres")
	if err != nil {
		return nil, errors.Wrap(err, "failed to load or create PostgreSQL client")
	}
	if pgPool == nil {
		return nil, nil
	}

	store, err := session.NewPostgresSessionStore(!debugMode, sessionSecret, *pgPool, "sessions")
	if err != nil {
		(*pgPool).Close()
		return nil, errors.Wrap(err, "failed to create PostgreSQL session store")
	}

	srv.SetSessionStore(store)
	log.Info().Msg("Using PostgreSQL session store")

	return func() error {
		(*pgPool).Close()
		return nil
	}, nil
}

func configureMemcachedStore(cfg *config.Config, srv *server.Server, sessionSecret []byte, debugMode bool) (sessionStoreCloser, error) {
	memcachedClient, err := config.GetClient[database.MemcachedConfig](cfg, "memcached")
	if err != nil {
		return nil, errors.Wrap(err, "failed to load or create Memcached client")
	}
	if memcachedClient == nil {
		return nil, nil
	}

	store, err := session.NewMemcachedSessionStore(!debugMode, sessionSecret, *memcachedClient)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create Memcached session store")
	}

	srv.SetSessionStore(store)
	log.Info().Msg("Using Memcached session store")

	// Memcached client doesn't need explicit cleanup
	return func() error { return nil }, nil
}
