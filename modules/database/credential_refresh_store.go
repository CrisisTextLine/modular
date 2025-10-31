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

	// Strip any existing password/token from DSN for backward compatibility
	// This allows applications that previously passed DSN with token placeholders to continue working
	cleanDSN := stripPasswordFromDSN(connConfig.DSN)

	// Extract endpoint from DSN
	endpoint, err := extractEndpointFromDSN(cleanDSN)
	if err != nil {
		return nil, fmt.Errorf("failed to extract endpoint from DSN: %w", err)
	}

	// Extract database name and options from DSN
	dbName, opts, err := extractDatabaseAndOptions(cleanDSN)
	if err != nil {
		return nil, fmt.Errorf("failed to extract database name and options: %w", err)
	}

	// Extract username from DSN if present, otherwise use config
	username := extractUsernameFromDSN(cleanDSN)
	if username == "" {
		username = connConfig.AWSIAMAuth.DBUser
	}

	// Determine driver name and port
	driverName, port := determineDriverAndPort(connConfig.Driver, endpoint)

	// Create AWS RDS store for credential management
	store, err := awsrds.NewStore(&awsrds.Config{
		Credentials: awsConfig.Credentials,
		Endpoint:    endpoint,
		Region:      connConfig.AWSIAMAuth.Region,
		User:        username,
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

// stripPasswordFromDSN removes any password from a DSN for backward compatibility
// This allows applications that previously passed DSN with token placeholders to continue working
func stripPasswordFromDSN(dsn string) string {
	if strings.Contains(dsn, "://") {
		// URL-style DSN (e.g., postgres://user:password@host:port/database)
		// Find user info section
		schemeEnd := strings.Index(dsn, "://")
		if schemeEnd == -1 {
			return dsn
		}

		atIdx := strings.Index(dsn[schemeEnd+3:], "@")
		if atIdx == -1 {
			return dsn // No credentials
		}

		// Extract scheme and credentials section
		scheme := dsn[:schemeEnd+3]
		credentials := dsn[schemeEnd+3 : schemeEnd+3+atIdx]
		remainder := dsn[schemeEnd+3+atIdx:]

		// Check if there's a password (contains colon)
		colonIdx := strings.Index(credentials, ":")
		if colonIdx == -1 {
			return dsn // No password
		}

		// Rebuild DSN without password
		username := credentials[:colonIdx]
		return scheme + username + remainder
	}

	// Key-value style DSN (e.g., host=localhost port=5432 password=token dbname=mydb)
	parts := strings.Fields(dsn)
	var result []string
	for _, part := range parts {
		if !strings.HasPrefix(part, "password=") {
			result = append(result, part)
		}
	}
	return strings.Join(result, " ")
}

// extractUsernameFromDSN extracts the username from a DSN
func extractUsernameFromDSN(dsn string) string {
	if strings.Contains(dsn, "://") {
		// URL-style DSN
		schemeEnd := strings.Index(dsn, "://")
		if schemeEnd == -1 {
			return ""
		}

		atIdx := strings.Index(dsn[schemeEnd+3:], "@")
		if atIdx == -1 {
			return "" // No credentials
		}

		credentials := dsn[schemeEnd+3 : schemeEnd+3+atIdx]

		// Check if there's a colon (username:password format)
		colonIdx := strings.Index(credentials, ":")
		if colonIdx != -1 {
			return credentials[:colonIdx]
		}

		// No colon means just username
		return credentials
	}

	// Key-value style DSN
	parts := strings.Fields(dsn)
	for _, part := range parts {
		if strings.HasPrefix(part, "user=") {
			return strings.TrimPrefix(part, "user=")
		}
	}
	return ""
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
	// Set default port based on driver
	defaultPort := 5432
	driver := "pgx"
	if driverName == "mysql" {
		defaultPort = 3306
		driver = "mysql"
	}
	port := defaultPort

	// Override with explicit port if present
	if colonIdx := strings.LastIndex(endpoint, ":"); colonIdx != -1 {
		if n, err := fmt.Sscanf(endpoint[colonIdx+1:], "%d", &port); err != nil || n != 1 {
			port = defaultPort // Restore default if parsing fails
		}
	}

	return driver, port
}
