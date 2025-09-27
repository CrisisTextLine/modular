package feeders

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestInstanceAwareEnvFeederErrors tests error conditions
func TestInstanceAwareEnvFeederErrors(t *testing.T) {
	tests := []struct {
		name        string
		input       interface{}
		expectError bool
	}{
		{
			name:        "nil_input",
			input:       nil,
			expectError: true,
		},
		{
			name:        "non_pointer_input",
			input:       struct{}{},
			expectError: true,
		},
		{
			name:        "non_struct_pointer",
			input:       new(string),
			expectError: true,
		},
		{
			name: "valid_struct_pointer",
			input: &struct {
				Field string `env:"FIELD"`
			}{},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			feeder := NewInstanceAwareEnvFeeder(nil)
			err := feeder.Feed(tt.input)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestInstanceAwareEnvFeederErrorHandling(t *testing.T) {
	type TestConfig struct {
		Value string `env:"VALUE"`
	}

	feeder := NewInstanceAwareEnvFeeder(
		func(instanceKey string) string {
			return instanceKey + "_"
		},
	)

	tests := []struct {
		name          string
		config        interface{}
		shouldError   bool
		expectedError string
	}{
		{
			name:          "nil_config",
			config:        nil,
			shouldError:   true,
			expectedError: "env: invalid structure",
		},
		{
			name:          "non_pointer_config",
			config:        TestConfig{},
			shouldError:   true,
			expectedError: "env: invalid structure",
		},
		{
			name:          "pointer_to_non_struct",
			config:        &[]string{},
			shouldError:   true,
			expectedError: "env: invalid structure",
		},
		{
			name:        "valid_config",
			config:      &TestConfig{},
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := feeder.Feed(tt.config)

			if tt.shouldError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Helper function to clean up test environment variables
func cleanupInstanceTestEnv() {
	envVars := []string{
		"DB_PRIMARY_DRIVER", "DB_PRIMARY_DSN", "DB_PRIMARY_USERNAME",
		"DB_SECONDARY_DRIVER", "DB_SECONDARY_DSN", "DB_SECONDARY_USERNAME",
		"APP_PRIMARY_NAME", "APP_PRIMARY_PORT", "APP_PRIMARY_TIMEOUT", "APP_PRIMARY_RETRIES",
	}

	for _, envVar := range envVars {
		os.Unsetenv(envVar)
	}
}
