package feeders

import (
	"os"
	"testing"
)

// TestTenantAffixedEnvFeeder_FeedWithoutPrefixSuffix tests the NEW behavior where
// Feed() returns nil instead of error when prefix/suffix are not configured
func TestTenantAffixedEnvFeeder_FeedWithoutPrefixSuffix(t *testing.T) {
	type TestConfig struct {
		Name string `env:"NAME"`
		Port int    `env:"PORT"`
	}

	config := &TestConfig{}

	// Create feeder without pre-configuring prefix/suffix
	feeder := NewTenantAffixedEnvFeeder(
		func(tenantId string) string { return "PREFIX_" + tenantId + "_" },
		func(env string) string { return "_" + env },
	)

	// Feed() should now return nil (NEW behavior - not an error)
	err := feeder.Feed(config)
	if err != nil {
		t.Errorf("Expected Feed() to return nil when prefix/suffix not set, got error: %v", err)
	}

	// Verify prefix and suffix are still empty (Feed did nothing)
	if feeder.Prefix != "" {
		t.Errorf("Expected Prefix to remain empty, got: %s", feeder.Prefix)
	}
	if feeder.Suffix != "" {
		t.Errorf("Expected Suffix to remain empty, got: %s", feeder.Suffix)
	}
}

// TestTenantAffixedEnvFeeder_FeedKeyPreservesPreConfiguredPrefix tests that
// FeedKey() preserves prefix when it's already set (NEW behavior)
func TestTenantAffixedEnvFeeder_FeedKeyPreservesPreConfiguredPrefix(t *testing.T) {
	// Set up environment variable with pre-configured prefix
	// Format: PREFIX_CTL_ + NAME = PREFIX_CTL_NAME (framework no longer adds underscores)
	os.Setenv("PREFIX_CTL_NAME", "crisis-text-line")
	defer os.Unsetenv("PREFIX_CTL_NAME")

	type TestConfig struct {
		Name string `env:"NAME"`
	}

	config := &TestConfig{}

	// Create feeder
	feeder := NewTenantAffixedEnvFeeder(
		func(tenantId string) string { return "PREFIX_" + tenantId + "_" },
		func(env string) string { return "_" + env },
	)

	// Pre-configure prefix to "PREFIX_CTL_" (simulating tenant loader behavior)
	feeder.SetPrefixFunc("CTL")
	if feeder.Prefix != "PREFIX_CTL_" {
		t.Fatalf("Expected prefix 'PREFIX_CTL_', got: %s", feeder.Prefix)
	}

	// Now call FeedKey with a different key (section name)
	// FeedKey should PRESERVE the prefix, not overwrite it
	err := feeder.FeedKey("notifications", config)
	if err != nil {
		t.Errorf("FeedKey failed: %v", err)
	}

	// Verify prefix was preserved (NOT overwritten with "PREFIX_notifications_")
	if feeder.Prefix != "PREFIX_CTL_" {
		t.Errorf("Expected prefix to be preserved as 'PREFIX_CTL_', got: %s", feeder.Prefix)
	}

	// Verify config was populated using the preserved prefix
	if config.Name != "crisis-text-line" {
		t.Errorf("Expected Name to be 'crisis-text-line', got: %s", config.Name)
	}
}

// TestTenantAffixedEnvFeeder_FeedKeyPreservesPreConfiguredSuffix tests that
// FeedKey() preserves suffix when it's already set (NEW behavior)
// When only suffix is set (prefix empty), FeedKey preserves BOTH - doesn't set prefix from key
func TestTenantAffixedEnvFeeder_FeedKeyPreservesPreConfiguredSuffix(t *testing.T) {
	// Set up environment variable with only suffix (no prefix)
	// Format: NAME + _PROD = NAME_PROD (framework no longer adds underscores)
	os.Setenv("NAME_PROD", "production-app")
	defer os.Unsetenv("NAME_PROD")

	type TestConfig struct {
		Name string `env:"NAME"`
	}

	config := &TestConfig{}

	// Create feeder
	feeder := NewTenantAffixedEnvFeeder(
		func(tenantId string) string { return tenantId + "_" },
		func(env string) string { return "_" + env },
	)

	// Pre-configure suffix to "_PROD" (simulating tenant loader behavior)
	feeder.SetSuffixFunc("PROD")
	if feeder.Suffix != "_PROD" {
		t.Fatalf("Expected suffix '_PROD', got: %s", feeder.Suffix)
	}

	// Now call FeedKey with a different key (section name)
	// FeedKey should PRESERVE both - prefix stays empty, suffix preserved
	err := feeder.FeedKey("database", config)
	if err != nil {
		t.Errorf("FeedKey failed: %v", err)
	}

	// Verify prefix stays empty (NOT set from key)
	if feeder.Prefix != "" {
		t.Errorf("Expected prefix to remain empty, got: %s", feeder.Prefix)
	}

	// Verify suffix was preserved (NOT overwritten with "_database")
	if feeder.Suffix != "_PROD" {
		t.Errorf("Expected suffix to be preserved as '_PROD', got: %s", feeder.Suffix)
	}

	// Verify config was populated using the preserved suffix
	if config.Name != "production-app" {
		t.Errorf("Expected Name 'production-app', got: %s", config.Name)
	}
}

// TestTenantAffixedEnvFeeder_FeedKeyPreservesBothPrefixAndSuffix tests that
// FeedKey() preserves both prefix and suffix when both are set (NEW behavior)
func TestTenantAffixedEnvFeeder_FeedKeyPreservesBothPrefixAndSuffix(t *testing.T) {
	// Set up environment variable with both prefix and suffix
	os.Setenv("CTL_NAME_PROD", "ctl-production")
	defer os.Unsetenv("CTL_NAME_PROD")

	type TestConfig struct {
		Name string `env:"NAME"`
	}

	config := &TestConfig{}

	// Create feeder
	feeder := NewTenantAffixedEnvFeeder(
		func(tenantId string) string { return tenantId + "_" },
		func(env string) string { return "_" + env },
	)

	// Pre-configure both prefix and suffix (simulating tenant loader behavior)
	feeder.SetPrefixFunc("CTL")
	feeder.SetSuffixFunc("PROD")

	if feeder.Prefix != "CTL_" {
		t.Fatalf("Expected prefix 'CTL_', got: %s", feeder.Prefix)
	}
	if feeder.Suffix != "_PROD" {
		t.Fatalf("Expected suffix '_PROD', got: %s", feeder.Suffix)
	}

	// Now call FeedKey with a section name
	// FeedKey should PRESERVE both prefix and suffix
	err := feeder.FeedKey("aws", config)
	if err != nil {
		t.Errorf("FeedKey failed: %v", err)
	}

	// Verify both were preserved
	if feeder.Prefix != "CTL_" {
		t.Errorf("Expected prefix to be preserved as 'CTL_', got: %s", feeder.Prefix)
	}
	if feeder.Suffix != "_PROD" {
		t.Errorf("Expected suffix to be preserved as '_PROD', got: %s", feeder.Suffix)
	}

	// Verify config was populated correctly
	if config.Name != "ctl-production" {
		t.Errorf("Expected Name to be 'ctl-production', got: %s", config.Name)
	}
}

// TestTenantAffixedEnvFeeder_FeedKeySetsFromKeyWhenNotPreConfigured tests that
// FeedKey() still sets prefix/suffix from key when they're not pre-configured (backward compatibility)
func TestTenantAffixedEnvFeeder_FeedKeySetsFromKeyWhenNotPreConfigured(t *testing.T) {
	// Set up environment variable with tenant-specific prefix and suffix
	// Prefix func returns: tenantId + "_" = "tenant123_"
	// Suffix func returns: "_" + env = "_tenant123"
	// Format: TENANT123_ + NAME + _TENANT123 = TENANT123_NAME_TENANT123 (ToUpper applied, framework no longer adds underscores)
	os.Setenv("TENANT123_NAME_TENANT123", "tenant-app")
	defer os.Unsetenv("TENANT123_NAME_TENANT123")

	type TestConfig struct {
		Name string `env:"NAME"`
	}

	config := &TestConfig{}

	// Create feeder WITHOUT pre-configuring prefix/suffix
	feeder := NewTenantAffixedEnvFeeder(
		func(tenantId string) string { return tenantId + "_" },
		func(env string) string { return "_" + env },
	)

	// Call FeedKey with tenant ID - should set prefix AND suffix from the key parameter
	err := feeder.FeedKey("tenant123", config)
	if err != nil {
		t.Errorf("FeedKey failed: %v", err)
	}

	// Verify prefix was set from the key parameter
	if feeder.Prefix != "tenant123_" {
		t.Errorf("Expected prefix 'tenant123_', got: %s", feeder.Prefix)
	}

	// Verify suffix was also set from the key parameter
	if feeder.Suffix != "_tenant123" {
		t.Errorf("Expected suffix '_tenant123', got: %s", feeder.Suffix)
	}

	// Verify config was populated
	if config.Name != "tenant-app" {
		t.Errorf("Expected Name to be 'tenant-app', got: %s", config.Name)
	}
}

// TestTenantAffixedEnvFeeder_SequentialFeedKeyCallsPreserveFirst tests that
// when FeedKey is called multiple times, the first prefix/suffix are preserved
func TestTenantAffixedEnvFeeder_SequentialFeedKeyCallsPreserveFirst(t *testing.T) {
	// Set up environment variables for first tenant
	// Prefix: tenant1_ , Suffix: _tenant1
	// Format: TENANT1_ + NAME + _TENANT1 = TENANT1_NAME_TENANT1 (framework no longer adds underscores)
	os.Setenv("TENANT1_NAME_TENANT1", "tenant-one")
	os.Setenv("TENANT1_PORT_TENANT1", "8080")
	defer func() {
		os.Unsetenv("TENANT1_NAME_TENANT1")
		os.Unsetenv("TENANT1_PORT_TENANT1")
	}()

	type TestConfig struct {
		Name string `env:"NAME"`
		Port int    `env:"PORT"`
	}

	config1 := &TestConfig{}
	config2 := &TestConfig{}

	// Create feeder
	feeder := NewTenantAffixedEnvFeeder(
		func(tenantId string) string { return tenantId + "_" },
		func(env string) string { return "_" + env },
	)

	// First FeedKey call - sets prefix to "tenant1_" and suffix to "_tenant1"
	err := feeder.FeedKey("tenant1", config1)
	if err != nil {
		t.Errorf("First FeedKey failed: %v", err)
	}

	if feeder.Prefix != "tenant1_" {
		t.Fatalf("Expected first prefix 'tenant1_', got: %s", feeder.Prefix)
	}

	if feeder.Suffix != "_tenant1" {
		t.Fatalf("Expected first suffix '_tenant1', got: %s", feeder.Suffix)
	}

	if config1.Name != "tenant-one" {
		t.Errorf("Expected first config Name 'tenant-one', got: %s", config1.Name)
	}

	// Second FeedKey call with different key - should PRESERVE "tenant1_" prefix and suffix
	err = feeder.FeedKey("tenant2", config2)
	if err != nil {
		t.Errorf("Second FeedKey failed: %v", err)
	}

	// Verify prefix was preserved (not overwritten to "tenant2_")
	if feeder.Prefix != "tenant1_" {
		t.Errorf("Expected prefix to remain 'tenant1_' after second call, got: %s", feeder.Prefix)
	}

	// Verify suffix was preserved (not overwritten to "_tenant2")
	if feeder.Suffix != "_tenant1" {
		t.Errorf("Expected suffix to remain '_tenant1' after second call, got: %s", feeder.Suffix)
	}

	// Second config should also use first tenant's env vars
	if config2.Name != "tenant-one" {
		t.Errorf("Expected second config to use preserved prefix, got Name: %s", config2.Name)
	}
}

// TestTenantAffixedEnvFeeder_TenantLoaderWorkflow tests the complete workflow
// where tenant loader pre-configures prefix, then config builder calls FeedKey with section names
func TestTenantAffixedEnvFeeder_TenantLoaderWorkflow(t *testing.T) {
	// Set up environment variables for CTL tenant with AWS section
	// Format: CTL_ + AWS_REGION = CTL_AWS_REGION (framework no longer adds underscores)
	os.Setenv("CTL_AWS_REGION", "us-east-1")
	os.Setenv("CTL_AWS_BUCKET", "ctl-uploads")
	os.Setenv("CTL_DATABASE_HOST", "ctl-db.example.com")
	os.Setenv("CTL_DATABASE_PORT", "5432")
	defer func() {
		os.Unsetenv("CTL_AWS_REGION")
		os.Unsetenv("CTL_AWS_BUCKET")
		os.Unsetenv("CTL_DATABASE_HOST")
		os.Unsetenv("CTL_DATABASE_PORT")
	}()

	type AWSConfig struct {
		Region string `env:"AWS_REGION"`
		Bucket string `env:"AWS_BUCKET"`
	}

	type DatabaseConfig struct {
		Host string `env:"DATABASE_HOST"`
		Port int    `env:"DATABASE_PORT"`
	}

	awsConfig := &AWSConfig{}
	dbConfig := &DatabaseConfig{}

	// Create feeder (as tenant loader would)
	feeder := NewTenantAffixedEnvFeeder(
		func(tenantId string) string { return tenantId + "_" },
		func(env string) string { return "_" + env },
	)

	// Step 1: Tenant loader pre-configures prefix for tenant "CTL"
	feeder.SetPrefixFunc("CTL")
	if feeder.Prefix != "CTL_" {
		t.Fatalf("Expected prefix 'CTL_', got: %s", feeder.Prefix)
	}

	// Step 2: Config builder calls FeedKey with section name "aws"
	// This should PRESERVE the "CTL_" prefix, not overwrite it with "aws_"
	err := feeder.FeedKey("aws", awsConfig)
	if err != nil {
		t.Errorf("FeedKey for aws section failed: %v", err)
	}

	// Verify prefix was preserved
	if feeder.Prefix != "CTL_" {
		t.Errorf("Expected prefix preserved as 'CTL_', got: %s", feeder.Prefix)
	}

	// Verify AWS config was loaded correctly using CTL_ prefix
	if awsConfig.Region != "us-east-1" {
		t.Errorf("Expected AWS Region 'us-east-1', got: %s", awsConfig.Region)
	}
	if awsConfig.Bucket != "ctl-uploads" {
		t.Errorf("Expected AWS Bucket 'ctl-uploads', got: %s", awsConfig.Bucket)
	}

	// Step 3: Config builder calls FeedKey again with section name "database"
	// Prefix should STILL be preserved
	err = feeder.FeedKey("database", dbConfig)
	if err != nil {
		t.Errorf("FeedKey for database section failed: %v", err)
	}

	// Verify prefix still preserved
	if feeder.Prefix != "CTL_" {
		t.Errorf("Expected prefix still preserved as 'CTL_', got: %s", feeder.Prefix)
	}

	// Verify database config was loaded correctly using CTL_ prefix
	if dbConfig.Host != "ctl-db.example.com" {
		t.Errorf("Expected Database Host 'ctl-db.example.com', got: %s", dbConfig.Host)
	}
	if dbConfig.Port != 5432 {
		t.Errorf("Expected Database Port 5432, got: %d", dbConfig.Port)
	}
}

// TestTenantAffixedEnvFeeder_OnlyPrefixPreConfigured tests preservation when only prefix is set
// The behavior: if EITHER prefix or suffix is set, FeedKey preserves BOTH (no setting from key)
func TestTenantAffixedEnvFeeder_OnlyPrefixPreConfigured(t *testing.T) {
	// Env var format: CTL_ + NAME = CTL_NAME (no suffix, framework no longer adds underscores)
	os.Setenv("CTL_NAME", "ctl-app")
	defer os.Unsetenv("CTL_NAME")

	type TestConfig struct {
		Name string `env:"NAME"`
	}

	config := &TestConfig{}

	feeder := NewTenantAffixedEnvFeeder(
		func(tenantId string) string { return tenantId + "_" },
		func(env string) string { return "_" + env },
	)

	// Only set prefix, not suffix
	feeder.SetPrefixFunc("CTL")

	// Verify only prefix is set
	if feeder.Prefix != "CTL_" {
		t.Fatalf("Expected prefix 'CTL_', got: %s", feeder.Prefix)
	}
	if feeder.Suffix != "" {
		t.Fatalf("Expected suffix to be empty, got: %s", feeder.Suffix)
	}

	// FeedKey should preserve BOTH - prefix stays, suffix stays empty (NOT set from key)
	err := feeder.FeedKey("DEV", config)
	if err != nil {
		t.Errorf("FeedKey failed: %v", err)
	}

	// Verify prefix preserved and suffix STILL empty (not set to "_DEV")
	if feeder.Prefix != "CTL_" {
		t.Errorf("Expected prefix preserved as 'CTL_', got: %s", feeder.Prefix)
	}
	if feeder.Suffix != "" {
		t.Errorf("Expected suffix to remain empty, got: %s", feeder.Suffix)
	}

	if config.Name != "ctl-app" {
		t.Errorf("Expected Name 'ctl-app', got: %s", config.Name)
	}
}

// TestTenantAffixedEnvFeeder_OnlySuffixPreConfigured tests preservation when only suffix is set
// The behavior: if EITHER prefix or suffix is set, FeedKey preserves BOTH (no setting from key)
func TestTenantAffixedEnvFeeder_OnlySuffixPreConfigured(t *testing.T) {
	// Env var format: NAME + _PROD = NAME_PROD (framework no longer adds underscores)
	os.Setenv("NAME_PROD", "prod-app")
	defer os.Unsetenv("NAME_PROD")

	type TestConfig struct {
		Name string `env:"NAME"`
	}

	config := &TestConfig{}

	feeder := NewTenantAffixedEnvFeeder(
		func(tenantId string) string { return tenantId + "_" },
		func(env string) string { return "_" + env },
	)

	// Only set suffix, not prefix
	feeder.SetSuffixFunc("PROD")

	// Verify only suffix is set
	if feeder.Prefix != "" {
		t.Fatalf("Expected prefix to be empty, got: %s", feeder.Prefix)
	}
	if feeder.Suffix != "_PROD" {
		t.Fatalf("Expected suffix '_PROD', got: %s", feeder.Suffix)
	}

	// FeedKey should preserve BOTH - prefix stays empty, suffix stays (NOT set from key)
	err := feeder.FeedKey("tenant1", config)
	if err != nil {
		t.Errorf("FeedKey failed: %v", err)
	}

	// Verify prefix STILL empty (not set to "tenant1_") and suffix preserved
	if feeder.Prefix != "" {
		t.Errorf("Expected prefix to remain empty, got: %s", feeder.Prefix)
	}
	if feeder.Suffix != "_PROD" {
		t.Errorf("Expected suffix preserved as '_PROD', got: %s", feeder.Suffix)
	}

	if config.Name != "prod-app" {
		t.Errorf("Expected Name 'prod-app', got: %s", config.Name)
	}
}
