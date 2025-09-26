package feeders

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestInstanceAwareEnvFeederSetVerboseDebug tests the verbose debug functionality
func TestInstanceAwareEnvFeederSetVerboseDebug(t *testing.T) {
	feeder := NewInstanceAwareEnvFeeder(
		func(instanceKey string) string {
			return "DB_" + instanceKey + "_"
		},
	)

	// Test setting verbose debug to true
	feeder.SetVerboseDebug(true, nil)

	// Test setting verbose debug to false
	feeder.SetVerboseDebug(false, nil)

	// Since there's no public way to check the internal verboseDebug field,
	// we just verify the method runs without error
	assert.NotNil(t, feeder)
}

// Mock logger for testing verbose functionality
type MockVerboseLogger struct {
	DebugCalls []struct {
		Msg  string
		Args []any
	}
}

func (m *MockVerboseLogger) Debug(msg string, args ...any) {
	m.DebugCalls = append(m.DebugCalls, struct {
		Msg  string
		Args []any
	}{Msg: msg, Args: args})
}

func TestInstanceAwareEnvFeeder_SetVerboseDebug(t *testing.T) {
	tests := []struct {
		name           string
		enabled        bool
		expectLogEntry bool
	}{
		{
			name:           "enable verbose debug",
			enabled:        true,
			expectLogEntry: true,
		},
		{
			name:           "disable verbose debug",
			enabled:        false,
			expectLogEntry: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockLogger := &MockVerboseLogger{}
			feeder := NewInstanceAwareEnvFeeder(func(instanceKey string) string {
				return "DB_" + instanceKey + "_"
			})

			// Call SetVerboseDebug
			feeder.SetVerboseDebug(tt.enabled, mockLogger)

			// Verify the state
			assert.Equal(t, tt.enabled, feeder.verboseDebug)
			if tt.enabled {
				assert.Equal(t, mockLogger, feeder.logger)
				// Should have logged the enable message
				require.Len(t, mockLogger.DebugCalls, 1)
				assert.Equal(t, "Verbose instance-aware environment feeder debugging enabled", mockLogger.DebugCalls[0].Msg)
			} else {
				assert.Equal(t, mockLogger, feeder.logger)
				// Should not have logged anything when disabled
				assert.Empty(t, mockLogger.DebugCalls)
			}
		})
	}
}

func TestInstanceAwareEnvFeeder_Feed_WithVerboseDebug(t *testing.T) {
	type TestConfig struct {
		Driver string `env:"DRIVER"`
		DSN    string `env:"DSN"`
	}

	tests := []struct {
		name                string
		config              interface{}
		expectError         bool
		expectedLogContains []string // Check if these messages are included in logs
	}{
		{
			name:        "valid struct with verbose logging",
			config:      &TestConfig{},
			expectError: false,
			expectedLogContains: []string{
				"InstanceAwareEnvFeeder: Starting feed process (single instance)",
			},
		},
		{
			name:        "nil config with verbose logging",
			config:      nil,
			expectError: true,
			expectedLogContains: []string{
				"InstanceAwareEnvFeeder: Starting feed process (single instance)",
				"InstanceAwareEnvFeeder: Structure type is nil",
			},
		},
		{
			name:        "non-pointer config with verbose logging",
			config:      TestConfig{},
			expectError: true,
			expectedLogContains: []string{
				"InstanceAwareEnvFeeder: Starting feed process (single instance)",
				"InstanceAwareEnvFeeder: Structure is not a pointer",
			},
		},
		{
			name:        "non-struct config with verbose logging",
			config:      new(string),
			expectError: true,
			expectedLogContains: []string{
				"InstanceAwareEnvFeeder: Starting feed process (single instance)",
				"InstanceAwareEnvFeeder: Structure element is not a struct",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockLogger := &MockVerboseLogger{}
			feeder := NewInstanceAwareEnvFeeder(func(instanceKey string) string {
				return "DB_" + instanceKey + "_"
			})

			// Enable verbose debugging
			feeder.SetVerboseDebug(true, mockLogger)

			// Call Feed
			err := feeder.Feed(tt.config)

			// Verify error expectation
			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			// Verify that expected debug messages are present
			logMessages := make([]string, len(mockLogger.DebugCalls))
			for i, call := range mockLogger.DebugCalls {
				logMessages[i] = call.Msg
			}

			for _, expectedLog := range tt.expectedLogContains {
				found := false
				for _, logMsg := range logMessages {
					if logMsg == expectedLog {
						found = true
						break
					}
				}
				assert.True(t, found, "Expected to find log message: %s in %v", expectedLog, logMessages)
			}
		})
	}
}

func TestInstanceAwareEnvFeeder_FeedKey_WithVerboseDebug(t *testing.T) {
	type TestConfig struct {
		Driver string `env:"DRIVER"`
		DSN    string `env:"DSN"`
	}

	// Clean up environment
	defer cleanupInstanceTestEnv()

	// Set up environment variables
	os.Setenv("DB_PRIMARY_DRIVER", "postgres")
	os.Setenv("DB_PRIMARY_DSN", "postgres://localhost/primary")

	mockLogger := &MockVerboseLogger{}
	feeder := NewInstanceAwareEnvFeeder(func(instanceKey string) string {
		return "DB_" + instanceKey + "_"
	})

	// Enable verbose debugging
	feeder.SetVerboseDebug(true, mockLogger)

	config := &TestConfig{}

	// Call FeedKey
	err := feeder.FeedKey("primary", config)
	require.NoError(t, err)

	// Verify the configuration was loaded
	assert.Equal(t, "postgres", config.Driver)
	assert.Equal(t, "postgres://localhost/primary", config.DSN)

	// Verify verbose logging occurred
	assert.NotEmpty(t, mockLogger.DebugCalls, "Expected verbose debug calls")

	// Look for key verbose logging messages
	foundStartMessage := false
	foundCompletedMessage := false
	for _, call := range mockLogger.DebugCalls {
		if call.Msg == "InstanceAwareEnvFeeder: Starting FeedKey process" {
			foundStartMessage = true
		}
		if call.Msg == "InstanceAwareEnvFeeder: FeedKey completed successfully" {
			foundCompletedMessage = true
		}
	}

	assert.True(t, foundStartMessage, "Expected to find start message in debug logs")
	assert.True(t, foundCompletedMessage, "Expected to find completion message in debug logs")
}
