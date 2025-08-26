package database

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

func TestNeo4jOptions(t *testing.T) {
	options := &Neo4jOptions{
		Uri:      "bolt://localhost:7687",
		Username: "neo4j",
		Password: "password",
		Realm:    "neo4j",
	}

	if options.Uri != "bolt://localhost:7687" {
		t.Errorf("Expected Uri to be 'bolt://localhost:7687', got %s", options.Uri)
	}
	if options.Username != "neo4j" {
		t.Errorf("Expected Username to be 'neo4j', got %s", options.Username)
	}
	if options.Password != "password" {
		t.Errorf("Expected Password to be 'password', got %s", options.Password)
	}
	if options.Realm != "neo4j" {
		t.Errorf("Expected Realm to be 'neo4j', got %s", options.Realm)
	}
}

func TestNewNeo4jDriver_InvalidUri(t *testing.T) {
	options := &Neo4jOptions{
		Uri:      "invalid://invalid:7687",
		Username: "neo4j",
		Password: "password",
		Realm:    "",
	}

	driver, cleanup, err := NewNeo4jDriver(options)

	// We expect this to return an error
	if err == nil {
		t.Error("Expected NewNeo4jDriver to return an error with invalid URI, but it didn't")
		if cleanup != nil {
			_ = cleanup()
		}
		return
	}

	// Driver should be nil when there's an error
	if driver != nil {
		t.Error("Expected driver to be nil when there's an error")
		if cleanup != nil {
			_ = cleanup()
		}
	}

	// Cleanup function should be nil when there's an error
	if cleanup != nil {
		t.Error("Expected cleanup function to be nil when there's an error")
	}
}

func TestNewNeo4jDriver_EmptyUri(t *testing.T) {
	options := &Neo4jOptions{
		Uri:      "",
		Username: "neo4j",
		Password: "password",
		Realm:    "",
	}

	driver, cleanup, err := NewNeo4jDriver(options)

	// We expect this to return an error
	if err == nil {
		t.Error("Expected NewNeo4jDriver to return an error with empty URI, but it didn't")
		if cleanup != nil {
			_ = cleanup()
		}
		return
	}

	// Driver should be nil when there's an error
	if driver != nil {
		t.Error("Expected driver to be nil when there's an error")
		if cleanup != nil {
			_ = cleanup()
		}
	}

	// Cleanup function should be nil when there's an error
	if cleanup != nil {
		t.Error("Expected cleanup function to be nil when there's an error")
	}
}

func TestNewNeo4jDriver_EmptyCredentials(t *testing.T) {
	options := &Neo4jOptions{
		Uri:      "bolt://localhost:7687",
		Username: "",
		Password: "",
		Realm:    "",
	}

	// This might succeed or fail depending on Neo4j configuration
	// We just test that the function doesn't panic and returns proper values
	driver, cleanup, err := NewNeo4jDriver(options)

	if err != nil {
		// If there's an error, driver and cleanup should be nil
		if driver != nil {
			t.Error("Expected driver to be nil when there's an error")
		}
		if cleanup != nil {
			t.Error("Expected cleanup function to be nil when there's an error")
		}
	} else {
		// If no error, driver and cleanup should not be nil
		if driver == nil {
			t.Error("Expected driver to not be nil when there's no error")
		}
		if cleanup == nil {
			t.Error("Expected cleanup function to not be nil when there's no error")
		} else {
			// Test cleanup function
			err := cleanup()
			if err != nil {
				t.Errorf("Cleanup function returned an error: %v", err)
			}
		}
	}
}

func TestNewNeo4jDriverFromEnv_MissingEnvVars(t *testing.T) {
	// Save original env vars
	originalUri := os.Getenv("NEO4J_URI")
	originalUsername := os.Getenv("NEO4J_USERNAME")
	originalPassword := os.Getenv("NEO4J_PASSWORD")
	originalRealm := os.Getenv("NEO4J_REALM")

	// Clear env vars
	_ = os.Unsetenv("NEO4J_URI")
	_ = os.Unsetenv("NEO4J_USERNAME")
	_ = os.Unsetenv("NEO4J_PASSWORD")
	_ = os.Unsetenv("NEO4J_REALM")

	defer func() {
		// Restore original env vars
		if originalUri != "" {
			_ = os.Setenv("NEO4J_URI", originalUri)
		}
		if originalUsername != "" {
			_ = os.Setenv("NEO4J_USERNAME", originalUsername)
		}
		if originalPassword != "" {
			_ = os.Setenv("NEO4J_PASSWORD", originalPassword)
		}
		if originalRealm != "" {
			_ = os.Setenv("NEO4J_REALM", originalRealm)
		}
	}()

	driver, cleanup, err := NewNeo4jDriverFromEnv()

	// We expect this to return an error due to empty URI
	if err == nil {
		t.Error("Expected NewNeo4jDriverFromEnv to return an error with missing env vars, but it didn't")
		if cleanup != nil {
			_ = cleanup()
		}
		return
	}

	// Driver should be nil when there's an error
	if driver != nil {
		t.Error("Expected driver to be nil when there's an error")
		if cleanup != nil {
			_ = cleanup()
		}
	}

	// Cleanup function should be nil when there's an error
	if cleanup != nil {
		t.Error("Expected cleanup function to be nil when there's an error")
	}
}

func TestNewNeo4jDriverFromEnv_WithInvalidEnvVars(t *testing.T) {
	// Save original env vars
	originalUri := os.Getenv("NEO4J_URI")
	originalUsername := os.Getenv("NEO4J_USERNAME")
	originalPassword := os.Getenv("NEO4J_PASSWORD")
	originalRealm := os.Getenv("NEO4J_REALM")

	// Set invalid env vars
	_ = os.Setenv("NEO4J_URI", "invalid://localhost:7687")
	_ = os.Setenv("NEO4J_USERNAME", "neo4j")
	_ = os.Setenv("NEO4J_PASSWORD", "password")
	_ = os.Setenv("NEO4J_REALM", "neo4j")

	defer func() {
		// Restore original env vars
		if originalUri != "" {
			_ = os.Setenv("NEO4J_URI", originalUri)
		} else {
			_ = os.Unsetenv("NEO4J_URI")
		}
		if originalUsername != "" {
			_ = os.Setenv("NEO4J_USERNAME", originalUsername)
		} else {
			_ = os.Unsetenv("NEO4J_USERNAME")
		}
		if originalPassword != "" {
			_ = os.Setenv("NEO4J_PASSWORD", originalPassword)
		} else {
			_ = os.Unsetenv("NEO4J_PASSWORD")
		}
		if originalRealm != "" {
			_ = os.Setenv("NEO4J_REALM", originalRealm)
		} else {
			_ = os.Unsetenv("NEO4J_REALM")
		}
	}()

	driver, cleanup, err := NewNeo4jDriverFromEnv()

	// We expect this to return an error
	if err == nil {
		t.Error("Expected NewNeo4jDriverFromEnv to return an error with invalid URI, but it didn't")
		if cleanup != nil {
			_ = cleanup()
		}
		return
	}

	// Driver should be nil when there's an error
	if driver != nil {
		t.Error("Expected driver to be nil when there's an error")
		if cleanup != nil {
			_ = cleanup()
		}
	}

	// Cleanup function should be nil when there's an error
	if cleanup != nil {
		t.Error("Expected cleanup function to be nil when there's an error")
	}
}

// TestNeo4jDriver_Integration is an integration test that requires a running Neo4j instance
// This test is skipped by default and only runs when NEO4J_TEST_INTEGRATION=true
func TestNeo4jDriver_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	options := &Neo4jOptions{
		Uri:      getEnvOrDefault("NEO4J_URI", "bolt://localhost:7687"),
		Username: getEnvOrDefault("NEO4J_USERNAME", "neo4j"),
		Password: getEnvOrDefault("NEO4J_PASSWORD", "password"),
		Realm:    getEnvOrDefault("NEO4J_REALM", ""),
	}

	driver, cleanup, err := NewNeo4jDriver(options)
	if err != nil {
		t.Fatalf("Failed to create driver: %v", err)
	}
	if driver == nil {
		t.Fatal("Expected driver to be created, got nil")
	}
	if cleanup == nil {
		t.Fatal("Expected cleanup function to be created, got nil")
	}

	defer func() {
		err := cleanup()
		if err != nil {
			t.Errorf("Cleanup function returned an error: %v", err)
		}
	}()

	// Test that we can create a session
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	session := driver.NewSession(ctx, neo4j.SessionConfig{})
	if session == nil {
		t.Fatal("Expected session to be created, got nil")
	}

	defer func() {
		err := session.Close(ctx)
		if err != nil {
			t.Errorf("Failed to close session: %v", err)
		}
	}()

	// Test a simple query
	result, err := session.Run(ctx, "RETURN 1 as number", nil)
	if err != nil {
		t.Fatalf("Failed to run simple query: %v", err)
	}

	if result.Next(ctx) {
		record := result.Record()
		value, ok := record.Get("number")
		if !ok {
			t.Fatal("Expected to get 'number' field from result")
		}
		if value != int64(1) {
			t.Errorf("Expected number to be 1, got %v", value)
		}
	} else {
		t.Fatal("Expected at least one result record")
	}
}

// TestNeo4jDriverFromEnv_Integration is an integration test using environment variables
func TestNeo4jDriverFromEnv_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	// Ensure required env vars are set for integration test
	requiredEnvVars := []string{"NEO4J_URI", "NEO4J_USERNAME", "NEO4J_PASSWORD"}
	for _, envVar := range requiredEnvVars {
		if os.Getenv(envVar) == "" {
			t.Skipf("Skipping integration test. Environment variable %s is not set.", envVar)
		}
	}

	driver, cleanup, err := NewNeo4jDriverFromEnv()
	if err != nil {
		t.Fatalf("Failed to create driver from env: %v", err)
	}

	if driver == nil {
		t.Fatal("Expected driver to be created, got nil")
	}

	if cleanup == nil {
		t.Fatal("Expected cleanup function to be created, got nil")
	}

	defer func() {
		err := cleanup()
		if err != nil {
			t.Errorf("Cleanup function returned an error: %v", err)
		}
	}()

	// Test that we can verify connectivity
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = driver.VerifyConnectivity(ctx)
	if err != nil {
		t.Fatalf("Failed to verify connectivity: %v", err)
	}
}

// Test cleanup function behavior
func TestCleanupFunction(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	options := &Neo4jOptions{
		Uri:      getEnvOrDefault("NEO4J_URI", "bolt://localhost:7687"),
		Username: getEnvOrDefault("NEO4J_USERNAME", "neo4j"),
		Password: getEnvOrDefault("NEO4J_PASSWORD", "password"),
		Realm:    getEnvOrDefault("NEO4J_REALM", ""),
	}

	_, cleanup, err := NewNeo4jDriver(options)
	if err != nil {
		t.Fatalf("Failed to create driver: %v", err)
	}

	if cleanup == nil {
		t.Fatal("Expected cleanup function to be created, got nil")
	}

	// Test that cleanup function works
	err = cleanup()
	if err != nil {
		t.Errorf("Cleanup function returned an error: %v", err)
	}

	// Test that calling cleanup multiple times doesn't cause issues
	err = cleanup()
	if err != nil {
		t.Errorf("Second call to cleanup function returned an error: %v", err)
	}
}

// Helper function to get environment variable with default value
func getEnvOrDefault(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// Benchmark tests
func BenchmarkNewNeo4jDriver(b *testing.B) {
	if testing.Short() {
		b.Skip()
	}

	options := &Neo4jOptions{
		Uri:      getEnvOrDefault("NEO4J_URI", "bolt://localhost:7687"),
		Username: getEnvOrDefault("NEO4J_USERNAME", "neo4j"),
		Password: getEnvOrDefault("NEO4J_PASSWORD", "password"),
		Realm:    getEnvOrDefault("NEO4J_REALM", ""),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, cleanup, err := NewNeo4jDriver(options)
		if err != nil {
			b.Fatalf("Failed to create driver: %v", err)
		}
		if cleanup != nil {
			err = cleanup()
			if err != nil {
				b.Fatalf("Failed to clean up driver: %v", err)
			}
		}
	}
}

func BenchmarkNewNeo4jDriverFromEnv(b *testing.B) {
	if testing.Short() {
		b.Skip()
	}

	// Ensure required env vars are set for benchmark
	requiredEnvVars := []string{"NEO4J_URI", "NEO4J_USERNAME", "NEO4J_PASSWORD"}
	for _, envVar := range requiredEnvVars {
		if os.Getenv(envVar) == "" {
			b.Skipf("Skipping benchmark test. Environment variable %s is not set.", envVar)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, cleanup, err := NewNeo4jDriverFromEnv()
		if err != nil {
			b.Fatalf("Failed to create driver from env: %v", err)
		}
		if cleanup != nil {
			err = cleanup()
			if err != nil {
				b.Fatalf("Failed to clean up driver: %v", err)
			}
		}
	}
}
