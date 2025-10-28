package database

import (
	"strings"
	"testing"
	"time"
)

// TestPostgresConfig_Validate tests the validation logic
func TestPostgresConfig_Validate(t *testing.T) {
	tests := []struct {
		name          string
		config        *PostgresConfig
		errorExpected bool
		errorContains string
	}{
		{
			name: "valid config with all required fields",
			config: &PostgresConfig{
				Host:     "localhost",
				Port:     5432,
				Database: "testdb",
				User:     "testuser",
				Password: "testpass",
			},
			errorExpected: false,
		},
		{
			name: "valid config with optional SSL mode",
			config: &PostgresConfig{
				Host:     "localhost",
				Port:     5432,
				Database: "testdb",
				User:     "testuser",
				Password: "testpass",
				SSLMode:  "require",
			},
			errorExpected: false,
		},
		{
			name: "valid config with pool settings",
			config: &PostgresConfig{
				Host:              "localhost",
				Port:              5432,
				Database:          "testdb",
				User:              "testuser",
				Password:          "testpass",
				MaxConns:          10,
				MinConns:          2,
				MaxConnLifetime:   time.Hour,
				MaxConnIdleTime:   30 * time.Minute,
				HealthCheckPeriod: time.Minute,
			},
			errorExpected: false,
		},
		{
			name: "missing host",
			config: &PostgresConfig{
				Port:     5432,
				Database: "testdb",
				User:     "testuser",
				Password: "testpass",
			},
			errorExpected: true,
			errorContains: "host must be set",
		},
		{
			name: "missing port",
			config: &PostgresConfig{
				Host:     "localhost",
				Database: "testdb",
				User:     "testuser",
				Password: "testpass",
			},
			errorExpected: true,
			errorContains: "port must be set",
		},
		{
			name: "missing database",
			config: &PostgresConfig{
				Host:     "localhost",
				Port:     5432,
				User:     "testuser",
				Password: "testpass",
			},
			errorExpected: true,
			errorContains: "database must be set",
		},
		{
			name: "missing user",
			config: &PostgresConfig{
				Host:     "localhost",
				Port:     5432,
				Database: "testdb",
				Password: "testpass",
			},
			errorExpected: true,
			errorContains: "user must be set",
		},
		{
			name: "missing password",
			config: &PostgresConfig{
				Host:     "localhost",
				Port:     5432,
				Database: "testdb",
				User:     "testuser",
			},
			errorExpected: true,
			errorContains: "password must be set",
		},
		{
			name: "invalid SSL mode",
			config: &PostgresConfig{
				Host:     "localhost",
				Port:     5432,
				Database: "testdb",
				User:     "testuser",
				Password: "testpass",
				SSLMode:  "invalid-mode",
			},
			errorExpected: true,
			errorContains: "invalid ssl_mode",
		},
		{
			name: "negative max_conns",
			config: &PostgresConfig{
				Host:     "localhost",
				Port:     5432,
				Database: "testdb",
				User:     "testuser",
				Password: "testpass",
				MaxConns: -1,
			},
			errorExpected: true,
			errorContains: "max_conns must be non-negative",
		},
		{
			name: "negative min_conns",
			config: &PostgresConfig{
				Host:     "localhost",
				Port:     5432,
				Database: "testdb",
				User:     "testuser",
				Password: "testpass",
				MinConns: -1,
			},
			errorExpected: true,
			errorContains: "min_conns must be non-negative",
		},
		{
			name: "min_conns greater than max_conns",
			config: &PostgresConfig{
				Host:     "localhost",
				Port:     5432,
				Database: "testdb",
				User:     "testuser",
				Password: "testpass",
				MaxConns: 5,
				MinConns: 10,
			},
			errorExpected: true,
			errorContains: "min_conns",
		},
		{
			name: "negative max_conn_lifetime",
			config: &PostgresConfig{
				Host:            "localhost",
				Port:            5432,
				Database:        "testdb",
				User:            "testuser",
				Password:        "testpass",
				MaxConnLifetime: -time.Hour,
			},
			errorExpected: true,
			errorContains: "max_conn_lifetime must be non-negative",
		},
		{
			name: "negative max_conn_idle_time",
			config: &PostgresConfig{
				Host:            "localhost",
				Port:            5432,
				Database:        "testdb",
				User:            "testuser",
				Password:        "testpass",
				MaxConnIdleTime: -time.Minute,
			},
			errorExpected: true,
			errorContains: "max_conn_idle_time must be non-negative",
		},
		{
			name: "negative health_check_period",
			config: &PostgresConfig{
				Host:              "localhost",
				Port:              5432,
				Database:          "testdb",
				User:              "testuser",
				Password:          "testpass",
				HealthCheckPeriod: -time.Second,
			},
			errorExpected: true,
			errorContains: "health_check_period must be non-negative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.errorExpected {
				if err == nil {
					t.Errorf("Expected error but got nil")
				} else if !strings.Contains(err.Error(), tt.errorContains) {
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

// TestPostgresConfig_BuildConnectionString tests connection string generation
func TestPostgresConfig_BuildConnectionString(t *testing.T) {
	tests := []struct {
		name           string
		config         *PostgresConfig
		expectedString string
	}{
		{
			name: "basic connection string",
			config: &PostgresConfig{
				Host:     "localhost",
				Port:     5432,
				Database: "testdb",
				User:     "testuser",
				Password: "testpass",
			},
			expectedString: "host=localhost port=5432 dbname=testdb user=testuser password=testpass sslmode=prefer",
		},
		{
			name: "with SSL mode disabled",
			config: &PostgresConfig{
				Host:     "db.example.com",
				Port:     5433,
				Database: "proddb",
				User:     "admin",
				Password: "secret",
				SSLMode:  "disable",
			},
			expectedString: "host=db.example.com port=5433 dbname=proddb user=admin password=secret sslmode=disable",
		},
		{
			name: "with SSL mode required",
			config: &PostgresConfig{
				Host:     "secure-db.example.com",
				Port:     5432,
				Database: "securedb",
				User:     "secureuser",
				Password: "securepass",
				SSLMode:  "require",
			},
			expectedString: "host=secure-db.example.com port=5432 dbname=securedb user=secureuser password=securepass sslmode=require",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			connString := tt.config.buildConnectionString()
			if connString != tt.expectedString {
				t.Errorf("Expected connection string:\n%s\nGot:\n%s", tt.expectedString, connString)
			}
		})
	}
}

// TestPostgresConfig_ValidSSLModes tests all valid SSL modes
func TestPostgresConfig_ValidSSLModes(t *testing.T) {
	validModes := []string{"disable", "allow", "prefer", "require", "verify-ca", "verify-full"}

	for _, mode := range validModes {
		t.Run("ssl_mode_"+mode, func(t *testing.T) {
			config := &PostgresConfig{
				Host:     "localhost",
				Port:     5432,
				Database: "testdb",
				User:     "testuser",
				Password: "testpass",
				SSLMode:  mode,
			}

			err := config.Validate()
			if err != nil {
				t.Errorf("Expected SSL mode %q to be valid, got error: %v", mode, err)
			}
		})
	}
}

// TestPostgresConfig_CreateClient tests the CreateClient method
func TestPostgresConfig_CreateClient(t *testing.T) {
	tests := []struct {
		name    string
		config  *PostgresConfig
		wantErr bool
	}{
		{
			name: "valid config creates pool",
			config: &PostgresConfig{
				Host:     "localhost",
				Port:     5432,
				Database: "my_blog_db",
				User:     "user",
				Password: "password",
				SSLMode:  "disable",
			},
			wantErr: false,
		},
		{
			name: "invalid config returns error",
			config: &PostgresConfig{
				Host:     "",
				Port:     5432,
				Database: "testdb",
				User:     "testuser",
				Password: "testpass",
			},
			wantErr: true,
		},
		{
			name: "valid config with all pool settings",
			config: &PostgresConfig{
				Host:              "localhost",
				Port:              5432,
				Database:          "my_blog_db",
				User:              "user",
				Password:          "password",
				SSLMode:           "disable",
				MaxConns:          20,
				MinConns:          5,
				MaxConnLifetime:   time.Hour,
				MaxConnIdleTime:   30 * time.Minute,
				HealthCheckPeriod: 2 * time.Minute,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool, err := tt.config.CreateClient()
			if tt.wantErr {
				if err == nil {
					t.Errorf("CreateClient() expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("CreateClient() unexpected error = %v", err)
				}
				if pool == nil {
					t.Errorf("CreateClient() returned nil pool")
				}
				// Verify pool configuration was applied
				if pool != nil {
					cfg := pool.Config()
					if tt.config.MaxConns > 0 && cfg.MaxConns != tt.config.MaxConns {
						t.Errorf("Expected MaxConns %d, got %d", tt.config.MaxConns, cfg.MaxConns)
					}
					if tt.config.MinConns > 0 && cfg.MinConns != tt.config.MinConns {
						t.Errorf("Expected MinConns %d, got %d", tt.config.MinConns, cfg.MinConns)
					}
					if tt.config.MaxConnLifetime > 0 && cfg.MaxConnLifetime != tt.config.MaxConnLifetime {
						t.Errorf("Expected MaxConnLifetime %v, got %v", tt.config.MaxConnLifetime, cfg.MaxConnLifetime)
					}
					if tt.config.MaxConnIdleTime > 0 && cfg.MaxConnIdleTime != tt.config.MaxConnIdleTime {
						t.Errorf("Expected MaxConnIdleTime %v, got %v", tt.config.MaxConnIdleTime, cfg.MaxConnIdleTime)
					}
					if tt.config.HealthCheckPeriod > 0 && cfg.HealthCheckPeriod != tt.config.HealthCheckPeriod {
						t.Errorf("Expected HealthCheckPeriod %v, got %v", tt.config.HealthCheckPeriod, cfg.HealthCheckPeriod)
					}
					pool.Close()
				}
			}
		})
	}
}
