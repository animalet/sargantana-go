package database

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"os"
	"time"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// MongoDBConfig holds configuration for MongoDB connection
type MongoDBConfig struct {
	// URI is the MongoDB connection string
	// Example: "mongodb://localhost:27017" or "mongodb://user:pass@host:27017/dbname"
	URI string `yaml:"uri"`

	// Database name to use
	Database string `yaml:"database"`

	// Username for authentication (optional if included in URI)
	Username string `yaml:"username"`

	// Password for authentication (optional if included in URI)
	Password string `yaml:"password"`

	// AuthSource is the database name for authentication
	// Default: "admin" if not specified
	AuthSource string `yaml:"auth_source"`

	// TLS configuration (optional)
	TLS *MongoDBTLSConfig `yaml:"tls"`

	// ConnectTimeout is the maximum time to wait for connection
	// Default: 10s if not specified
	ConnectTimeout time.Duration `yaml:"connect_timeout"`

	// MaxPoolSize is the maximum number of connections in the pool
	// Default: 100 if not specified
	MaxPoolSize uint64 `yaml:"max_pool_size"`

	// MinPoolSize is the minimum number of connections in the pool
	// Default: 0 if not specified
	MinPoolSize uint64 `yaml:"min_pool_size"`
}

// MongoDBTLSConfig holds TLS configuration for MongoDB.
// The presence of this configuration block enables TLS - there is no separate "enabled" flag.
type MongoDBTLSConfig struct {
	// InsecureSkipVerify skips certificate verification (not recommended for production)
	InsecureSkipVerify bool `yaml:"insecure_skip_verify"`

	// CertFile is the path to the client certificate file
	CertFile string `yaml:"cert_file"`

	// KeyFile is the path to the client private key file
	KeyFile string `yaml:"key_file"`

	// CAFile is the path to the CA certificate file
	CAFile string `yaml:"ca_file"`
}

// Validate checks if the MongoDBConfig has all required fields set
func (m MongoDBConfig) Validate() error {
	if m.URI == "" {
		return errors.New("MongoDB URI is required")
	}

	if m.Database == "" {
		return errors.New("MongoDB database name is required")
	}

	if m.ConnectTimeout < 0 {
		return errors.New("connect_timeout cannot be negative")
	}

	if m.MinPoolSize > m.MaxPoolSize && m.MaxPoolSize > 0 {
		return errors.New("min_pool_size cannot be greater than max_pool_size")
	}

	// Validate TLS configuration if provided
	// The presence of TLS config block means TLS is enabled
	if m.TLS != nil {
		if m.TLS.CertFile != "" && m.TLS.KeyFile == "" {
			return errors.New("key_file is required when cert_file is specified")
		}
		if m.TLS.KeyFile != "" && m.TLS.CertFile == "" {
			return errors.New("cert_file is required when key_file is specified")
		}

		// Check file existence
		if m.TLS.CertFile != "" {
			if _, err := os.Stat(m.TLS.CertFile); os.IsNotExist(err) {
				return errors.Errorf("cert_file does not exist: %s", m.TLS.CertFile)
			}
		}
		if m.TLS.KeyFile != "" {
			if _, err := os.Stat(m.TLS.KeyFile); os.IsNotExist(err) {
				return errors.Errorf("key_file does not exist: %s", m.TLS.KeyFile)
			}
		}
		if m.TLS.CAFile != "" {
			if _, err := os.Stat(m.TLS.CAFile); os.IsNotExist(err) {
				return errors.Errorf("ca_file does not exist: %s", m.TLS.CAFile)
			}
		}
	}

	return nil
}

// CreateClient creates and configures a MongoDB client from this config.
// Implements the config.ClientFactory[*mongo.Client] interface.
// Returns *mongo.Client on success, or an error if creation fails.

func (m MongoDBConfig) CreateClient() (*mongo.Client, error) {
	clientOpts := options.Client().ApplyURI(m.URI)

	// Set authentication if provided
	if m.Username != "" || m.Password != "" {
		authSource := m.AuthSource
		if authSource == "" {
			authSource = "admin"
		}
		credential := options.Credential{
			Username:   m.Username,
			Password:   m.Password,
			AuthSource: authSource,
		}
		clientOpts.SetAuth(credential)
	}

	// Set connection timeout
	connectTimeout := m.ConnectTimeout
	if connectTimeout == 0 {
		connectTimeout = 10 * time.Second
	}
	clientOpts.SetConnectTimeout(connectTimeout)

	// Set pool size limits
	if m.MaxPoolSize > 0 {
		clientOpts.SetMaxPoolSize(m.MaxPoolSize)
	} else {
		clientOpts.SetMaxPoolSize(100)
	}

	if m.MinPoolSize > 0 {
		clientOpts.SetMinPoolSize(m.MinPoolSize)
	}

	// Configure TLS if TLS config is provided
	if m.TLS != nil {
		tlsConfig, err := m.buildTLSConfig()
		if err != nil {
			return nil, errors.Wrap(err, "failed to build TLS configuration")
		}
		clientOpts.SetTLSConfig(tlsConfig)
	}

	// Create the MongoDB client
	ctx, cancel := context.WithTimeout(context.Background(), connectTimeout)
	defer cancel()

	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to MongoDB")
	}

	// Verify connection with ping
	pingCtx, pingCancel := context.WithTimeout(context.Background(), connectTimeout)
	defer pingCancel()

	if err := client.Ping(pingCtx, readpref.Primary()); err != nil {
		// Close the client if ping fails
		disconnectCtx, disconnectCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer disconnectCancel()
		if disconnectErr := client.Disconnect(disconnectCtx); disconnectErr != nil {
			// Log or wrap both errors
			return nil, errors.Wrapf(err, "failed to ping MongoDB (disconnect also failed: %v)", disconnectErr)
		}
		return nil, errors.Wrap(err, "failed to ping MongoDB")
	}

	return client, nil
}

// buildTLSConfig creates a TLS configuration from the MongoDB TLS settings.
// Returns nil if no TLS config is provided.
func (m MongoDBConfig) buildTLSConfig() (*tls.Config, error) {
	if m.TLS == nil {
		return nil, nil
	}

	// #nosec G402 -- InsecureSkipVerify is a configurable option for dev/test environments
	tlsConfig := &tls.Config{
		InsecureSkipVerify: m.TLS.InsecureSkipVerify,
	}

	// Load client certificate and key
	if m.TLS.CertFile != "" && m.TLS.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(m.TLS.CertFile, m.TLS.KeyFile)
		if err != nil {
			return nil, errors.Wrap(err, "failed to load client certificate")
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	// Load CA certificate
	if m.TLS.CAFile != "" {
		caCert, err := os.ReadFile(m.TLS.CAFile)
		if err != nil {
			return nil, errors.Wrap(err, "failed to read CA certificate")
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, errors.New("failed to parse CA certificate")
		}
		tlsConfig.RootCAs = caCertPool
	}

	return tlsConfig, nil
}
