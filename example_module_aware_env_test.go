package modular

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRealWorldModuleAwareEnvUsage demonstrates the module-aware environment variable functionality
// working with realistic configuration scenarios that mirror actual module usage patterns.
func TestRealWorldModuleAwareEnvUsage(t *testing.T) {
	
	t.Run("reverseproxy_realistic_config", func(t *testing.T) {
		// This test simulates a real reverse proxy configuration that might have conflicts
		// with other modules using similar environment variable names
		
		type ReverseProxyConfig struct {
			DefaultBackend    string `env:"DEFAULT_BACKEND" default:"http://localhost:8080"`
			RequestTimeout    int    `env:"REQUEST_TIMEOUT" default:"30"`
			CacheEnabled      bool   `env:"CACHE_ENABLED" default:"false"`
			MetricsEnabled    bool   `env:"METRICS_ENABLED" default:"false"`
			TenantIDHeader    string `env:"TENANT_ID_HEADER" default:"X-Tenant-ID"`
		}
		
		// Clear all environment variables
		envVars := []string{
			"DEFAULT_BACKEND", "REVERSEPROXY_DEFAULT_BACKEND", "DEFAULT_BACKEND_REVERSEPROXY",
			"REQUEST_TIMEOUT", "REVERSEPROXY_REQUEST_TIMEOUT", "REQUEST_TIMEOUT_REVERSEPROXY",
			"CACHE_ENABLED", "REVERSEPROXY_CACHE_ENABLED", "CACHE_ENABLED_REVERSEPROXY",
			"METRICS_ENABLED", "REVERSEPROXY_METRICS_ENABLED", "METRICS_ENABLED_REVERSEPROXY",
			"TENANT_ID_HEADER", "REVERSEPROXY_TENANT_ID_HEADER", "TENANT_ID_HEADER_REVERSEPROXY",
		}
		for _, env := range envVars {
			os.Unsetenv(env)
		}
		
		// Set up environment variables that might conflict across modules
		testEnvVars := map[string]string{
			// Global settings that multiple modules might want to use
			"DEFAULT_BACKEND":   "http://global.example.com",
			"REQUEST_TIMEOUT":   "10",
			"CACHE_ENABLED":     "true",
			"METRICS_ENABLED":   "true",
			
			// Reverse proxy specific settings (should override globals)
			"REVERSEPROXY_DEFAULT_BACKEND": "http://reverseproxy.example.com",
			"REVERSEPROXY_REQUEST_TIMEOUT": "60",
			"CACHE_ENABLED_REVERSEPROXY":   "false", // Uses suffix pattern
		}
		
		for key, value := range testEnvVars {
			err := os.Setenv(key, value)
			require.NoError(t, err)
		}
		
		defer func() {
			for _, env := range envVars {
				os.Unsetenv(env)
			}
		}()
		
		// Create application and register module
		app := createTestApplication(t)
		mockModule := &mockModuleAwareConfigModule{
			name:   "reverseproxy",
			config: &ReverseProxyConfig{},
		}
		app.RegisterModule(mockModule)
		
		// Initialize the application to trigger config loading
		err := app.Init()
		require.NoError(t, err)
		
		// Verify the configuration was populated with the correct priorities
		config := mockModule.config.(*ReverseProxyConfig)
		
		// Should use module-specific values when available
		assert.Equal(t, "http://reverseproxy.example.com", config.DefaultBackend) // From REVERSEPROXY_DEFAULT_BACKEND
		assert.Equal(t, 60, config.RequestTimeout)                                // From REVERSEPROXY_REQUEST_TIMEOUT
		assert.False(t, config.CacheEnabled)                                      // From CACHE_ENABLED_REVERSEPROXY (suffix)
		
		// Should fall back to global values when module-specific not available
		assert.True(t, config.MetricsEnabled)                       // From METRICS_ENABLED (global)
		assert.Equal(t, "X-Tenant-ID", config.TenantIDHeader)       // From default (no env var set)
	})
	
	t.Run("multiple_modules_same_env_vars", func(t *testing.T) {
		// Test scenario where multiple modules use the same environment variable names
		// but need different values
		
		type DatabaseConfig struct {
			Host    string `env:"HOST" default:"localhost"`
			Port    int    `env:"PORT" default:"5432"`
			Timeout int    `env:"TIMEOUT" default:"30"`
		}
		
		type HTTPServerConfig struct {
			Host    string `env:"HOST" default:"0.0.0.0"`
			Port    int    `env:"PORT" default:"8080"`
			Timeout int    `env:"TIMEOUT" default:"60"`
		}
		
		// Clear environment variables
		envVars := []string{
			"HOST", "DATABASE_HOST", "HOST_DATABASE",
			"PORT", "DATABASE_PORT", "PORT_DATABASE",
			"TIMEOUT", "DATABASE_TIMEOUT", "TIMEOUT_DATABASE",
			"HTTPSERVER_HOST", "HOST_HTTPSERVER",
			"HTTPSERVER_PORT", "PORT_HTTPSERVER",
			"HTTPSERVER_TIMEOUT", "TIMEOUT_HTTPSERVER",
		}
		for _, env := range envVars {
			os.Unsetenv(env)
		}
		
		// Set up different values for each module
		testEnvVars := map[string]string{
			// Database-specific
			"DATABASE_HOST":    "db.example.com",
			"DATABASE_PORT":    "5432",
			"TIMEOUT_DATABASE": "120", // Using suffix pattern
			
			// HTTP server-specific
			"HTTPSERVER_HOST":    "api.example.com",
			"PORT_HTTPSERVER":    "9090", // Using suffix pattern
			"HTTPSERVER_TIMEOUT": "30",
			
			// Global fallbacks
			"HOST": "fallback.example.com",
			"PORT": "8000",
		}
		
		for key, value := range testEnvVars {
			err := os.Setenv(key, value)
			require.NoError(t, err)
		}
		
		defer func() {
			for _, env := range envVars {
				os.Unsetenv(env)
			}
		}()
		
		// Create application and register both modules
		app := createTestApplication(t)
		
		dbModule := &mockModuleAwareConfigModule{
			name:   "database",
			config: &DatabaseConfig{},
		}
		httpModule := &mockModuleAwareConfigModule{
			name:   "httpserver",
			config: &HTTPServerConfig{},
		}
		
		app.RegisterModule(dbModule)
		app.RegisterModule(httpModule)
		
		// Initialize the application
		err := app.Init()
		require.NoError(t, err)
		
		// Verify each module got its specific configuration
		dbConfig := dbModule.config.(*DatabaseConfig)
		assert.Equal(t, "db.example.com", dbConfig.Host)    // From DATABASE_HOST
		assert.Equal(t, 5432, dbConfig.Port)                // From DATABASE_PORT
		assert.Equal(t, 120, dbConfig.Timeout)              // From TIMEOUT_DATABASE
		
		httpConfig := httpModule.config.(*HTTPServerConfig)
		assert.Equal(t, "api.example.com", httpConfig.Host) // From HTTPSERVER_HOST
		assert.Equal(t, 9090, httpConfig.Port)              // From PORT_HTTPSERVER
		assert.Equal(t, 30, httpConfig.Timeout)             // From HTTPSERVER_TIMEOUT
	})
	
	t.Run("module_with_no_env_overrides", func(t *testing.T) {
		// Test that modules still work normally when no module-specific env vars are set
		
		type SimpleConfig struct {
			Name        string `env:"NAME" default:"default-name"`
			Value       int    `env:"VALUE" default:"42"`
			Enabled     bool   `env:"ENABLED"` // Remove default to avoid conflicts
		}
		
		// Clear all environment variables
		envVars := []string{
			"NAME", "SIMPLE_NAME", "NAME_SIMPLE",
			"VALUE", "SIMPLE_VALUE", "VALUE_SIMPLE",
			"ENABLED", "SIMPLE_ENABLED", "ENABLED_SIMPLE",
		}
		for _, env := range envVars {
			os.Unsetenv(env)
		}
		
		// Set only base environment variables
		testEnvVars := map[string]string{
			"NAME":    "global-name",
			"VALUE":   "100",
			"ENABLED": "false",
		}
		
		for key, value := range testEnvVars {
			err := os.Setenv(key, value)
			require.NoError(t, err)
		}
		
		defer func() {
			for _, env := range envVars {
				os.Unsetenv(env)
			}
		}()
		
		// Create application and register module
		app := createTestApplication(t)
		mockModule := &mockModuleAwareConfigModule{
			name:   "simple",
			config: &SimpleConfig{},
		}
		app.RegisterModule(mockModule)
		
		// Initialize the application
		err := app.Init()
		require.NoError(t, err)
		
		// Verify the configuration uses base environment variables (backward compatibility)
		config := mockModule.config.(*SimpleConfig)
		assert.Equal(t, "global-name", config.Name)
		assert.Equal(t, 100, config.Value)
		assert.False(t, config.Enabled)
	})
}