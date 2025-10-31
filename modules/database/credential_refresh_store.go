package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/davepgreene/go-db-credential-refresh/driver"
	"github.com/davepgreene/go-db-credential-refresh/store/awsrds"
)

var (
	ErrAWSIAMAuthNotEnabled = errors.New("AWS IAM auth not enabled")
	ErrInvalidDSNFormat     = errors.New("invalid DSN format")
)

// createDBWithCredentialRefresh creates a database connection using go-db-credential-refresh
// This automatically handles token refresh and connection recreation on auth errors
func createDBWithCredentialRefresh(ctx context.Context, connConfig ConnectionConfig) (*sql.DB, error) {
	if connConfig.AWSIAMAuth == nil || !connConfig.AWSIAMAuth.Enabled {
		return nil, ErrAWSIAMAuthNotEnabled
	}

	// Load AWS configuration
	awsConfig, err := config.LoadDefaultConfig(ctx, config.WithRegion(connConfig.AWSIAMAuth.Region))
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Extract endpoint from DSN
	endpoint, err := extractEndpointFromDSN(connConfig.DSN)
	if err != nil {
		return nil, fmt.Errorf("failed to extract endpoint from DSN: %w", err)
	}

	// Extract database name and options from DSN
	dbName, opts, err := extractDatabaseAndOptions(connConfig.DSN)
	if err != nil {
		return nil, fmt.Errorf("failed to extract database name and options: %w", err)
	}

	// Determine driver name and port
	driverName, port := determineDriverAndPort(connConfig.Driver, endpoint)

	// Create AWS RDS store for credential management
	store, err := awsrds.NewStore(&awsrds.Config{
		Credentials: awsConfig.Credentials,
		Endpoint:    endpoint,
		Region:      connConfig.AWSIAMAuth.Region,
		User:        connConfig.AWSIAMAuth.DBUser,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS RDS store: %w", err)
	}

	// Extract hostname from endpoint (remove port)
	hostname := endpoint
	if colonIdx := strings.LastIndex(endpoint, ":"); colonIdx != -1 {
		hostname = endpoint[:colonIdx]
	}

	// Create connector configuration
	cfg := &driver.Config{
		Host:    hostname,
		Port:    port,
		DB:      dbName,
		Opts:    opts,
		Retries: 1, // Retry once on auth failure
	}

	// Create connector with credential refresh
	connector, err := driver.NewConnector(store, driverName, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create connector: %w", err)
	}

	// Open database using the connector
	db := sql.OpenDB(connector)

	// Configure connection pool
	configureConnectionPool(db, connConfig)

	return db, nil
}

// configureConnectionPool applies connection pool settings to a database connection
func configureConnectionPool(db *sql.DB, config ConnectionConfig) {
	if config.MaxOpenConnections > 0 {
		db.SetMaxOpenConns(config.MaxOpenConnections)
	}
	if config.MaxIdleConnections > 0 {
		db.SetMaxIdleConns(config.MaxIdleConnections)
	}
	if config.ConnectionMaxLifetime > 0 {
		db.SetConnMaxLifetime(config.ConnectionMaxLifetime)
	}
	if config.ConnectionMaxIdleTime > 0 {
		db.SetConnMaxIdleTime(config.ConnectionMaxIdleTime)
	}
}

// extractDatabaseAndOptions extracts the database name and connection options from a DSN
func extractDatabaseAndOptions(dsn string) (string, map[string]string, error) {
	opts := make(map[string]string)

	if strings.Contains(dsn, "://") {
		// URL-style DSN (e.g., postgres://user:password@host:port/database?option=value)
		// Parse URL to get path (database) and query params (options)
		parts := strings.Split(dsn, "://")
		if len(parts) != 2 {
			return "", nil, ErrInvalidDSNFormat
		}

		remainder := parts[1]

		// Find database name (after last / before ?)
		dbStart := strings.LastIndex(remainder, "/")
		if dbStart == -1 {
			return "", opts, nil // No database specified
		}

		dbPart := remainder[dbStart+1:]
		dbName := dbPart

		// Extract query parameters if present
		if qIdx := strings.Index(dbPart, "?"); qIdx != -1 {
			dbName = dbPart[:qIdx]
			queryString := dbPart[qIdx+1:]

			// Parse query parameters
			for _, pair := range strings.Split(queryString, "&") {
				if kv := strings.SplitN(pair, "=", 2); len(kv) == 2 {
					opts[kv[0]] = kv[1]
				}
			}
		}

		return dbName, opts, nil
	}

	// Key-value style DSN (e.g., host=localhost port=5432 dbname=mydb sslmode=disable)
	parts := strings.Fields(dsn)
	dbName := ""

	for _, part := range parts {
		if strings.HasPrefix(part, "dbname=") {
			dbName = strings.TrimPrefix(part, "dbname=")
		} else if !strings.HasPrefix(part, "host=") &&
			!strings.HasPrefix(part, "port=") &&
			!strings.HasPrefix(part, "user=") &&
			!strings.HasPrefix(part, "password=") {
			// Extract as option
			if kv := strings.SplitN(part, "=", 2); len(kv) == 2 {
				opts[kv[0]] = kv[1]
			}
		}
	}

	return dbName, opts, nil
}

// determineDriverAndPort determines the correct driver name for go-db-credential-refresh
// and extracts the port from the endpoint
func determineDriverAndPort(driverName string, endpoint string) (string, int) {
	port := 5432 // Default PostgreSQL port

	// Extract port from endpoint if present
	if colonIdx := strings.LastIndex(endpoint, ":"); colonIdx != -1 {
		// Try to parse the port - if parsing fails, the default port will be used
		// This is safe because invalid ports will be caught by the database driver
		_, _ = fmt.Sscanf(endpoint[colonIdx+1:], "%d", &port)
	}

	// Map driver names to go-db-credential-refresh driver names
	// Note: Port defaults are only applied when no port was extracted from the endpoint
	switch driverName {
	case "postgres":
		// Use pgx for postgres driver (better performance and features)
		// Port default of 5432 is already set above
		return "pgx", port
	case "mysql":
		// Only set MySQL default port if we're still using the PostgreSQL default
		// This means no port was explicitly provided in the endpoint
		if port == 5432 {
			port = 3306 // Default MySQL port
		}
		return "mysql", port
	default:
		// Default to pgx for unknown drivers
		return "pgx", port
	}
}
