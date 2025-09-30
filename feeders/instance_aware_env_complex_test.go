package feeders

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestInstanceAwareEnvFeederComplexFeeder tests the ComplexFeeder interface
func TestInstanceAwareEnvFeederComplexFeeder(t *testing.T) {
	type ConnectionConfig struct {
		Host string `env:"HOST"`
		Port int    `env:"PORT"`
	}

	// Set up environment for database instance
	clearTestEnv(t)
	err := os.Setenv("DATABASE_HOST", "db.example.com")
	require.NoError(t, err)
	err = os.Setenv("DATABASE_PORT", "5432")
	require.NoError(t, err)

	defer func() {
		os.Unsetenv("DATABASE_HOST")
		os.Unsetenv("DATABASE_PORT")
	}()

	config := &ConnectionConfig{}
	feeder := NewInstanceAwareEnvFeeder(func(instanceKey string) string {
		return instanceKey + "_"
	})

	// Test FeedKey method (ComplexFeeder interface)
	err = feeder.FeedKey("database", config)
	require.NoError(t, err)

	assert.Equal(t, "db.example.com", config.Host)
	assert.Equal(t, 5432, config.Port)
}

// TestInstanceAwareEnvFeederFeedKey tests the FeedKey method with various scenarios
func TestInstanceAwareEnvFeederFeedKey(t *testing.T) {
	type TestConfig struct {
		Driver   string `env:"DRIVER"`
		DSN      string `env:"DSN"`
		Username string `env:"USERNAME"`
	}

	tests := []struct {
		name           string
		instanceKey    string
		envVars        map[string]string
		expectedConfig TestConfig
	}{
		{
			name:        "feed_key_with_values",
			instanceKey: "primary",
			envVars: map[string]string{
				"DB_PRIMARY_DRIVER":   "postgres",
				"DB_PRIMARY_DSN":      "postgres://localhost/primary",
				"DB_PRIMARY_USERNAME": "primary_user",
			},
			expectedConfig: TestConfig{
				Driver:   "postgres",
				DSN:      "postgres://localhost/primary",
				Username: "primary_user",
			},
		},
		{
			name:        "feed_key_with_missing_values",
			instanceKey: "secondary",
			envVars: map[string]string{
				"DB_SECONDARY_DRIVER": "mysql",
				// Missing DSN and USERNAME
			},
			expectedConfig: TestConfig{
				Driver:   "mysql",
				DSN:      "",
				Username: "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up environment
			defer cleanupInstanceTestEnv()

			// Set up environment variables
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}

			// Create feeder
			feeder := NewInstanceAwareEnvFeeder(
				func(instanceKey string) string {
					return "DB_" + instanceKey + "_"
				},
			)

			// Create config instance
			config := &TestConfig{}

			// Feed the specific key
			err := feeder.FeedKey(tt.instanceKey, config)
			require.NoError(t, err)

			// Verify the configuration
			assert.Equal(t, tt.expectedConfig.Driver, config.Driver)
			assert.Equal(t, tt.expectedConfig.DSN, config.DSN)
			assert.Equal(t, tt.expectedConfig.Username, config.Username)
		})
	}
}

// TestInstanceAwareEnvFeederComplexTypes tests feeding complex types
func TestInstanceAwareEnvFeederComplexTypes(t *testing.T) {
	type NestedConfig struct {
		Timeout string `env:"TIMEOUT"`
		Retries string `env:"RETRIES"`
	}

	type ComplexConfig struct {
		Name      string        `env:"NAME"`
		Port      string        `env:"PORT"`
		Nested    NestedConfig  // No env tag - should be processed as nested struct
		NestedPtr *NestedConfig `env:"NESTED_PTR"`
	}

	// Clean up environment
	defer cleanupInstanceTestEnv()

	// Set up environment variables
	envVars := map[string]string{
		"APP_PRIMARY_NAME":    "Primary App",
		"APP_PRIMARY_PORT":    "8080",
		"APP_PRIMARY_TIMEOUT": "30s",
		"APP_PRIMARY_RETRIES": "3",
	}

	for key, value := range envVars {
		os.Setenv(key, value)
	}

	// Create feeder
	feeder := NewInstanceAwareEnvFeeder(
		func(instanceKey string) string {
			return "APP_" + instanceKey + "_"
		},
	)

	// Create config instance
	config := &ComplexConfig{}

	// Feed the configuration
	err := feeder.FeedKey("primary", config)
	require.NoError(t, err)

	// Verify the configuration
	assert.Equal(t, "Primary App", config.Name)
	assert.Equal(t, "8080", config.Port)
	assert.Equal(t, "30s", config.Nested.Timeout)
	assert.Equal(t, "3", config.Nested.Retries)
}
