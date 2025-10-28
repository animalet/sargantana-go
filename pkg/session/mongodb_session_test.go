package session

import (
	"context"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// TestNewMongoDBSessionStore tests creating a new MongoDB session store
func TestNewMongoDBSessionStore(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://admin:adminpass@localhost:27017"))
	if err != nil {
		t.Skipf("MongoDB not available: %v", err)
	}
	defer func() { _ = client.Disconnect(context.Background()) }()

	// Test ping to ensure connection
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		t.Skipf("MongoDB not available: %v", err)
	}

	secret := []byte("test-secret-key-that-is-long-enough")

	tests := []struct {
		name       string
		secure     bool
		secret     []byte
		client     *mongo.Client
		database   string
		collection string
		wantErr    bool
	}{
		{
			name:       "valid configuration with default collection",
			secure:     true,
			secret:     secret,
			client:     client,
			database:   "sessions_test",
			collection: "",
			wantErr:    false,
		},
		{
			name:       "valid configuration with custom collection",
			secure:     false,
			secret:     secret,
			client:     client,
			database:   "sessions_test",
			collection: "custom_sessions",
			wantErr:    false,
		},
		{
			name:       "nil client",
			secure:     true,
			secret:     secret,
			client:     nil,
			database:   "sessions_test",
			collection: "sessions",
			wantErr:    true,
		},
		{
			name:       "empty secret",
			secure:     true,
			secret:     []byte{},
			client:     client,
			database:   "sessions_test",
			collection: "sessions",
			wantErr:    true,
		},
		{
			name:       "nil secret",
			secure:     true,
			secret:     nil,
			client:     client,
			database:   "sessions_test",
			collection: "sessions",
			wantErr:    true,
		},
		{
			name:       "empty database",
			secure:     true,
			secret:     secret,
			client:     client,
			database:   "",
			collection: "sessions",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, err := NewMongoDBSessionStore(tt.secure, tt.secret, tt.client, tt.database, tt.collection)
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				if store != nil {
					t.Error("Expected nil store but got non-nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if store == nil {
					t.Error("Expected store but got nil")
				}
			}
		})
	}
}

// TestMongoDBSessionStore_Integration tests the session store with actual MongoDB
func TestMongoDBSessionStore_Integration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://admin:adminpass@localhost:27017"))
	if err != nil {
		t.Skipf("MongoDB not available: %v", err)
	}
	defer func() { _ = client.Disconnect(context.Background()) }()

	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		t.Skipf("MongoDB not available: %v", err)
	}

	secret := []byte("test-secret-key-that-is-long-enough")
	store, err := NewMongoDBSessionStore(false, secret, client, "sessions_test", "test_sessions")
	if err != nil {
		t.Fatalf("Failed to create session store: %v", err)
	}

	if store == nil {
		t.Fatal("Expected non-nil store")
	}

	// Clean up test collection
	collection := client.Database("sessions_test").Collection("test_sessions")
	_ = collection.Drop(context.Background())
}

// TestMongoDBSessionStore_WithAuthentication tests session store with authenticated user
func TestMongoDBSessionStore_WithAuthentication(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Connect as testuser with limited permissions
	client, err := mongo.Connect(ctx, options.Client().
		ApplyURI("mongodb://testuser:testpass@localhost:27017").
		SetAuth(options.Credential{
			Username:   "testuser",
			Password:   "testpass",
			AuthSource: "admin",
		}))
	if err != nil {
		t.Skipf("MongoDB not available or authentication failed: %v", err)
	}
	defer func() { _ = client.Disconnect(context.Background()) }()

	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		t.Skipf("MongoDB not available: %v", err)
	}

	secret := []byte("test-secret-key-that-is-long-enough")
	store, err := NewMongoDBSessionStore(false, secret, client, "sessions_test", "auth_test_sessions")
	if err != nil {
		t.Fatalf("Failed to create session store with authenticated client: %v", err)
	}

	if store == nil {
		t.Fatal("Expected non-nil store")
	}

	// Clean up test collection
	collection := client.Database("sessions_test").Collection("auth_test_sessions")
	_ = collection.Drop(context.Background())
}

// TestMongoDBSessionStore_DefaultCollection tests using default collection name
func TestMongoDBSessionStore_DefaultCollection(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://admin:adminpass@localhost:27017"))
	if err != nil {
		t.Skipf("MongoDB not available: %v", err)
	}
	defer func() { _ = client.Disconnect(context.Background()) }()

	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		t.Skipf("MongoDB not available: %v", err)
	}

	secret := []byte("test-secret-key-that-is-long-enough")

	// Pass empty string for collection to use default "sessions"
	store, err := NewMongoDBSessionStore(true, secret, client, "sessions_test", "")
	if err != nil {
		t.Fatalf("Failed to create session store with default collection: %v", err)
	}

	if store == nil {
		t.Fatal("Expected non-nil store")
	}

	// Clean up test collection (default name is "sessions")
	collection := client.Database("sessions_test").Collection("sessions")
	_ = collection.Drop(context.Background())
}
