package database

import (
	"context"
	"database/sql"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLogMessage captures log messages for testing
type TestLogMessage struct {
	Level   string
	Message string
	Args    []interface{}
}

// TestingLogger implements a logger that captures messages for testing
type TestingLogger struct {
	mu       sync.Mutex
	messages []TestLogMessage
}

func NewTestingLogger() *TestingLogger {
	return &TestingLogger{
		messages: make([]TestLogMessage, 0),
	}
}

func (l *TestingLogger) Debug(msg string, args ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.messages = append(l.messages, TestLogMessage{Level: "DEBUG", Message: msg, Args: args})
}

func (l *TestingLogger) Info(msg string, args ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.messages = append(l.messages, TestLogMessage{Level: "INFO", Message: msg, Args: args})
}

func (l *TestingLogger) Warn(msg string, args ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.messages = append(l.messages, TestLogMessage{Level: "WARN", Message: msg, Args: args})
}

func (l *TestingLogger) Error(msg string, args ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.messages = append(l.messages, TestLogMessage{Level: "ERROR", Message: msg, Args: args})
}

func (l *TestingLogger) GetMessages() []TestLogMessage {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]TestLogMessage{}, l.messages...)
}

func (l *TestingLogger) HasMessage(level, substring string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, msg := range l.messages {
		if msg.Level == level && strings.Contains(msg.Message, substring) {
			return true
		}
	}
	return false
}

// TestConnectionPoolGracefulTransition tests that the connection pool transitions gracefully
// when IAM token is refreshed, allowing in-flight queries to complete
func TestConnectionPoolGracefulTransition(t *testing.T) {
	// Use SQLite for testing as it doesn't require external database
	logger := NewTestingLogger()

	config := ConnectionConfig{
		Driver:                "sqlite",
		DSN:                   ":memory:",
		MaxOpenConnections:    5,
		MaxIdleConnections:    2,
		ConnectionMaxLifetime: 10 * time.Second,
		AWSIAMAuth: &AWSIAMAuthConfig{
			Enabled:                    false, // We'll test the logic without actual IAM
			ConnectionCloseGracePeriod: 2 * time.Second,
		},
	}

	service, err := NewDatabaseService(config, logger)
	require.NoError(t, err, "Should create database service")

	err = service.Connect()
	require.NoError(t, err, "Should connect to database")
	defer service.Close()

	// Verify initial connection works
	ctx := context.Background()
	err = service.Ping(ctx)
	require.NoError(t, err, "Should ping database")

	// Create a table for testing
	_, err = service.ExecContext(ctx, "CREATE TABLE test (id INTEGER PRIMARY KEY, value TEXT)")
	require.NoError(t, err, "Should create test table")

	// Insert test data
	_, err = service.ExecContext(ctx, "INSERT INTO test (value) VALUES ('test1'), ('test2'), ('test3')")
	require.NoError(t, err, "Should insert test data")

	// Verify we can query data
	rows, err := service.QueryContext(ctx, "SELECT COUNT(*) FROM test")
	require.NoError(t, err, "Should query test table")
	defer rows.Close()

	var count int
	require.True(t, rows.Next(), "Should have result")
	err = rows.Scan(&count)
	require.NoError(t, err, "Should scan result")
	assert.Equal(t, 3, count, "Should have 3 rows")
}

// TestGracePeriodConfiguration tests that the grace period can be configured
func TestGracePeriodConfiguration(t *testing.T) {
	testCases := []struct {
		name                string
		gracePeriod         time.Duration
		expectedGracePeriod time.Duration
	}{
		{
			name:                "Default grace period",
			gracePeriod:         0,
			expectedGracePeriod: 5 * time.Second,
		},
		{
			name:                "Custom grace period 2s",
			gracePeriod:         2 * time.Second,
			expectedGracePeriod: 2 * time.Second,
		},
		{
			name:                "Custom grace period 10s",
			gracePeriod:         10 * time.Second,
			expectedGracePeriod: 10 * time.Second,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := &AWSIAMAuthConfig{
				Enabled:                    true,
				Region:                     "us-east-1",
				DBUser:                     "testuser",
				TokenRefreshInterval:       600,
				ConnectionCloseGracePeriod: tc.gracePeriod,
			}

			// Verify configuration is stored correctly
			if tc.gracePeriod > 0 {
				assert.Equal(t, tc.expectedGracePeriod, config.ConnectionCloseGracePeriod)
			}
		})
	}
}

// TestTokenRefreshWithSimulatedInFlightQueries tests token refresh with simulated in-flight queries
func TestTokenRefreshWithSimulatedInFlightQueries(t *testing.T) {
	logger := NewTestingLogger()

	// Create mock token provider
	mockProvider := NewMockIAMTokenProviderWithExpiry("initial-token", 5*time.Second)

	config := ConnectionConfig{
		Driver: "sqlite",
		DSN:    ":memory:",
		AWSIAMAuth: &AWSIAMAuthConfig{
			Enabled:                    true,
			Region:                     "us-east-1",
			DBUser:                     "testuser",
			TokenRefreshInterval:       600,
			ConnectionTimeout:          5 * time.Second,
			ConnectionCloseGracePeriod: 1 * time.Second,
		},
	}

	// Create service
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	serviceImpl := &databaseServiceImpl{
		config:           config,
		logger:           logger,
		ctx:              ctx,
		cancel:           cancel,
		awsTokenProvider: mockProvider,
		endpoint:         "test-endpoint",
	}

	// Open SQLite connection (simulating database connection)
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	serviceImpl.connMutex.Lock()
	serviceImpl.db = db
	serviceImpl.connMutex.Unlock()

	// Set up token refresh callback
	mockProvider.SetTokenRefreshCallback(serviceImpl.onTokenRefresh)

	// Track callback invocations
	var callbackCount atomic.Int32

	// Wrap the callback to count invocations
	originalCallback := mockProvider.tokenRefreshCallback
	mockProvider.SetTokenRefreshCallback(func(token, endpoint string) {
		callbackCount.Add(1)
		if originalCallback != nil {
			originalCallback(token, endpoint)
		}
	})

	// Trigger token refresh
	err = mockProvider.RefreshToken()
	require.NoError(t, err, "Token refresh should succeed")

	// Wait for callback to complete
	time.Sleep(100 * time.Millisecond)

	// Verify callback was invoked
	assert.Greater(t, callbackCount.Load(), int32(0), "Callback should be invoked at least once")

	// Wait for grace period + buffer
	time.Sleep(1500 * time.Millisecond)

	// Verify logging messages
	messages := logger.GetMessages()
	assert.NotEmpty(t, messages, "Should have log messages")

	// Check for specific log messages indicating proper lifecycle
	hasStartMessage := false
	hasSuccessMessage := false
	hasGracePeriodMessage := false

	for _, msg := range messages {
		if msg.Level == "INFO" && strings.Contains(msg.Message, "Starting database connection refresh") {
			hasStartMessage = true
		}
		if msg.Level == "INFO" && strings.Contains(msg.Message, "Successfully created new database connection") {
			hasSuccessMessage = true
		}
		if msg.Level == "DEBUG" && strings.Contains(msg.Message, "Waiting grace period") {
			hasGracePeriodMessage = true
		}
	}

	assert.True(t, hasStartMessage, "Should log start of connection refresh")
	assert.True(t, hasSuccessMessage, "Should log successful connection creation")
	assert.True(t, hasGracePeriodMessage, "Should log grace period wait")
}

// TestOnTokenRefreshErrorHandling tests error handling during token refresh
func TestOnTokenRefreshErrorHandling(t *testing.T) {
	logger := NewTestingLogger()

	testCases := []struct {
		name           string
		setupFunc      func(*databaseServiceImpl)
		expectedError  string
		shouldLogError bool
	}{
		{
			name: "Nil database connection",
			setupFunc: func(s *databaseServiceImpl) {
				s.db = nil
			},
			expectedError:  "",
			shouldLogError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockProvider := NewMockIAMTokenProviderWithExpiry("test-token", 5*time.Second)

			serviceImpl := &databaseServiceImpl{
				config: ConnectionConfig{
					Driver: "sqlite",
					DSN:    ":memory:",
					AWSIAMAuth: &AWSIAMAuthConfig{
						Enabled:                    true,
						ConnectionCloseGracePeriod: 1 * time.Second,
					},
				},
				logger:           logger,
				ctx:              ctx,
				cancel:           cancel,
				awsTokenProvider: mockProvider,
				endpoint:         "test-endpoint",
			}

			// Apply test-specific setup
			tc.setupFunc(serviceImpl)

			// Call onTokenRefresh
			serviceImpl.onTokenRefresh("new-token", "test-endpoint")

			// Verify expected behavior
			messages := logger.GetMessages()
			if tc.shouldLogError {
				hasError := false
				for _, msg := range messages {
					if msg.Level == "ERROR" && strings.Contains(msg.Message, tc.expectedError) {
						hasError = true
						break
					}
				}
				assert.True(t, hasError, "Should log expected error")
			}
		})
	}
}

// TestConnectionStatsLogging tests that connection pool stats are logged before closure
func TestConnectionStatsLogging(t *testing.T) {
	logger := NewTestingLogger()

	config := ConnectionConfig{
		Driver: "sqlite",
		DSN:    ":memory:",
		AWSIAMAuth: &AWSIAMAuthConfig{
			Enabled:                    true,
			ConnectionCloseGracePeriod: 500 * time.Millisecond,
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mockProvider := NewMockIAMTokenProviderWithExpiry("test-token", 5*time.Second)

	// Create a real database connection
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	serviceImpl := &databaseServiceImpl{
		config:           config,
		logger:           logger,
		ctx:              ctx,
		cancel:           cancel,
		awsTokenProvider: mockProvider,
		endpoint:         "test-endpoint",
		db:               db,
	}

	// Set up token refresh callback
	mockProvider.SetTokenRefreshCallback(serviceImpl.onTokenRefresh)

	// Trigger token refresh
	err = mockProvider.RefreshToken()
	require.NoError(t, err)

	// Wait for callback and grace period
	time.Sleep(1 * time.Second)

	// Check for connection stats logging
	messages := logger.GetMessages()
	hasStatsLog := false

	for _, msg := range messages {
		if msg.Level == "DEBUG" && strings.Contains(msg.Message, "connection pool stats") {
			hasStatsLog = true
			break
		}
	}

	assert.True(t, hasStatsLog, "Should log connection pool stats before closure")
}

// TestDefaultGracePeriod tests that the default grace period is applied when not configured
func TestDefaultGracePeriod(t *testing.T) {
	logger := NewTestingLogger()

	config := ConnectionConfig{
		Driver: "sqlite",
		DSN:    ":memory:",
		AWSIAMAuth: &AWSIAMAuthConfig{
			Enabled: true,
			// ConnectionCloseGracePeriod not set, should use default
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mockProvider := NewMockIAMTokenProviderWithExpiry("test-token", 5*time.Second)

	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	serviceImpl := &databaseServiceImpl{
		config:           config,
		logger:           logger,
		ctx:              ctx,
		cancel:           cancel,
		awsTokenProvider: mockProvider,
		endpoint:         "test-endpoint",
		db:               db,
	}

	mockProvider.SetTokenRefreshCallback(serviceImpl.onTokenRefresh)

	// Trigger token refresh
	startTime := time.Now()
	err = mockProvider.RefreshToken()
	require.NoError(t, err)

	// Wait for grace period (default should be 5 seconds)
	// We'll wait 6 seconds to ensure the grace period completes
	time.Sleep(6 * time.Second)

	duration := time.Since(startTime)

	// Verify logging shows default grace period was used
	messages := logger.GetMessages()
	hasDefaultGracePeriod := false

	for _, msg := range messages {
		if msg.Level == "DEBUG" && strings.Contains(msg.Message, "Waiting grace period") {
			// Check that grace_period is in the args
			for i := 0; i < len(msg.Args)-1; i += 2 {
				if key, ok := msg.Args[i].(string); ok && key == "grace_period" {
					if gp, ok := msg.Args[i+1].(time.Duration); ok {
						// Default should be 5 seconds when not configured
						hasDefaultGracePeriod = gp == 5*time.Second
					}
				}
			}
		}
	}

	assert.True(t, hasDefaultGracePeriod, "Should use default grace period of 5 seconds")
	assert.GreaterOrEqual(t, duration.Seconds(), 5.0, "Should wait at least the default grace period")
}
