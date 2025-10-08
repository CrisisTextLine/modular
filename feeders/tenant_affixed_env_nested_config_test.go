package feeders

import (
	"os"
	"testing"
)

// Tests for deeply nested configuration structures to ensure prefix/suffix
// preservation works at all levels of nesting

// TestTenantAffixedEnvFeeder_NestedConfig_TwoLevels tests a config with
// one level of nested structs
func TestTenantAffixedEnvFeeder_NestedConfig_TwoLevels(t *testing.T) {
	// Set up env vars for nested config
	// Note: Env tags should include context for clarity (e.g., DATABASE_SSL_ENABLED)
	os.Setenv("CTL_DATABASE_HOST", "db.example.com")
	os.Setenv("CTL_DATABASE_PORT", "5432")
	os.Setenv("CTL_DATABASE_SSL_ENABLED", "true")
	os.Setenv("CTL_DATABASE_SSL_CERT_PATH", "/etc/ssl/cert.pem")
	defer func() {
		os.Unsetenv("CTL_DATABASE_HOST")
		os.Unsetenv("CTL_DATABASE_PORT")
		os.Unsetenv("CTL_DATABASE_SSL_ENABLED")
		os.Unsetenv("CTL_DATABASE_SSL_CERT_PATH")
	}()

	type SSLConfig struct {
		Enabled  bool   `env:"DATABASE_SSL_ENABLED"`   // Include parent context
		CertPath string `env:"DATABASE_SSL_CERT_PATH"` // Include parent context
	}

	type DatabaseConfig struct {
		Host string    `env:"DATABASE_HOST"`
		Port int       `env:"DATABASE_PORT"`
		SSL  SSLConfig // Nested struct
	}

	config := &DatabaseConfig{}

	// Create feeder with tenant prefix (include trailing underscore)
	feeder := NewTenantAffixedEnvFeeder(
		func(tenantId string) string { return tenantId + "_" },
		func(env string) string { return "" },
	)

	// Tenant loader pre-configures prefix
	feeder.SetPrefixFunc("CTL")

	// Config builder calls FeedKey with section name
	err := feeder.FeedKey("database", config)
	if err != nil {
		t.Errorf("FeedKey failed: %v", err)
	}

	// Verify top-level fields populated
	if config.Host != "db.example.com" {
		t.Errorf("Expected Host 'db.example.com', got: %s", config.Host)
	}
	if config.Port != 5432 {
		t.Errorf("Expected Port 5432, got: %d", config.Port)
	}

	// Verify nested struct fields populated (critical test!)
	if !config.SSL.Enabled {
		t.Errorf("Expected SSL.Enabled true, got: %v", config.SSL.Enabled)
	}
	if config.SSL.CertPath != "/etc/ssl/cert.pem" {
		t.Errorf("Expected SSL.CertPath '/etc/ssl/cert.pem', got: %s", config.SSL.CertPath)
	}
}

// TestTenantAffixedEnvFeeder_NestedConfig_ThreeLevels tests a config with
// two levels of nested structs
func TestTenantAffixedEnvFeeder_NestedConfig_ThreeLevels(t *testing.T) {
	// Set up env vars for deeply nested config
	// Note: env tags should include full context path for clarity
	os.Setenv("SAMPLEAFF1_API_URL", "https://api.example.com")
	os.Setenv("SAMPLEAFF1_API_TIMEOUT", "30")
	os.Setenv("SAMPLEAFF1_API_AUTH_TYPE", "oauth2")
	os.Setenv("SAMPLEAFF1_API_AUTH_TOKEN", "secret-token")
	os.Setenv("SAMPLEAFF1_API_AUTH_OAUTH_CLIENT_ID", "client-123")
	os.Setenv("SAMPLEAFF1_API_AUTH_OAUTH_CLIENT_SECRET", "secret-456")
	os.Setenv("SAMPLEAFF1_API_AUTH_OAUTH_TOKEN_URL", "https://oauth.example.com/token")
	defer func() {
		os.Unsetenv("SAMPLEAFF1_API_URL")
		os.Unsetenv("SAMPLEAFF1_API_TIMEOUT")
		os.Unsetenv("SAMPLEAFF1_API_AUTH_TYPE")
		os.Unsetenv("SAMPLEAFF1_API_AUTH_TOKEN")
		os.Unsetenv("SAMPLEAFF1_API_AUTH_OAUTH_CLIENT_ID")
		os.Unsetenv("SAMPLEAFF1_API_AUTH_OAUTH_CLIENT_SECRET")
		os.Unsetenv("SAMPLEAFF1_API_AUTH_OAUTH_TOKEN_URL")
	}()

	type OAuthConfig struct {
		ClientID     string `env:"API_AUTH_OAUTH_CLIENT_ID"`     // Full context path
		ClientSecret string `env:"API_AUTH_OAUTH_CLIENT_SECRET"` // Full context path
		TokenURL     string `env:"API_AUTH_OAUTH_TOKEN_URL"`     // Full context path
	}

	type AuthConfig struct {
		Type  string      `env:"API_AUTH_TYPE"`  // Include API context
		Token string      `env:"API_AUTH_TOKEN"` // Include API context
		OAuth OAuthConfig // Nested struct (level 3)
	}

	type APIConfig struct {
		URL     string     `env:"API_URL"`
		Timeout int        `env:"API_TIMEOUT"`
		Auth    AuthConfig // Nested struct (level 2)
	}

	config := &APIConfig{}

	// Create feeder with tenant prefix (include trailing underscore)
	feeder := NewTenantAffixedEnvFeeder(
		func(tenantId string) string { return tenantId + "_" },
		func(env string) string { return "" },
	)

	// Tenant loader pre-configures prefix
	feeder.SetPrefixFunc("SAMPLEAFF1")

	// Config builder calls FeedKey with section name
	err := feeder.FeedKey("api", config)
	if err != nil {
		t.Errorf("FeedKey failed: %v", err)
	}

	// Verify level 1 (top-level) fields
	if config.URL != "https://api.example.com" {
		t.Errorf("Expected URL 'https://api.example.com', got: %s", config.URL)
	}
	if config.Timeout != 30 {
		t.Errorf("Expected Timeout 30, got: %d", config.Timeout)
	}

	// Verify level 2 (nested) fields
	if config.Auth.Type != "oauth2" {
		t.Errorf("Expected Auth.Type 'oauth2', got: %s", config.Auth.Type)
	}
	if config.Auth.Token != "secret-token" {
		t.Errorf("Expected Auth.Token 'secret-token', got: %s", config.Auth.Token)
	}

	// Verify level 3 (deeply nested) fields - CRITICAL!
	if config.Auth.OAuth.ClientID != "client-123" {
		t.Errorf("Expected Auth.OAuth.ClientID 'client-123', got: %s", config.Auth.OAuth.ClientID)
	}
	if config.Auth.OAuth.ClientSecret != "secret-456" {
		t.Errorf("Expected Auth.OAuth.ClientSecret 'secret-456', got: %s", config.Auth.OAuth.ClientSecret)
	}
	if config.Auth.OAuth.TokenURL != "https://oauth.example.com/token" {
		t.Errorf("Expected Auth.OAuth.TokenURL 'https://oauth.example.com/token', got: %s", config.Auth.OAuth.TokenURL)
	}
}

// TestTenantAffixedEnvFeeder_NestedConfig_WithPointers tests nested structs
// using pointer fields (common pattern for optional nested configs)
func TestTenantAffixedEnvFeeder_NestedConfig_WithPointers(t *testing.T) {
	os.Setenv("CTL_CACHE_ENABLED", "true")
	os.Setenv("CTL_CACHE_TTL", "3600")
	os.Setenv("CTL_CACHE_REDIS_HOST", "redis.example.com")
	os.Setenv("CTL_CACHE_REDIS_PORT", "6379")
	os.Setenv("CTL_CACHE_REDIS_PASSWORD", "redis-secret")
	defer func() {
		os.Unsetenv("CTL_CACHE_ENABLED")
		os.Unsetenv("CTL_CACHE_TTL")
		os.Unsetenv("CTL_CACHE_REDIS_HOST")
		os.Unsetenv("CTL_CACHE_REDIS_PORT")
		os.Unsetenv("CTL_CACHE_REDIS_PASSWORD")
	}()

	type RedisConfig struct {
		Host     string `env:"CACHE_REDIS_HOST"`     // Include cache context
		Port     int    `env:"CACHE_REDIS_PORT"`     // Include cache context
		Password string `env:"CACHE_REDIS_PASSWORD"` // Include cache context
	}

	type CacheConfig struct {
		Enabled bool         `env:"CACHE_ENABLED"`
		TTL     int          `env:"CACHE_TTL"`
		Redis   *RedisConfig // Pointer to nested struct
	}

	// Initialize with pointer
	config := &CacheConfig{
		Redis: &RedisConfig{}, // Pre-initialize pointer
	}

	feeder := NewTenantAffixedEnvFeeder(
		func(tenantId string) string { return tenantId + "_" },
		func(env string) string { return "" },
	)

	feeder.SetPrefixFunc("CTL")

	err := feeder.FeedKey("cache", config)
	if err != nil {
		t.Errorf("FeedKey failed: %v", err)
	}

	// Verify top-level fields
	if !config.Enabled {
		t.Errorf("Expected Enabled true, got: %v", config.Enabled)
	}
	if config.TTL != 3600 {
		t.Errorf("Expected TTL 3600, got: %d", config.TTL)
	}

	// Verify nested pointer struct fields
	if config.Redis == nil {
		t.Fatal("Expected Redis to be non-nil")
	}
	if config.Redis.Host != "redis.example.com" {
		t.Errorf("Expected Redis.Host 'redis.example.com', got: %s", config.Redis.Host)
	}
	if config.Redis.Port != 6379 {
		t.Errorf("Expected Redis.Port 6379, got: %d", config.Redis.Port)
	}
	if config.Redis.Password != "redis-secret" {
		t.Errorf("Expected Redis.Password 'redis-secret', got: %s", config.Redis.Password)
	}
}

// TestTenantAffixedEnvFeeder_NestedConfig_MultipleSections tests that prefix
// is preserved across multiple sections with nested configs
func TestTenantAffixedEnvFeeder_NestedConfig_MultipleSections(t *testing.T) {
	// Section 1: AWS config (nested)
	os.Setenv("CTL_AWS_REGION", "us-east-1")
	os.Setenv("CTL_AWS_S3_BUCKET", "ctl-uploads")
	os.Setenv("CTL_AWS_S3_PREFIX", "prod/")

	// Section 2: Database config (nested)
	os.Setenv("CTL_DB_HOST", "db.example.com")
	os.Setenv("CTL_DB_POOL_MIN_CONNS", "5")
	os.Setenv("CTL_DB_POOL_MAX_CONNS", "20")

	defer func() {
		os.Unsetenv("CTL_AWS_REGION")
		os.Unsetenv("CTL_AWS_S3_BUCKET")
		os.Unsetenv("CTL_AWS_S3_PREFIX")
		os.Unsetenv("CTL_DB_HOST")
		os.Unsetenv("CTL_DB_POOL_MIN_CONNS")
		os.Unsetenv("CTL_DB_POOL_MAX_CONNS")
	}()

	type S3Config struct {
		Bucket string `env:"AWS_S3_BUCKET"` // Include AWS context
		Prefix string `env:"AWS_S3_PREFIX"` // Include AWS context
	}

	type AWSConfig struct {
		Region string   `env:"AWS_REGION"`
		S3     S3Config // Nested
	}

	type PoolConfig struct {
		MinConns int `env:"DB_POOL_MIN_CONNS"` // Include DB context
		MaxConns int `env:"DB_POOL_MAX_CONNS"` // Include DB context
	}

	type DBConfig struct {
		Host string     `env:"DB_HOST"`
		Pool PoolConfig // Nested
	}

	awsConfig := &AWSConfig{}
	dbConfig := &DBConfig{}

	// Create feeder once, use for both sections
	feeder := NewTenantAffixedEnvFeeder(
		func(tenantId string) string { return tenantId + "_" },
		func(env string) string { return "" },
	)

	// Tenant loader pre-configures prefix once
	feeder.SetPrefixFunc("CTL")

	// Simulate config builder processing multiple sections
	// Section 1: aws
	err := feeder.FeedKey("aws", awsConfig)
	if err != nil {
		t.Errorf("FeedKey for aws failed: %v", err)
	}

	// Verify prefix is still CTL_ after first section (includes trailing underscore from prefix function)
	if feeder.Prefix != "CTL_" {
		t.Errorf("Expected prefix 'CTL_' after first section, got: %s", feeder.Prefix)
	}

	// Section 2: database
	err = feeder.FeedKey("database", dbConfig)
	if err != nil {
		t.Errorf("FeedKey for database failed: %v", err)
	}

	// Verify prefix is STILL CTL_ after second section (not overwritten to "database_")
	if feeder.Prefix != "CTL_" {
		t.Errorf("Expected prefix 'CTL_' after second section, got: %s", feeder.Prefix)
	}

	// Verify AWS config (section 1) with nested struct
	if awsConfig.Region != "us-east-1" {
		t.Errorf("Expected AWS Region 'us-east-1', got: %s", awsConfig.Region)
	}
	if awsConfig.S3.Bucket != "ctl-uploads" {
		t.Errorf("Expected AWS S3 Bucket 'ctl-uploads', got: %s", awsConfig.S3.Bucket)
	}
	if awsConfig.S3.Prefix != "prod/" {
		t.Errorf("Expected AWS S3 Prefix 'prod/', got: %s", awsConfig.S3.Prefix)
	}

	// Verify DB config (section 2) with nested struct
	if dbConfig.Host != "db.example.com" {
		t.Errorf("Expected DB Host 'db.example.com', got: %s", dbConfig.Host)
	}
	if dbConfig.Pool.MinConns != 5 {
		t.Errorf("Expected DB Pool MinConns 5, got: %d", dbConfig.Pool.MinConns)
	}
	if dbConfig.Pool.MaxConns != 20 {
		t.Errorf("Expected DB Pool MaxConns 20, got: %d", dbConfig.Pool.MaxConns)
	}
}

// TestTenantAffixedEnvFeeder_NestedConfig_WithSuffix tests nested configs
// with both tenant prefix and environment suffix
func TestTenantAffixedEnvFeeder_NestedConfig_WithSuffix(t *testing.T) {
	os.Setenv("CTL_SERVICE_URL_PROD", "https://ctl-prod.example.com")
	os.Setenv("CTL_SERVICE_TIMEOUT_PROD", "60")
	os.Setenv("CTL_SERVICE_RETRY_MAX_ATTEMPTS_PROD", "3")
	os.Setenv("CTL_SERVICE_RETRY_BACKOFF_MS_PROD", "1000")
	defer func() {
		os.Unsetenv("CTL_SERVICE_URL_PROD")
		os.Unsetenv("CTL_SERVICE_TIMEOUT_PROD")
		os.Unsetenv("CTL_SERVICE_RETRY_MAX_ATTEMPTS_PROD")
		os.Unsetenv("CTL_SERVICE_RETRY_BACKOFF_MS_PROD")
	}()

	type RetryConfig struct {
		MaxAttempts int `env:"SERVICE_RETRY_MAX_ATTEMPTS"` // Include service context
		BackoffMs   int `env:"SERVICE_RETRY_BACKOFF_MS"`   // Include service context
	}

	type ServiceConfig struct {
		URL     string      `env:"SERVICE_URL"`
		Timeout int         `env:"SERVICE_TIMEOUT"`
		Retry   RetryConfig // Nested
	}

	config := &ServiceConfig{}

	feeder := NewTenantAffixedEnvFeeder(
		func(tenantId string) string { return tenantId + "_" },
		func(env string) string { return "_" + env },
	)

	// Configure both prefix and suffix
	feeder.SetPrefixFunc("CTL")
	feeder.SetSuffixFunc("PROD")

	// Config builder calls FeedKey with section name
	err := feeder.FeedKey("service", config)
	if err != nil {
		t.Errorf("FeedKey failed: %v", err)
	}

	// Verify both prefix and suffix preserved (with separators included)
	if feeder.Prefix != "CTL_" {
		t.Errorf("Expected prefix 'CTL_', got: %s", feeder.Prefix)
	}
	if feeder.Suffix != "_PROD" {
		t.Errorf("Expected suffix '_PROD', got: %s", feeder.Suffix)
	}

	// Verify top-level fields with prefix and suffix
	if config.URL != "https://ctl-prod.example.com" {
		t.Errorf("Expected URL 'https://ctl-prod.example.com', got: %s", config.URL)
	}
	if config.Timeout != 60 {
		t.Errorf("Expected Timeout 60, got: %d", config.Timeout)
	}

	// Verify nested struct fields with prefix and suffix
	if config.Retry.MaxAttempts != 3 {
		t.Errorf("Expected Retry.MaxAttempts 3, got: %d", config.Retry.MaxAttempts)
	}
	if config.Retry.BackoffMs != 1000 {
		t.Errorf("Expected Retry.BackoffMs 1000, got: %d", config.Retry.BackoffMs)
	}
}

// TestTenantAffixedEnvFeeder_NestedConfig_FourLevels tests extremely deep nesting
func TestTenantAffixedEnvFeeder_NestedConfig_FourLevels(t *testing.T) {
	os.Setenv("TENANT1_APP_NAME", "myapp")
	os.Setenv("TENANT1_APP_SERVER_HOST", "localhost")
	os.Setenv("TENANT1_APP_SERVER_TLS_ENABLED", "true")
	os.Setenv("TENANT1_APP_SERVER_TLS_CERTS_CA_PATH", "/etc/ssl/ca.pem")
	os.Setenv("TENANT1_APP_SERVER_TLS_CERTS_CERT_PATH", "/etc/ssl/cert.pem")
	os.Setenv("TENANT1_APP_SERVER_TLS_CERTS_KEY_PATH", "/etc/ssl/key.pem")
	defer func() {
		os.Unsetenv("TENANT1_APP_NAME")
		os.Unsetenv("TENANT1_APP_SERVER_HOST")
		os.Unsetenv("TENANT1_APP_SERVER_TLS_ENABLED")
		os.Unsetenv("TENANT1_APP_SERVER_TLS_CERTS_CA_PATH")
		os.Unsetenv("TENANT1_APP_SERVER_TLS_CERTS_CERT_PATH")
		os.Unsetenv("TENANT1_APP_SERVER_TLS_CERTS_KEY_PATH")
	}()

	type CertsConfig struct {
		CAPath   string `env:"APP_SERVER_TLS_CERTS_CA_PATH"`   // Full context path
		CertPath string `env:"APP_SERVER_TLS_CERTS_CERT_PATH"` // Full context path
		KeyPath  string `env:"APP_SERVER_TLS_CERTS_KEY_PATH"`  // Full context path
	}

	type TLSConfig struct {
		Enabled bool        `env:"APP_SERVER_TLS_ENABLED"` // Full context path
		Certs   CertsConfig // Level 4
	}

	type ServerConfig struct {
		Host string    `env:"APP_SERVER_HOST"` // Include app context
		TLS  TLSConfig // Level 3
	}

	type AppConfig struct {
		Name   string       `env:"APP_NAME"`
		Server ServerConfig // Level 2
	}

	config := &AppConfig{}

	feeder := NewTenantAffixedEnvFeeder(
		func(tenantId string) string { return tenantId + "_" },
		func(env string) string { return "" },
	)

	feeder.SetPrefixFunc("TENANT1")

	err := feeder.FeedKey("app", config)
	if err != nil {
		t.Errorf("FeedKey failed: %v", err)
	}

	// Verify level 1
	if config.Name != "myapp" {
		t.Errorf("Expected Name 'myapp', got: %s", config.Name)
	}

	// Verify level 2
	if config.Server.Host != "localhost" {
		t.Errorf("Expected Server.Host 'localhost', got: %s", config.Server.Host)
	}

	// Verify level 3
	if !config.Server.TLS.Enabled {
		t.Errorf("Expected Server.TLS.Enabled true, got: %v", config.Server.TLS.Enabled)
	}

	// Verify level 4 (deepest nesting) - CRITICAL!
	if config.Server.TLS.Certs.CAPath != "/etc/ssl/ca.pem" {
		t.Errorf("Expected Server.TLS.Certs.CAPath '/etc/ssl/ca.pem', got: %s", config.Server.TLS.Certs.CAPath)
	}
	if config.Server.TLS.Certs.CertPath != "/etc/ssl/cert.pem" {
		t.Errorf("Expected Server.TLS.Certs.CertPath '/etc/ssl/cert.pem', got: %s", config.Server.TLS.Certs.CertPath)
	}
	if config.Server.TLS.Certs.KeyPath != "/etc/ssl/key.pem" {
		t.Errorf("Expected Server.TLS.Certs.KeyPath '/etc/ssl/key.pem', got: %s", config.Server.TLS.Certs.KeyPath)
	}
}
