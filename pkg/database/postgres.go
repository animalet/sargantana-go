package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresConfig holds configuration options for PostgreSQL connection pool
type PostgresConfig struct {
	Host              string        `yaml:"host"`
	Port              uint16        `yaml:"port"`
	Database          string        `yaml:"database"`
	User              string        `yaml:"user"`
	Password          string        `yaml:"password"`
	SSLMode           string        `yaml:"ssl_mode,omitempty"`            // disable, allow, prefer, require, verify-ca, verify-full
	MaxConns          int32         `yaml:"max_conns,omitempty"`           // Maximum number of connections in the pool
	MinConns          int32         `yaml:"min_conns,omitempty"`           // Minimum number of connections in the pool
	MaxConnLifetime   time.Duration `yaml:"max_conn_lifetime,omitempty"`   // Maximum lifetime of a connection
	MaxConnIdleTime   time.Duration `yaml:"max_conn_idle_time,omitempty"`  // Maximum idle time of a connection
	HealthCheckPeriod time.Duration `yaml:"health_check_period,omitempty"` // Period between health checks
}

// Validate checks if the PostgresConfig has all required fields set
func (p PostgresConfig) Validate() error {
	if p.Host == "" {
		return fmt.Errorf("postgres host must be set and non-empty")
	}
	if p.Port == 0 {
		return fmt.Errorf("postgres port must be set and non-zero")
	}
	if p.Database == "" {
		return fmt.Errorf("postgres database must be set and non-empty")
	}
	if p.User == "" {
		return fmt.Errorf("postgres user must be set and non-empty")
	}
	if p.Password == "" {
		return fmt.Errorf("postgres password must be set and non-empty")
	}

	// Validate SSL mode if provided
	if p.SSLMode != "" {
		validSSLModes := map[string]bool{
			"disable":     true,
			"allow":       true,
			"prefer":      true,
			"require":     true,
			"verify-ca":   true,
			"verify-full": true,
		}
		if !validSSLModes[p.SSLMode] {
			return fmt.Errorf("invalid ssl_mode %q, must be one of: disable, allow, prefer, require, verify-ca, verify-full", p.SSLMode)
		}
	}

	// Validate pool settings
	if p.MaxConns < 0 {
		return fmt.Errorf("max_conns must be non-negative")
	}
	if p.MinConns < 0 {
		return fmt.Errorf("min_conns must be non-negative")
	}
	if p.MaxConns > 0 && p.MinConns > p.MaxConns {
		return fmt.Errorf("min_conns (%d) cannot be greater than max_conns (%d)", p.MinConns, p.MaxConns)
	}
	if p.MaxConnLifetime < 0 {
		return fmt.Errorf("max_conn_lifetime must be non-negative")
	}
	if p.MaxConnIdleTime < 0 {
		return fmt.Errorf("max_conn_idle_time must be non-negative")
	}
	if p.HealthCheckPeriod < 0 {
		return fmt.Errorf("health_check_period must be non-negative")
	}

	return nil
}

// CreateClient creates and configures a PostgreSQL connection pool from this config.
// Implements the config.ClientFactory[*pgxpool.Pool] interface.
// Returns *pgxpool.Pool on success.

func (p PostgresConfig) CreateClient() (*pgxpool.Pool, error) {
	// Build connection string
	connString := p.buildConnectionString()

	// Parse and create pool config
	poolConfig, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse PostgreSQL connection string: %w", err)
	}

	// Apply pool configuration
	if p.MaxConns > 0 {
		poolConfig.MaxConns = p.MaxConns
	}
	if p.MinConns > 0 {
		poolConfig.MinConns = p.MinConns
	}
	if p.MaxConnLifetime > 0 {
		poolConfig.MaxConnLifetime = p.MaxConnLifetime
	}
	if p.MaxConnIdleTime > 0 {
		poolConfig.MaxConnIdleTime = p.MaxConnIdleTime
	}
	if p.HealthCheckPeriod > 0 {
		poolConfig.HealthCheckPeriod = p.HealthCheckPeriod
	}

	// Create the connection pool
	ctx := context.Background()
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create PostgreSQL connection pool: %w", err)
	}

	// Verify connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping PostgreSQL database: %w", err)
	}

	return pool, nil
}

// buildConnectionString creates a PostgreSQL connection string from the config
func (p PostgresConfig) buildConnectionString() string {
	sslMode := p.SSLMode
	if sslMode == "" {
		sslMode = "prefer" // Default to prefer if not specified
	}

	return fmt.Sprintf(
		"host=%s port=%d dbname=%s user=%s password=%s sslmode=%s",
		p.Host,
		p.Port,
		p.Database,
		p.User,
		p.Password,
		sslMode,
	)
}
