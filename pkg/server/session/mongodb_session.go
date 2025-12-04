package session

import (
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/mongo/mongodriver"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/mongo"
)

// NewMongoDBSessionStore creates a new MongoDB-backed session store.
// It configures the session store with appropriate security settings based on the mode.
//
// Parameters:
//   - secure: Whether to set the Secure flag on session cookies (typically true in release mode)
//   - secret: The secret key used for session encryption (should be at least 32 bytes)
//   - client: Pre-configured MongoDB client with connection details
//   - database: The database name to use for session storage
//   - collection: The collection name to use for session storage (default: "sessions" if empty)
//
// Returns:
//   - sessions.Store: The configured MongoDB session store
//   - error: An error if store creation fails
//
// Example usage:
//
//	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
//	if err != nil {
//	    log.Fatal(err)
//	}
//	store, err := session.NewMongoDBSessionStore(true, []byte("secret-key"), mongoClient, "mydb", "sessions")
//	if err != nil {
//	    log.Fatal(err)
//	}
func NewMongoDBSessionStore(secure bool, secret []byte, client *mongo.Client, database, collection string) (sessions.Store, error) {
	if client == nil {
		return nil, errors.New("MongoDB client cannot be nil")
	}

	if len(secret) == 0 {
		return nil, errors.New("session secret cannot be empty")
	}

	if database == "" {
		return nil, errors.New("database name cannot be empty")
	}

	// Use default collection name if not specified
	if collection == "" {
		collection = "sessions"
	}

	// Get the MongoDB collection
	coll := client.Database(database).Collection(collection)

	// Create MongoDB-backed session store using mongo-driver
	store := mongodriver.NewStore(coll, 3600, true, secret) // maxAge: 3600 seconds, ensureTTL: true

	// Configure session options
	store.Options(sessions.Options{
		Path:     "/",
		MaxAge:   86400, // 24 hours
		Secure:   secure,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode, // Strict mode for enhanced security
	})

	return store, nil
}
