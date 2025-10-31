package database

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExtractDatabaseAndOptions tests the extraction of database name and options from DSN
func TestExtractDatabaseAndOptions(t *testing.T) {
	tests := []struct {
		name         string
		dsn          string
		expectedDB   string
		expectedOpts map[string]string
		expectError  bool
	}{
		{
			name:         "URL-style DSN with database",
			dsn:          "postgres://user:password@host:5432/mydb",
			expectedDB:   "mydb",
			expectedOpts: map[string]string{},
			expectError:  false,
		},
		{
			name:         "URL-style DSN with database and options",
			dsn:          "postgres://user:password@host:5432/mydb?sslmode=disable&connect_timeout=10",
			expectedDB:   "mydb",
			expectedOpts: map[string]string{"sslmode": "disable", "connect_timeout": "10"},
			expectError:  false,
		},
		{
			name:         "URL-style DSN without database",
			dsn:          "postgres://user:password@host:5432",
			expectedDB:   "",
			expectedOpts: map[string]string{},
			expectError:  false,
		},
		{
			name:         "Key-value style DSN with database",
			dsn:          "host=localhost port=5432 dbname=mydb sslmode=disable",
			expectedDB:   "mydb",
			expectedOpts: map[string]string{"sslmode": "disable"},
			expectError:  false,
		},
		{
			name:         "Key-value style DSN without database",
			dsn:          "host=localhost port=5432 sslmode=disable",
			expectedDB:   "",
			expectedOpts: map[string]string{"sslmode": "disable"},
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, opts, err := extractDatabaseAndOptions(tt.dsn)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedDB, db)
				assert.Equal(t, len(tt.expectedOpts), len(opts))
				for k, v := range tt.expectedOpts {
					assert.Equal(t, v, opts[k], "Option %s mismatch", k)
				}
			}
		})
	}
}

// TestDetermineDriverAndPort tests driver name and port determination
func TestDetermineDriverAndPort(t *testing.T) {
	tests := []struct {
		name           string
		driverName     string
		endpoint       string
		expectedDriver string
		expectedPort   int
	}{
		{
			name:           "Postgres with port",
			driverName:     "postgres",
			endpoint:       "host.example.com:5432",
			expectedDriver: "pgx",
			expectedPort:   5432,
		},
		{
			name:           "Postgres without port",
			driverName:     "postgres",
			endpoint:       "host.example.com",
			expectedDriver: "pgx",
			expectedPort:   5432,
		},
		{
			name:           "MySQL with port",
			driverName:     "mysql",
			endpoint:       "host.example.com:3306",
			expectedDriver: "mysql",
			expectedPort:   3306,
		},
		{
			name:           "MySQL without port",
			driverName:     "mysql",
			endpoint:       "host.example.com",
			expectedDriver: "mysql",
			expectedPort:   3306,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			driver, port := determineDriverAndPort(tt.driverName, tt.endpoint)
			assert.Equal(t, tt.expectedDriver, driver)
			assert.Equal(t, tt.expectedPort, port)
		})
	}
}

// TestCreateDBWithCredentialRefresh_ValidationErrors tests validation errors
func TestCreateDBWithCredentialRefresh_ValidationErrors(t *testing.T) {
	ctx := context.Background()

	t.Run("IAM auth not enabled", func(t *testing.T) {
		config := ConnectionConfig{
			Driver: "postgres",
			DSN:    "postgres://user:password@host:5432/mydb",
		}

		_, err := createDBWithCredentialRefresh(ctx, config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "AWS IAM auth not enabled")
	})

	t.Run("IAM auth enabled but nil config", func(t *testing.T) {
		config := ConnectionConfig{
			Driver:     "postgres",
			DSN:        "postgres://user:password@host:5432/mydb",
			AWSIAMAuth: nil,
		}

		_, err := createDBWithCredentialRefresh(ctx, config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "AWS IAM auth not enabled")
	})

	t.Run("IAM auth config but disabled", func(t *testing.T) {
		config := ConnectionConfig{
			Driver: "postgres",
			DSN:    "postgres://user:password@host:5432/mydb",
			AWSIAMAuth: &AWSIAMAuthConfig{
				Enabled: false,
			},
		}

		_, err := createDBWithCredentialRefresh(ctx, config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "AWS IAM auth not enabled")
	})
}

// TestCreateDBWithCredentialRefresh_IntegrationWithLibrary tests integration with go-db-credential-refresh
// This test validates that we're using the library correctly but doesn't require actual AWS credentials
func TestCreateDBWithCredentialRefresh_IntegrationWithLibrary(t *testing.T) {
	// Skip this test in normal runs as it requires AWS credentials
	// It's here to document the integration pattern
	t.Skip("Requires AWS credentials and RDS instance")

	ctx := context.Background()

	config := ConnectionConfig{
		Driver: "postgres",
		DSN:    "postgres://testuser:placeholder@mydb.region.rds.amazonaws.com:5432/mydb",
		AWSIAMAuth: &AWSIAMAuthConfig{
			Enabled:              true,
			Region:               "us-east-1",
			DBUser:               "testuser",
			TokenRefreshInterval: 600,
			ConnectionTimeout:    5 * time.Second,
		},
		MaxOpenConnections:    10,
		MaxIdleConnections:    5,
		ConnectionMaxLifetime: 1 * time.Hour,
		ConnectionMaxIdleTime: 10 * time.Minute,
	}

	db, err := createDBWithCredentialRefresh(ctx, config)
	require.NoError(t, err, "Should create database connection with credential refresh")
	require.NotNil(t, db, "Database connection should not be nil")

	// Verify connection pool settings were applied
	stats := db.Stats()
	assert.Equal(t, 10, stats.MaxOpenConnections)

	// Clean up
	err = db.Close()
	assert.NoError(t, err)
}

// TestDatabaseService_ConnectWithIAM tests the service Connect method with IAM auth
func TestDatabaseService_ConnectWithIAM(t *testing.T) {
	// Skip this test in normal runs as it requires AWS credentials
	// It's here to document the usage pattern
	t.Skip("Requires AWS credentials and RDS instance")

	config := ConnectionConfig{
		Driver: "postgres",
		DSN:    "postgres://testuser:placeholder@mydb.region.rds.amazonaws.com:5432/mydb",
		AWSIAMAuth: &AWSIAMAuthConfig{
			Enabled:              true,
			Region:               "us-east-1",
			DBUser:               "testuser",
			TokenRefreshInterval: 600,
		},
	}

	service, err := NewDatabaseService(config, &MockLogger{})
	require.NoError(t, err)
	require.NotNil(t, service)

	// Test connection
	err = service.Connect()
	require.NoError(t, err, "Should connect with IAM authentication")

	// Verify database connection is working
	db := service.DB()
	require.NotNil(t, db)

	ctx := context.Background()
	err = service.Ping(ctx)
	assert.NoError(t, err, "Should be able to ping database")

	// Clean up
	err = service.Close()
	assert.NoError(t, err)
}

// TestDatabaseService_AutomaticTokenRefresh documents the automatic token refresh behavior
func TestDatabaseService_AutomaticTokenRefresh(t *testing.T) {
	t.Skip("Requires AWS credentials and RDS instance for real testing")

	// This test documents that go-db-credential-refresh automatically handles:
	// 1. Token expiration detection
	// 2. Automatic token refresh on authentication errors
	// 3. Connection retry with new credentials
	//
	// The library handles this internally through the Connector interface,
	// so we don't need to manually manage token refresh anymore.
	//
	// The old implementation had these issues:
	// - Errors were exposed to database clients during token refresh
	// - Manual connection pool recreation was error-prone
	// - Background goroutines for token refresh could leak
	//
	// The new implementation (go-db-credential-refresh) fixes these by:
	// - Transparently handling token refresh on connection creation
	// - Detecting authentication errors and retrying with fresh tokens
	// - Managing token lifecycle internally without exposing errors to clients
}

// TestDatabaseService_WithoutIAM ensures non-IAM connections still work
func TestDatabaseService_WithoutIAM(t *testing.T) {
	config := ConnectionConfig{
		Driver: "sqlite",
		DSN:    ":memory:",
	}

	service, err := NewDatabaseService(config, &MockLogger{})
	require.NoError(t, err)
	require.NotNil(t, service)

	err = service.Connect()
	require.NoError(t, err, "Should connect without IAM authentication")

	// Verify database connection is working
	db := service.DB()
	require.NotNil(t, db)

	ctx := context.Background()
	err = service.Ping(ctx)
	assert.NoError(t, err, "Should be able to ping database")

	// Test a simple query
	result, err := service.ExecContext(ctx, "CREATE TABLE test (id INTEGER PRIMARY KEY, name TEXT)")
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Clean up
	err = service.Close()
	assert.NoError(t, err)
}

// TestDatabaseService_ConnectionPoolSettings tests that connection pool settings are applied
func TestDatabaseService_ConnectionPoolSettings(t *testing.T) {
	config := ConnectionConfig{
		Driver:                "sqlite",
		DSN:                   ":memory:",
		MaxOpenConnections:    20,
		MaxIdleConnections:    10,
		ConnectionMaxLifetime: 2 * time.Hour,
		ConnectionMaxIdleTime: 30 * time.Minute,
	}

	service, err := NewDatabaseService(config, &MockLogger{})
	require.NoError(t, err)

	err = service.Connect()
	require.NoError(t, err)

	// Verify connection pool settings
	db := service.DB()
	stats := db.Stats()
	assert.Equal(t, 20, stats.MaxOpenConnections)

	err = service.Close()
	assert.NoError(t, err)
}

// TestDatabaseService_IAMConfigValidation tests IAM configuration validation
func TestDatabaseService_IAMConfigValidation(t *testing.T) {
	t.Run("Missing region", func(t *testing.T) {
		// Skip actual AWS connection but test config structure
		config := ConnectionConfig{
			Driver: "postgres",
			DSN:    "postgres://user:password@host:5432/mydb",
			AWSIAMAuth: &AWSIAMAuthConfig{
				Enabled: true,
				Region:  "", // Missing region
				DBUser:  "testuser",
			},
		}

		service, err := NewDatabaseService(config, &MockLogger{})
		require.NoError(t, err) // Service creation should succeed
		require.NotNil(t, service)

		// Connection will fail due to missing region
		err = service.Connect()
		assert.Error(t, err) // Expected to fail without valid AWS config
	})

	t.Run("Missing DB user", func(t *testing.T) {
		config := ConnectionConfig{
			Driver: "postgres",
			DSN:    "postgres://user:password@host:5432/mydb",
			AWSIAMAuth: &AWSIAMAuthConfig{
				Enabled: true,
				Region:  "us-east-1",
				DBUser:  "", // Missing user
			},
		}

		service, err := NewDatabaseService(config, &MockLogger{})
		require.NoError(t, err)
		require.NotNil(t, service)

		// Connection will fail due to missing user
		err = service.Connect()
		assert.Error(t, err)
	})
}

// TestHelperFunctions_StillWork ensures helper functions from old implementation still work
func TestHelperFunctions_StillWork(t *testing.T) {
	t.Run("extractEndpointFromDSN", func(t *testing.T) {
		endpoint, err := extractEndpointFromDSN("postgres://user:password@host.example.com:5432/mydb")
		assert.NoError(t, err)
		assert.Equal(t, "host.example.com:5432", endpoint)
	})

	t.Run("replaceDSNPassword", func(t *testing.T) {
		newDSN, err := replaceDSNPassword("postgres://user:oldpass@host:5432/mydb", "newtoken")
		assert.NoError(t, err)
		assert.Contains(t, newDSN, "newtoken")
		assert.NotContains(t, newDSN, "oldpass")
	})

	t.Run("preprocessDSNForParsing", func(t *testing.T) {
		dsn, err := preprocessDSNForParsing("postgres://user:p@ss!word@host:5432/mydb")
		assert.NoError(t, err)
		assert.NotNil(t, dsn)
	})
}

// Benchmark for connection creation with credential refresh
func BenchmarkCreateDBWithCredentialRefresh(b *testing.B) {
	b.Skip("Requires AWS credentials")
	// This benchmark would measure the overhead of using go-db-credential-refresh
	// vs direct sql.Open, but requires actual AWS infrastructure
}
