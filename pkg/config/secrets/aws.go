package secrets

import (
	"context"
	"encoding/json"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

// AWSConfig holds configuration for AWS Secrets Manager
type AWSConfig struct {
	Region          string `yaml:"region"`
	AccessKeyID     string `yaml:"access_key_id"`
	SecretAccessKey string `yaml:"secret_access_key"`
	SecretName      string `yaml:"secret_name"`
	Endpoint        string `yaml:"endpoint"` // Optional: for LocalStack or custom endpoints
}

// Validate checks if the AWSConfig has all required fields set
func (a AWSConfig) Validate() error {
	if a.Region == "" {
		return errors.New("AWS region is required")
	}
	if a.SecretName == "" {
		return errors.New("AWS secret name is required")
	}
	// AccessKeyID and SecretAccessKey are optional - if not provided, will use IAM role or default credentials
	return nil
}

// CreateClient creates and configures an AWS Secrets Manager client from this config.
// Implements the config.ClientFactory[*secretsmanager.Client] interface.
// Returns *secretsmanager.Client on success, or an error if client creation fails.

func (a AWSConfig) CreateClient() (*secretsmanager.Client, error) {
	ctx := context.Background()

	var cfg aws.Config
	var err error

	// Build config options
	configOpts := []func(*config.LoadOptions) error{
		config.WithRegion(a.Region),
	}

	// Add custom endpoint if provided (for LocalStack or custom endpoints)
	if a.Endpoint != "" {
		configOpts = append(configOpts, config.WithBaseEndpoint(a.Endpoint))
	}

	// Add credentials if provided; otherwise use default credential chain (IAM role, env vars, etc.)
	if a.AccessKeyID != "" && a.SecretAccessKey != "" {
		configOpts = append(configOpts, config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				a.AccessKeyID,
				a.SecretAccessKey,
				"",
			),
		))
	}

	cfg, err = config.LoadDefaultConfig(ctx, configOpts...)

	if err != nil {
		return nil, errors.Wrap(err, "failed to load AWS configuration")
	}

	client := secretsmanager.NewFromConfig(cfg)
	return client, nil
}

// AWSSecretLoader retrieves secrets from AWS Secrets Manager.
// The secret value can be either a string or a JSON object with multiple key-value pairs.
//
// Example usage in config:
//
//	password: ${aws:DATABASE_PASSWORD}  # Reads from configured AWS secret
//
// The AWS secret name is configured when creating the resolver.
type AWSSecretLoader struct {
	client     *secretsmanager.Client
	secretName string
}

// NewAWSSecretLoader creates a new AWS Secrets Manager-based resolver
//
// Parameters:
//   - client: Configured AWS Secrets Manager client
//   - secretName: The name of the secret in AWS Secrets Manager
func NewAWSSecretLoader(client *secretsmanager.Client, secretName string) *AWSSecretLoader {
	return &AWSSecretLoader{
		client:     client,
		secretName: secretName,
	}
}

// Resolve retrieves a secret from AWS Secrets Manager
func (a *AWSSecretLoader) Resolve(key string) (string, error) {
	ctx := context.Background()

	input := &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(a.secretName),
	}

	result, err := a.client.GetSecretValue(ctx, input)
	if err != nil {
		return "", errors.Wrapf(err, "failed to read secret from AWS Secrets Manager: %q", a.secretName)
	}

	if result.SecretString == nil {
		return "", errors.Errorf("secret %q has no string value", a.secretName)
	}

	secretString := *result.SecretString

	// Try to parse as JSON first (for secrets with multiple key-value pairs)
	var secretData map[string]interface{}
	if err := json.Unmarshal([]byte(secretString), &secretData); err == nil {
		// It's a JSON object, extract the requested key
		if value, ok := secretData[key].(string); ok {
			log.Debug().
				Str("secret_name", a.secretName).
				Str("key", key).
				Msg("Retrieved secret from AWS Secrets Manager")
			return value, nil
		}
		return "", errors.Errorf("key %q not found in AWS secret %q", key, a.secretName)
	}

	// Not JSON, treat the entire secret as a single value
	// In this case, the key is ignored and the entire secret value is returned
	log.Debug().
		Str("secret_name", a.secretName).
		Msg("Retrieved secret from AWS Secrets Manager (plain text)")
	return secretString, nil
}
