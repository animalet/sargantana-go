package secrets

import (
	"net/http"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

// TestLocalStackHealthCheck verifies that the Docker LocalStack container is healthy
func TestLocalStackHealthCheck(t *testing.T) {
	localStackAddr := "http://localhost:4566"
	resp, err := http.Get(localStackAddr + "/_localstack/health")
	if err != nil {
		t.Fatalf("Failed to check LocalStack health: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		t.Errorf("Expected LocalStack to be healthy (status 200), got status %d", resp.StatusCode)
	}
}

// createTestAWSClient creates an AWS Secrets Manager client for testing with LocalStack
func createTestAWSClient(endpoint, region, accessKey, secretKey, secretName string) (*secretsmanager.Client, error) {
	awsCfg := &AWSConfig{
		Region:          region,
		AccessKeyID:     accessKey,
		SecretAccessKey: secretKey,
		SecretName:      secretName,
		Endpoint:        endpoint,
	}
	return createAWSClient(awsCfg)
}

// TestAWSResolver_Success tests successful secret retrieval from LocalStack
func TestAWSResolver_Success(t *testing.T) {
	client, err := createTestAWSClient(
		"http://localhost:4566",
		"us-east-1",
		"test",
		"test",
		"sargantana/test",
	)
	if err != nil {
		t.Fatalf("Failed to create AWS client: %v", err)
	}

	resolver := NewAWSResolver(client, "sargantana/test")

	// Test retrieving GOOGLE_KEY
	googleKey, err := resolver.Resolve("GOOGLE_KEY")
	if err != nil {
		t.Fatalf("Resolve failed for GOOGLE_KEY: %v", err)
	}
	if googleKey != "test-google-key" {
		t.Errorf("Expected 'test-google-key', got '%s'", googleKey)
	}

	// Test retrieving SESSION_SECRET
	sessionSecret, err := resolver.Resolve("SESSION_SECRET")
	if err != nil {
		t.Fatalf("Resolve failed for SESSION_SECRET: %v", err)
	}
	if sessionSecret != "test-session-secret-that-is-long-enough" {
		t.Errorf("Expected 'test-session-secret-that-is-long-enough', got '%s'", sessionSecret)
	}
}

// TestAWSResolver_PlainText tests retrieval of plain text secrets (non-JSON)
func TestAWSResolver_PlainText(t *testing.T) {
	client, err := createTestAWSClient(
		"http://localhost:4566",
		"us-east-1",
		"test",
		"test",
		"sargantana/plain-secret",
	)
	if err != nil {
		t.Fatalf("Failed to create AWS client: %v", err)
	}

	resolver := NewAWSResolver(client, "sargantana/plain-secret")

	// For plain text secrets, the key parameter is ignored and the entire value is returned
	plainValue, err := resolver.Resolve("ANY_KEY")
	if err != nil {
		t.Fatalf("Resolve failed for plain text secret: %v", err)
	}
	if plainValue != "plain-text-secret-value" {
		t.Errorf("Expected 'plain-text-secret-value', got '%s'", plainValue)
	}
}

// TestAWSResolver_NonexistentSecret tests reading from nonexistent AWS secret
func TestAWSResolver_NonexistentSecret(t *testing.T) {
	client, err := createTestAWSClient(
		"http://localhost:4566",
		"us-east-1",
		"test",
		"test",
		"nonexistent/secret",
	)
	if err != nil {
		t.Fatalf("Failed to create AWS client: %v", err)
	}

	resolver := NewAWSResolver(client, "nonexistent/secret")

	_, err = resolver.Resolve("SOME_KEY")
	if err == nil {
		t.Fatal("Expected error when reading from nonexistent secret")
	}
	if !strings.Contains(err.Error(), "failed to read secret from AWS Secrets Manager") {
		t.Errorf("Expected 'failed to read secret' error, got: %v", err)
	}
}

// TestAWSResolver_NonexistentKey tests reading nonexistent key from AWS secret
func TestAWSResolver_NonexistentKey(t *testing.T) {
	client, err := createTestAWSClient(
		"http://localhost:4566",
		"us-east-1",
		"test",
		"test",
		"sargantana/test",
	)
	if err != nil {
		t.Fatalf("Failed to create AWS client: %v", err)
	}

	resolver := NewAWSResolver(client, "sargantana/test")

	_, err = resolver.Resolve("NONEXISTENT_KEY")
	if err == nil {
		t.Fatal("Expected error when reading nonexistent key")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Expected 'not found' error, got: %v", err)
	}
}

// Note: LocalStack doesn't validate credentials, so we can't test invalid credentials
// In production AWS, invalid credentials would return an authentication error

// TestAWSResolver_Name tests the Name method
func TestAWSResolver_Name(t *testing.T) {
	client, err := createTestAWSClient(
		"http://localhost:4566",
		"us-east-1",
		"test",
		"test",
		"test/secret",
	)
	if err != nil {
		t.Fatalf("Failed to create AWS client: %v", err)
	}

	resolver := NewAWSResolver(client, "test/secret")
	if resolver.Name() != "AWS Secrets Manager" {
		t.Errorf("Expected name 'AWS Secrets Manager', got '%s'", resolver.Name())
	}
}

// TestAWSConfig_Validate tests AWSConfig validation
func TestAWSConfig_Validate(t *testing.T) {
	tests := []struct {
		name          string
		config        *AWSConfig
		errorExpected bool
		errorContains string
	}{
		{
			name: "valid config with credentials",
			config: &AWSConfig{
				Region:          "us-east-1",
				AccessKeyID:     "test",
				SecretAccessKey: "test",
				SecretName:      "test/secret",
			},
			errorExpected: false,
		},
		{
			name: "valid config without credentials",
			config: &AWSConfig{
				Region:     "us-east-1",
				SecretName: "test/secret",
			},
			errorExpected: false,
		},
		{
			name: "valid config with endpoint",
			config: &AWSConfig{
				Region:          "us-east-1",
				AccessKeyID:     "test",
				SecretAccessKey: "test",
				SecretName:      "test/secret",
				Endpoint:        "http://localhost:4566",
			},
			errorExpected: false,
		},
		{
			name: "missing region",
			config: &AWSConfig{
				Region:          "",
				AccessKeyID:     "test",
				SecretAccessKey: "test",
				SecretName:      "test/secret",
			},
			errorExpected: true,
			errorContains: "AWS region is required",
		},
		{
			name: "missing secret name",
			config: &AWSConfig{
				Region:          "us-east-1",
				AccessKeyID:     "test",
				SecretAccessKey: "test",
				SecretName:      "",
			},
			errorExpected: true,
			errorContains: "AWS secret name is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.errorExpected {
				if err == nil {
					t.Error("Expected error but got nil")
				} else if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error containing %q, got: %v", tt.errorContains, err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

// TestCreateAWSClient_Success tests successful AWS client creation
func TestCreateAWSClient_Success(t *testing.T) {
	awsCfg := &AWSConfig{
		Region:          "us-east-1",
		AccessKeyID:     "test",
		SecretAccessKey: "test",
		SecretName:      "test/secret",
		Endpoint:        "http://localhost:4566",
	}

	client, err := createAWSClient(awsCfg)
	if err != nil {
		t.Fatalf("createAWSClient failed: %v", err)
	}

	if client == nil {
		t.Fatal("AWS client should not be nil")
	}
}

// TestCreateAWSClient_InvalidConfig tests with invalid AWS configuration
func TestCreateAWSClient_InvalidConfig(t *testing.T) {
	awsCfg := &AWSConfig{
		Region:          "",
		AccessKeyID:     "",
		SecretAccessKey: "",
		SecretName:      "",
	}

	_, err := createAWSClient(awsCfg)
	if err == nil {
		t.Error("Expected error with invalid AWS configuration, got nil")
	}
	if err != nil && !strings.Contains(err.Error(), "AWS region is required") {
		t.Errorf("Expected region validation error, got: %v", err)
	}
}

// TestCreateAWSClient_WithoutEndpoint tests client creation without custom endpoint
func TestCreateAWSClient_WithoutEndpoint(t *testing.T) {
	awsCfg := &AWSConfig{
		Region:          "us-east-1",
		AccessKeyID:     "test",
		SecretAccessKey: "test",
		SecretName:      "test/secret",
		// No Endpoint specified - should use default AWS endpoints
	}

	client, err := createAWSClient(awsCfg)
	if err != nil {
		t.Fatalf("createAWSClient without endpoint failed: %v", err)
	}

	if client == nil {
		t.Fatal("AWS client should not be nil")
	}
}

// TestAWSConfig_CreateClient tests the ClientFactory pattern
func TestAWSConfig_CreateClient(t *testing.T) {
	awsCfg := &AWSConfig{
		Region:          "us-east-1",
		AccessKeyID:     "test",
		SecretAccessKey: "test",
		SecretName:      "test/secret",
		Endpoint:        "http://localhost:4566",
	}

	// Test using ClientFactory interface
	client, err := awsCfg.CreateClient()
	if err != nil {
		t.Fatalf("CreateClient failed: %v", err)
	}

	if client == nil {
		t.Fatal("Client should not be nil")
	}
}

// TestAWSPropertyResolution_Success tests property resolution using AWS resolver with LocalStack
func TestAWSPropertyResolution_Success(t *testing.T) {
	// Set up AWS resolver using docker compose LocalStack instance
	awsCfg := &AWSConfig{
		Region:          "us-east-1",
		AccessKeyID:     "test",
		SecretAccessKey: "test",
		SecretName:      "sargantana/test",
		Endpoint:        "http://localhost:4566",
	}

	client, err := awsCfg.CreateClient()
	if err != nil {
		t.Fatalf("Failed to create AWS client: %v", err)
	}

	awsResolver := NewAWSResolver(client, awsCfg.SecretName)

	// Register the resolver
	Register("aws", awsResolver)
	defer Unregister("aws")

	// Test resolving a property using aws: prefix
	result, err := Resolve("aws:GOOGLE_KEY")
	if err != nil {
		t.Fatalf("Failed to resolve aws:GOOGLE_KEY: %v", err)
	}

	expected := "test-google-key"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}

	// Test resolving another property
	sessionSecret, err := Resolve("aws:SESSION_SECRET")
	if err != nil {
		t.Fatalf("Failed to resolve aws:SESSION_SECRET: %v", err)
	}

	expectedSecret := "test-session-secret-that-is-long-enough"
	if sessionSecret != expectedSecret {
		t.Errorf("Expected '%s', got '%s'", expectedSecret, sessionSecret)
	}
}

// TestAWSPropertyResolution_PlainText tests property resolution with plain text secret
func TestAWSPropertyResolution_PlainText(t *testing.T) {
	awsCfg := &AWSConfig{
		Region:          "us-east-1",
		AccessKeyID:     "test",
		SecretAccessKey: "test",
		SecretName:      "sargantana/plain-secret",
		Endpoint:        "http://localhost:4566",
	}

	client, err := awsCfg.CreateClient()
	if err != nil {
		t.Fatalf("Failed to create AWS client: %v", err)
	}

	awsResolver := NewAWSResolver(client, awsCfg.SecretName)
	Register("aws", awsResolver)
	defer Unregister("aws")

	// For plain text secrets, any key returns the entire value
	result, err := Resolve("aws:ANY_KEY")
	if err != nil {
		t.Fatalf("Failed to resolve aws:ANY_KEY from plain text secret: %v", err)
	}

	expected := "plain-text-secret-value"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

// TestAWSPropertyResolution_NonexistentKey tests property resolution with nonexistent key
func TestAWSPropertyResolution_NonexistentKey(t *testing.T) {
	awsCfg := &AWSConfig{
		Region:          "us-east-1",
		AccessKeyID:     "test",
		SecretAccessKey: "test",
		SecretName:      "sargantana/test",
		Endpoint:        "http://localhost:4566",
	}

	client, err := awsCfg.CreateClient()
	if err != nil {
		t.Fatalf("Failed to create AWS client: %v", err)
	}

	awsResolver := NewAWSResolver(client, awsCfg.SecretName)
	Register("aws", awsResolver)
	defer Unregister("aws")

	_, err = Resolve("aws:NONEXISTENT_KEY")
	if err == nil {
		t.Fatal("Expected error when resolving nonexistent AWS key")
	}

	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Expected 'not found' error, got: %v", err)
	}
}

// Note: LocalStack doesn't validate credentials in property resolution tests either
// In production AWS, invalid credentials would cause authentication errors

// TestAWSPropertyResolution_NoResolverRegistered tests behavior when no resolver is registered
func TestAWSPropertyResolution_NoResolverRegistered(t *testing.T) {
	// Make sure aws resolver is not registered
	Unregister("aws")

	_, err := Resolve("aws:SOME_KEY")
	if err == nil {
		t.Fatal("Expected error when aws resolver is not registered")
	}

	if !strings.Contains(err.Error(), "no resolver registered") {
		t.Errorf("Expected 'no resolver registered' error, got: %v", err)
	}
}
