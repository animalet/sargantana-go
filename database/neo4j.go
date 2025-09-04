package database

import (
	"context"
	"os"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/rs/zerolog/log"
)

// Neo4jOptions holds the configuration options for connecting to a Neo4j database.
// It includes authentication credentials and connection parameters needed to establish
// a connection to a Neo4j graph database instance.
type Neo4jOptions struct {
	Uri      string // Neo4j server URI (e.g., "bolt://localhost:7687")
	Username string // Database username for authentication
	Password string // Database password for authentication
	Realm    string // Authentication realm (optional, can be empty)
}

// NewNeo4jDriver creates a new Neo4j driver with the provided configuration options.
// It establishes a connection to the Neo4j database, verifies connectivity, and returns
// both the driver instance and a cleanup function for proper resource management.
//
// Parameters:
//   - options: Neo4j connection configuration including URI, username, password, and realm
//
// Returns:
//   - neo4j.DriverWithContext: The configured Neo4j driver instance
//   - func(): Cleanup function that should be called to properly close the driver
//
// The function will log fatal errors and exit if connection establishment fails.
// Example usage:
//
//	driver, cleanup := NewNeo4jDriver(&Neo4jOptions{
//	  Uri: "bolt://localhost:7687",
//	  Username: "neo4j",
//	  Password: "password",
//	})
//	defer cleanup()
func NewNeo4jDriver(options *Neo4jOptions) (neo4j.DriverWithContext, func() error, error) {
	auth := neo4j.BasicAuth(options.Username, options.Password, options.Realm)
	driver, err := neo4j.NewDriverWithContext(options.Uri, auth)
	if err != nil {
		return nil, nil, err
	}

	timeout, cancelFunc := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelFunc()

	session := driver.NewSession(timeout, neo4j.SessionConfig{DatabaseName: "neo4j"})
	defer func() {
		err := session.Close(timeout)
		if err != nil {
			log.Error().Msgf("Failed to close Neo4j session: %v", err)
		}
	}()

	err = driver.VerifyConnectivity(timeout)
	if err != nil {
		return nil, nil, err
	}

	log.Info().Msgf("Connected to Neo4j at %s", options.Uri)

	closeFunc := func() error {
		err := driver.Close(timeout)
		if err != nil {
			return err
		}
		return nil
	}

	return driver, closeFunc, nil
}

// NewNeo4jDriverFromEnv creates a new Neo4j driver using environment variables for configuration.
// This is a convenience function that reads configuration from standard environment variables
// used in Docker and cloud deployments.
//
// Required environment variables:
//   - NEO4J_URI: Neo4j server URI (e.g., "bolt://localhost:7687")
//   - NEO4J_USERNAME: Database username for authentication
//   - NEO4J_PASSWORD: Database password for authentication
//
// Optional environment variables:
//   - NEO4J_REALM: Authentication realm (defaults to empty string)
//
// Returns:
//   - neo4j.DriverWithContext: The configured Neo4j driver instance
//   - func(): Cleanup function that should be called to properly close the driver
//
// The function will log fatal errors and exit if required environment variables are missing
// or if connection establishment fails.
func NewNeo4jDriverFromEnv() (neo4j.DriverWithContext, func() error, error) {
	return NewNeo4jDriver(&Neo4jOptions{
		Uri:      os.Getenv("NEO4J_URI"),
		Username: os.Getenv("NEO4J_USERNAME"),
		Password: os.Getenv("NEO4J_PASSWORD"),
		Realm:    os.Getenv("NEO4J_REALM"),
	})
}
