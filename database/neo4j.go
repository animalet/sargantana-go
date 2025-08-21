package database

import (
	"context"
	"log"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

type Neo4jOptions struct {
	Uri      string
	Username string
	Password string
	Realm    string
}

func NewNeo4jDriver(options *Neo4jOptions) (neo4j.DriverWithContext, func()) {
	auth := neo4j.BasicAuth(options.Username, options.Password, options.Realm)
	driver, err := neo4j.NewDriverWithContext(options.Uri, auth)
	if err != nil {
		log.Fatalf("Failed to create Neo4j driver: %v", err)
	}

	timeout, cancelFunc := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelFunc()

	session := driver.NewSession(timeout, neo4j.SessionConfig{DatabaseName: "neo4j"})
	defer func() {
		err := session.Close(timeout)
		if err != nil {
			log.Printf("Failed to close Neo4j session: %v", err)
		}
	}()

	err = driver.VerifyConnectivity(timeout)
	if err != nil {
		log.Fatalf("Failed to connect to Neo4j: %v", err)
	}

	log.Printf("Connected to Neo4j at %s", options.Uri)

	closeFunc := func() {
		err := driver.Close(timeout)
		if err != nil {
			log.Printf("Failed to close Neo4j driver: %v", err)
		}
	}

	return driver, closeFunc
}
