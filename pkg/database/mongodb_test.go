package database

import (
	"context"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// TestMongoDBHealthCheck tests if MongoDB service is running
func TestMongoDBHealthCheck(t *testing.T) {
	config := &MongoDBConfig{
		URI:            "mongodb://admin:adminpass@localhost:27017",
		Database:       "sessions_test",
		ConnectTimeout: 10 * time.Second,
	}

	client, err := config.CreateClient()
	if err != nil {
		t.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer func() { _ = client.Disconnect(context.Background()) }()

	// Verify connection with ping
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx, nil); err != nil {
		t.Fatalf("Failed to ping MongoDB: %v", err)
	}
}

// TestMongoDBConfig_Validate tests the validation logic
func TestMongoDBConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  MongoDBConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: MongoDBConfig{
				URI:            "mongodb://localhost:27017",
				Database:       "testdb",
				ConnectTimeout: 10 * time.Second,
				MaxPoolSize:    100,
				MinPoolSize:    10,
			},
			wantErr: false,
		},
		{
			name: "valid config with auth",
			config: MongoDBConfig{
				URI:            "mongodb://localhost:27017",
				Database:       "testdb",
				Username:       "user",
				Password:       "pass",
				AuthSource:     "admin",
				ConnectTimeout: 10 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "missing URI",
			config: MongoDBConfig{
				Database: "testdb",
			},
			wantErr: true,
		},
		{
			name: "missing database",
			config: MongoDBConfig{
				URI: "mongodb://localhost:27017",
			},
			wantErr: true,
		},
		{
			name: "negative timeout",
			config: MongoDBConfig{
				URI:            "mongodb://localhost:27017",
				Database:       "testdb",
				ConnectTimeout: -1 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "min pool size greater than max",
			config: MongoDBConfig{
				URI:         "mongodb://localhost:27017",
				Database:    "testdb",
				MaxPoolSize: 10,
				MinPoolSize: 20,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("MongoDBConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestMongoDBConfig_CreateClient tests client creation
func TestMongoDBConfig_CreateClient(t *testing.T) {
	tests := []struct {
		name      string
		config    *MongoDBConfig
		wantError bool
	}{
		{
			name: "valid configuration with auth",
			config: &MongoDBConfig{
				URI:            "mongodb://admin:adminpass@localhost:27017",
				Database:       "sessions_test",
				ConnectTimeout: 10 * time.Second,
			},
			wantError: false,
		},
		{
			name: "valid configuration without auth in URI",
			config: &MongoDBConfig{
				URI:            "mongodb://localhost:27017",
				Database:       "sessions_test",
				Username:       "admin",
				Password:       "adminpass",
				AuthSource:     "admin",
				ConnectTimeout: 10 * time.Second,
			},
			wantError: false,
		},
		{
			name: "invalid URI",
			config: &MongoDBConfig{
				URI:            "invalid://localhost:27017",
				Database:       "sessions_test",
				ConnectTimeout: 10 * time.Second,
			},
			wantError: true,
		},
		{
			name: "invalid host",
			config: &MongoDBConfig{
				URI:            "mongodb://invalid-host:99999",
				Database:       "sessions_test",
				ConnectTimeout: 2 * time.Second,
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := tt.config.CreateClient()
			if tt.wantError {
				if err == nil {
					t.Error("Expected error but got none")
					if client != nil {
						_ = client.Disconnect(context.Background())
					}
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if client == nil {
					t.Error("Expected client but got nil")
				} else {
					defer func() { _ = client.Disconnect(context.Background()) }()
				}
			}
		})
	}
}

// TestMongoDBOperations tests basic MongoDB operations
func TestMongoDBOperations(t *testing.T) {
	config := &MongoDBConfig{
		URI:            "mongodb://admin:adminpass@localhost:27017",
		Database:       "sessions_test",
		ConnectTimeout: 10 * time.Second,
	}

	client, err := config.CreateClient()
	if err != nil {
		t.Fatalf("Failed to create MongoDB client: %v", err)
	}
	defer func() { _ = client.Disconnect(context.Background()) }()

	// Get test collection
	collection := client.Database(config.Database).Collection("test_collection")
	ctx := context.Background()

	// Clean up before test
	_ = collection.Drop(ctx)

	// Test Insert operation
	doc := bson.M{"name": "test", "value": "test-value"}
	result, err := collection.InsertOne(ctx, doc)
	if err != nil {
		t.Fatalf("Failed to insert document: %v", err)
	}

	if result.InsertedID == nil {
		t.Error("Expected inserted ID but got nil")
	}

	// Test Find operation
	var foundDoc bson.M
	err = collection.FindOne(ctx, bson.M{"name": "test"}).Decode(&foundDoc)
	if err != nil {
		t.Fatalf("Failed to find document: %v", err)
	}

	if foundDoc["value"] != "test-value" {
		t.Errorf("Expected 'test-value', got '%v'", foundDoc["value"])
	}

	// Test Update operation
	_, err = collection.UpdateOne(ctx, bson.M{"name": "test"}, bson.M{"$set": bson.M{"value": "updated-value"}})
	if err != nil {
		t.Fatalf("Failed to update document: %v", err)
	}

	// Verify update
	err = collection.FindOne(ctx, bson.M{"name": "test"}).Decode(&foundDoc)
	if err != nil {
		t.Fatalf("Failed to find updated document: %v", err)
	}

	if foundDoc["value"] != "updated-value" {
		t.Errorf("Expected 'updated-value', got '%v'", foundDoc["value"])
	}

	// Test Delete operation
	_, err = collection.DeleteOne(ctx, bson.M{"name": "test"})
	if err != nil {
		t.Fatalf("Failed to delete document: %v", err)
	}

	// Verify deletion
	err = collection.FindOne(ctx, bson.M{"name": "test"}).Decode(&foundDoc)
	if err != mongo.ErrNoDocuments {
		t.Errorf("Expected no documents error, got: %v", err)
	}

	// Clean up after test
	_ = collection.Drop(ctx)
}

// TestMongoDBAuthenticationWithCredentials tests authentication using separate credentials
func TestMongoDBAuthenticationWithCredentials(t *testing.T) {
	config := &MongoDBConfig{
		URI:            "mongodb://localhost:27017",
		Database:       "sessions_test",
		Username:       "testuser",
		Password:       "testpass",
		AuthSource:     "admin",
		ConnectTimeout: 10 * time.Second,
	}

	client, err := config.CreateClient()
	if err != nil {
		t.Fatalf("Failed to create MongoDB client with credentials: %v", err)
	}
	defer func() { _ = client.Disconnect(context.Background()) }()

	// Test access to the database
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Should be able to perform operations on sessions_test database
	collection := client.Database(config.Database).Collection("test_collection")
	_, err = collection.CountDocuments(ctx, bson.M{})
	if err != nil {
		t.Errorf("Failed to access database with credentials: %v", err)
	}
}

// BenchmarkMongoDBInsert benchmarks MongoDB INSERT operations
func BenchmarkMongoDBInsert(b *testing.B) {
	config := &MongoDBConfig{
		URI:            "mongodb://admin:adminpass@localhost:27017",
		Database:       "sessions_test",
		ConnectTimeout: 10 * time.Second,
		MaxPoolSize:    100,
	}

	client, err := config.CreateClient()
	if err != nil {
		b.Fatalf("Failed to create MongoDB client: %v", err)
	}
	defer func() { _ = client.Disconnect(context.Background()) }()

	collection := client.Database(config.Database).Collection("bench_collection")
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		doc := bson.M{"key": "bench-key", "value": "bench-value", "index": i}
		_, _ = collection.InsertOne(ctx, doc)
	}

	// Clean up
	_ = collection.Drop(ctx)
}

// BenchmarkMongoDBFind benchmarks MongoDB FIND operations
func BenchmarkMongoDBFind(b *testing.B) {
	config := &MongoDBConfig{
		URI:            "mongodb://admin:adminpass@localhost:27017",
		Database:       "sessions_test",
		ConnectTimeout: 10 * time.Second,
		MaxPoolSize:    100,
	}

	client, err := config.CreateClient()
	if err != nil {
		b.Fatalf("Failed to create MongoDB client: %v", err)
	}
	defer func() { _ = client.Disconnect(context.Background()) }()

	collection := client.Database(config.Database).Collection("bench_collection")
	ctx := context.Background()

	// Insert test document
	doc := bson.M{"key": "bench-key", "value": "bench-value"}
	_, _ = collection.InsertOne(ctx, doc)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var result bson.M
		_ = collection.FindOne(ctx, bson.M{"key": "bench-key"}).Decode(&result)
	}

	// Clean up
	_ = collection.Drop(ctx)
}

// TestMongoDBConfig_ValidateTLS tests TLS configuration validation
func TestMongoDBConfig_ValidateTLS(t *testing.T) {
	tests := []struct {
		name    string
		config  MongoDBConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "TLS with cert but no key",
			config: MongoDBConfig{
				URI:      "mongodb://localhost:27017",
				Database: "testdb",
				TLS: &MongoDBTLSConfig{
					CertFile: "/path/to/cert.pem",
				},
			},
			wantErr: true,
			errMsg:  "key_file is required when cert_file is specified",
		},
		{
			name: "TLS with key but no cert",
			config: MongoDBConfig{
				URI:      "mongodb://localhost:27017",
				Database: "testdb",
				TLS: &MongoDBTLSConfig{
					KeyFile: "/path/to/key.pem",
				},
			},
			wantErr: true,
			errMsg:  "cert_file is required when key_file is specified",
		},
		{
			name: "TLS with nonexistent cert file",
			config: MongoDBConfig{
				URI:      "mongodb://localhost:27017",
				Database: "testdb",
				TLS: &MongoDBTLSConfig{
					CertFile: "/nonexistent/cert.pem",
					KeyFile:  "/nonexistent/key.pem",
				},
			},
			wantErr: true,
			errMsg:  "cert_file does not exist",
		},
		{
			name: "TLS with nonexistent key file",
			config: MongoDBConfig{
				URI:      "mongodb://localhost:27017",
				Database: "testdb",
				TLS: &MongoDBTLSConfig{
					CertFile: "../../certs/client.crt",
					KeyFile:  "/nonexistent/key.pem",
				},
			},
			wantErr: true,
			errMsg:  "key_file does not exist",
		},
		{
			name: "TLS with nonexistent CA file",
			config: MongoDBConfig{
				URI:      "mongodb://localhost:27017",
				Database: "testdb",
				TLS: &MongoDBTLSConfig{
					CAFile: "/nonexistent/ca.pem",
				},
			},
			wantErr: true,
			errMsg:  "ca_file does not exist",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				} else if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("Expected error containing %q, got %q", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

// TestMongoDBConfig_BuildTLSConfig tests TLS config building
func TestMongoDBConfig_BuildTLSConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  MongoDBConfig
		wantNil bool
		wantErr bool
	}{
		{
			name: "TLS disabled (nil config) returns nil",
			config: MongoDBConfig{
				URI:      "mongodb://localhost:27017",
				Database: "testdb",
				TLS:      nil,
			},
			wantNil: true,
			wantErr: false,
		},
		{
			name: "TLS with InsecureSkipVerify",
			config: MongoDBConfig{
				URI:      "mongodb://localhost:27017",
				Database: "testdb",
				TLS: &MongoDBTLSConfig{
					InsecureSkipVerify: true,
				},
			},
			wantNil: false,
			wantErr: false,
		},
		{
			name: "TLS with invalid cert/key pair",
			config: MongoDBConfig{
				URI:      "mongodb://localhost:27017",
				Database: "testdb",
				TLS: &MongoDBTLSConfig{
					CertFile: "../../certs/ca.crt",
					KeyFile:  "../../certs/client.key",
				},
			},
			wantNil: false,
			wantErr: true,
		},
		{
			name: "TLS with valid CA file",
			config: MongoDBConfig{
				URI:      "mongodb://localhost:27017",
				Database: "testdb",
				TLS: &MongoDBTLSConfig{
					CAFile: "../../certs/ca.crt",
				},
			},
			wantNil: false,
			wantErr: false,
		},
		{
			name: "TLS with invalid CA file content",
			config: MongoDBConfig{
				URI:      "mongodb://localhost:27017",
				Database: "testdb",
				TLS: &MongoDBTLSConfig{
					CAFile: "../../certs/client.key",
				},
			},
			wantNil: false,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tlsConfig, err := tt.config.buildTLSConfig()

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}

			if tt.wantNil {
				if tlsConfig != nil {
					t.Error("Expected nil TLS config but got non-nil")
				}
			} else if !tt.wantErr {
				if tlsConfig == nil {
					t.Error("Expected non-nil TLS config but got nil")
				}
			}
		})
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
