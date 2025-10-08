package feeders

import (
	"os"
	"testing"
)

// Real-world usage tests where prefix/suffix don't include extra underscores
// The code itself adds _ between prefix/envtag and envtag/suffix

// TestTenantAffixedEnvFeeder_RealWorld_TenantPrefix tests the intended usage
// where prefix is just the tenant name (e.g., "CTL") without trailing underscore
func TestTenantAffixedEnvFeeder_RealWorld_TenantPrefix(t *testing.T) {
	// Real-world env var: CTL_AWS_REGION (not CTL__AWS_REGION)
	// Prefix: "CTL" (code adds _)
	// EnvTag: "AWS_REGION"
	// Result: CTL + _ + AWS_REGION = CTL_AWS_REGION
	os.Setenv("CTL_AWS_REGION", "us-east-1")
	os.Setenv("CTL_AWS_BUCKET", "ctl-uploads")
	defer func() {
		os.Unsetenv("CTL_AWS_REGION")
		os.Unsetenv("CTL_AWS_BUCKET")
	}()

	type AWSConfig struct {
		Region string `env:"AWS_REGION"`
		Bucket string `env:"AWS_BUCKET"`
	}

	config := &AWSConfig{}

	// Create feeder with prefix/suffix functions that include underscores
	feeder := NewTenantAffixedEnvFeeder(
		func(tenantId string) string { return tenantId + "_" }, // Include trailing underscore
		func(env string) string { return "" },                  // No suffix in this example
	)

	// Tenant loader pre-configures prefix
	feeder.SetPrefixFunc("CTL")

	// Verify prefix is set with underscore
	if feeder.Prefix != "CTL_" {
		t.Fatalf("Expected prefix 'CTL_', got: %s", feeder.Prefix)
	}

	// Config builder calls FeedKey with section name - should PRESERVE "CTL" prefix
	err := feeder.FeedKey("aws", config)
	if err != nil {
		t.Errorf("FeedKey failed: %v", err)
	}

	// Verify prefix was preserved
	if feeder.Prefix != "CTL_" {
		t.Errorf("Expected prefix preserved as 'CTL_', got: %s", feeder.Prefix)
	}

	// Verify config was loaded
	if config.Region != "us-east-1" {
		t.Errorf("Expected Region 'us-east-1', got: %s", config.Region)
	}
	if config.Bucket != "ctl-uploads" {
		t.Errorf("Expected Bucket 'ctl-uploads', got: %s", config.Bucket)
	}
}

// TestTenantAffixedEnvFeeder_RealWorld_TenantPrefixAndEnvSuffix tests usage
// with both tenant prefix and environment suffix
func TestTenantAffixedEnvFeeder_RealWorld_TenantPrefixAndEnvSuffix(t *testing.T) {
	// Real-world env var: CTL_DATABASE_HOST_PROD
	// Prefix: "CTL"
	// EnvTag: "DATABASE_HOST"
	// Suffix: "PROD"
	// Result: CTL + _ + DATABASE_HOST + _ + PROD = CTL_DATABASE_HOST_PROD
	os.Setenv("CTL_DATABASE_HOST_PROD", "ctl-prod-db.example.com")
	os.Setenv("CTL_DATABASE_PORT_PROD", "5432")
	defer func() {
		os.Unsetenv("CTL_DATABASE_HOST_PROD")
		os.Unsetenv("CTL_DATABASE_PORT_PROD")
	}()

	type DatabaseConfig struct {
		Host string `env:"DATABASE_HOST"`
		Port int    `env:"DATABASE_PORT"`
	}

	config := &DatabaseConfig{}

	// Create feeder with functions that include underscores
	feeder := NewTenantAffixedEnvFeeder(
		func(tenantId string) string { return tenantId + "_" }, // Include trailing underscore
		func(env string) string { return "_" + env },           // Include leading underscore
	)

	// Tenant loader pre-configures BOTH prefix and suffix
	feeder.SetPrefixFunc("CTL")
	feeder.SetSuffixFunc("PROD")

	// Verify both are set with underscores
	if feeder.Prefix != "CTL_" {
		t.Fatalf("Expected prefix 'CTL_', got: %s", feeder.Prefix)
	}
	if feeder.Suffix != "_PROD" {
		t.Fatalf("Expected suffix '_PROD', got: %s", feeder.Suffix)
	}

	// Config builder calls FeedKey - should preserve both
	err := feeder.FeedKey("database", config)
	if err != nil {
		t.Errorf("FeedKey failed: %v", err)
	}

	// Verify both preserved
	if feeder.Prefix != "CTL_" {
		t.Errorf("Expected prefix preserved as 'CTL_', got: %s", feeder.Prefix)
	}
	if feeder.Suffix != "_PROD" {
		t.Errorf("Expected suffix preserved as '_PROD', got: %s", feeder.Suffix)
	}

	// Verify config was loaded
	if config.Host != "ctl-prod-db.example.com" {
		t.Errorf("Expected Host 'ctl-prod-db.example.com', got: %s", config.Host)
	}
	if config.Port != 5432 {
		t.Errorf("Expected Port 5432, got: %d", config.Port)
	}
}

// TestTenantAffixedEnvFeeder_RealWorld_MultiTenantWorkflow simulates the
// complete multi-tenant workflow with realistic env var names
func TestTenantAffixedEnvFeeder_RealWorld_MultiTenantWorkflow(t *testing.T) {
	// Simulate multiple tenants with realistic env vars
	// CTL tenant:
	os.Setenv("CTL_API_KEY_PROD", "ctl-prod-key-123")
	os.Setenv("CTL_API_URL_PROD", "https://ctl-api.example.com")
	// SAMPLEAFF1 tenant:
	os.Setenv("SAMPLEAFF1_API_KEY_PROD", "aff1-prod-key-456")
	os.Setenv("SAMPLEAFF1_API_URL_PROD", "https://aff1-api.example.com")

	defer func() {
		os.Unsetenv("CTL_API_KEY_PROD")
		os.Unsetenv("CTL_API_URL_PROD")
		os.Unsetenv("SAMPLEAFF1_API_KEY_PROD")
		os.Unsetenv("SAMPLEAFF1_API_URL_PROD")
	}()

	type APIConfig struct {
		Key string `env:"API_KEY"`
		URL string `env:"API_URL"`
	}

	// Test CTL tenant
	ctlConfig := &APIConfig{}
	ctlFeeder := NewTenantAffixedEnvFeeder(
		func(tenantId string) string { return tenantId + "_" },
		func(env string) string { return "_" + env },
	)

	// Tenant loader configures for CTL tenant in PROD environment
	ctlFeeder.SetPrefixFunc("CTL")
	ctlFeeder.SetSuffixFunc("PROD")

	// Load config (FeedKey called with section name)
	err := ctlFeeder.FeedKey("api", ctlConfig)
	if err != nil {
		t.Errorf("CTL FeedKey failed: %v", err)
	}

	// Verify CTL config
	if ctlConfig.Key != "ctl-prod-key-123" {
		t.Errorf("Expected CTL Key 'ctl-prod-key-123', got: %s", ctlConfig.Key)
	}
	if ctlConfig.URL != "https://ctl-api.example.com" {
		t.Errorf("Expected CTL URL 'https://ctl-api.example.com', got: %s", ctlConfig.URL)
	}

	// Test SAMPLEAFF1 tenant
	aff1Config := &APIConfig{}
	aff1Feeder := NewTenantAffixedEnvFeeder(
		func(tenantId string) string { return tenantId + "_" },
		func(env string) string { return "_" + env },
	)

	// Tenant loader configures for SAMPLEAFF1 tenant in PROD environment
	aff1Feeder.SetPrefixFunc("SAMPLEAFF1")
	aff1Feeder.SetSuffixFunc("PROD")

	// Load config (FeedKey called with section name)
	err = aff1Feeder.FeedKey("api", aff1Config)
	if err != nil {
		t.Errorf("SAMPLEAFF1 FeedKey failed: %v", err)
	}

	// Verify SAMPLEAFF1 config
	if aff1Config.Key != "aff1-prod-key-456" {
		t.Errorf("Expected SAMPLEAFF1 Key 'aff1-prod-key-456', got: %s", aff1Config.Key)
	}
	if aff1Config.URL != "https://aff1-api.example.com" {
		t.Errorf("Expected SAMPLEAFF1 URL 'https://aff1-api.example.com', got: %s", aff1Config.URL)
	}
}

// TestTenantAffixedEnvFeeder_RealWorld_FeedKeyPreservation tests the critical
// NEW behavior: FeedKey preserves pre-configured prefix/suffix
func TestTenantAffixedEnvFeeder_RealWorld_FeedKeyPreservation(t *testing.T) {
	// Real-world scenario from PR description:
	// Tenant loader pre-configures prefix (e.g., "CTL")
	// Config builder calls FeedKey() with section name (e.g., "aws")
	// OLD behavior would overwrite prefix to "aws"
	// NEW behavior preserves prefix as "CTL"

	// Env vars for CTL tenant
	os.Setenv("CTL_IMAGE_UPLOAD_BUCKET", "ctl-images")
	os.Setenv("CTL_VIDEO_UPLOAD_BUCKET", "ctl-videos")
	defer func() {
		os.Unsetenv("CTL_IMAGE_UPLOAD_BUCKET")
		os.Unsetenv("CTL_VIDEO_UPLOAD_BUCKET")
	}()

	type UploadConfig struct {
		ImageBucket string `env:"IMAGE_UPLOAD_BUCKET"`
		VideoBucket string `env:"VIDEO_UPLOAD_BUCKET"`
	}

	uploadConfig := &UploadConfig{}

	// Create feeder (realistic usage)
	feeder := NewTenantAffixedEnvFeeder(
		func(tenantId string) string { return tenantId + "_" },
		func(env string) string { return "" }, // No suffix in this scenario
	)

	// Step 1: Tenant loader pre-configures prefix for "CTL" tenant
	feeder.SetPrefixFunc("CTL")

	// Verify prefix is set with underscore
	if feeder.Prefix != "CTL_" {
		t.Fatalf("Expected prefix 'CTL_' after SetPrefixFunc, got: %s", feeder.Prefix)
	}

	// Step 2: Config builder calls FeedKey with section name "uploads"
	// This is the CRITICAL test: prefix should be PRESERVED, not overwritten
	err := feeder.FeedKey("uploads", uploadConfig)
	if err != nil {
		t.Errorf("FeedKey failed: %v", err)
	}

	// Step 3: Verify prefix was PRESERVED (not overwritten to "uploads_")
	if feeder.Prefix != "CTL_" {
		t.Errorf("CRITICAL: Expected prefix preserved as 'CTL_', but got: %s (this is the bug the PR fixes!)", feeder.Prefix)
	}

	// Step 4: Verify config was loaded using CTL prefix
	if uploadConfig.ImageBucket != "ctl-images" {
		t.Errorf("Expected ImageBucket 'ctl-images', got: %s", uploadConfig.ImageBucket)
	}
	if uploadConfig.VideoBucket != "ctl-videos" {
		t.Errorf("Expected VideoBucket 'ctl-videos', got: %s", uploadConfig.VideoBucket)
	}
}

// TestTenantAffixedEnvFeeder_RealWorld_BackwardCompatibility tests that
// the NEW behavior is still backward compatible with direct FeedKey usage
func TestTenantAffixedEnvFeeder_RealWorld_BackwardCompatibility(t *testing.T) {
	// OLD usage pattern: call FeedKey directly without pre-configuring
	// Should still work by setting prefix from the key parameter

	os.Setenv("MYAPP_SERVICE_URL", "https://myapp.example.com")
	os.Setenv("MYAPP_SERVICE_PORT", "8080")
	defer func() {
		os.Unsetenv("MYAPP_SERVICE_URL")
		os.Unsetenv("MYAPP_SERVICE_PORT")
	}()

	type ServiceConfig struct {
		URL  string `env:"SERVICE_URL"`
		Port int    `env:"SERVICE_PORT"`
	}

	serviceConfig := &ServiceConfig{}

	// Create feeder WITHOUT pre-configuring
	feeder := NewTenantAffixedEnvFeeder(
		func(tenantId string) string { return tenantId + "_" },
		func(env string) string { return "" },
	)

	// Call FeedKey directly (backward compatible usage)
	// Since prefix and suffix are both empty, they should be set from key
	err := feeder.FeedKey("MYAPP", serviceConfig)
	if err != nil {
		t.Errorf("FeedKey failed: %v", err)
	}

	// Verify prefix was set from key with underscore
	if feeder.Prefix != "MYAPP_" {
		t.Errorf("Expected prefix set to 'MYAPP_', got: %s", feeder.Prefix)
	}

	// Verify config was loaded
	if serviceConfig.URL != "https://myapp.example.com" {
		t.Errorf("Expected URL 'https://myapp.example.com', got: %s", serviceConfig.URL)
	}
	if serviceConfig.Port != 8080 {
		t.Errorf("Expected Port 8080, got: %d", serviceConfig.Port)
	}
}
