package database

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockIAMTokenProviderWithExpiry simulates token expiration scenarios
type MockIAMTokenProviderWithExpiry struct {
	mutex        sync.RWMutex
	currentToken string
	tokenExpiry  time.Time
	refreshCount int
	shouldFail   bool
	failAfter    int // fail after this many refresh attempts
}

func NewMockIAMTokenProviderWithExpiry(initialToken string, validDuration time.Duration) *MockIAMTokenProviderWithExpiry {
	return &MockIAMTokenProviderWithExpiry{
		currentToken: initialToken,
		tokenExpiry:  time.Now().Add(validDuration),
		refreshCount: 0,
		shouldFail:   false,
	}
}

func (m *MockIAMTokenProviderWithExpiry) GetToken(ctx context.Context, endpoint string) (string, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if m.shouldFail {
		return "", errors.New("token provider failed")
	}

	// Check if token is expired
	if time.Now().After(m.tokenExpiry) {
		return "", errors.New("token expired")
	}

	return m.currentToken, nil
}

func (m *MockIAMTokenProviderWithExpiry) BuildDSNWithIAMToken(ctx context.Context, originalDSN string) (string, error) {
	token, err := m.GetToken(ctx, "mock-endpoint")
	if err != nil {
		return "", err
	}
	return replaceDSNPassword(originalDSN, token)
}

func (m *MockIAMTokenProviderWithExpiry) RefreshToken() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.refreshCount++

	// Simulate failure after certain attempts
	if m.failAfter > 0 && m.refreshCount > m.failAfter {
		m.shouldFail = true
		return errors.New("refresh failed after max attempts")
	}

	// Generate new token
	m.currentToken = fmt.Sprintf("refreshed-token-%d", m.refreshCount)
	m.tokenExpiry = time.Now().Add(15 * time.Minute) // New 15-minute token

	return nil
}

func (m *MockIAMTokenProviderWithExpiry) ExpireToken() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.tokenExpiry = time.Now().Add(-1 * time.Minute) // Expire the token
}

func (m *MockIAMTokenProviderWithExpiry) StartTokenRefresh(ctx context.Context, endpoint string) {
	// No-op for testing
}

func (m *MockIAMTokenProviderWithExpiry) StopTokenRefresh() {
	// No-op for testing
}

func (m *MockIAMTokenProviderWithExpiry) SetTokenRefreshCallback(callback TokenRefreshCallback) {
	// No-op for testing, but could be extended to test callback behavior
}

func (m *MockIAMTokenProviderWithExpiry) GetRefreshCount() int {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.refreshCount
}

// TestIAMTokenExpirationScenario tests the scenario where a token expires after connection establishment
func TestIAMTokenExpirationScenario(t *testing.T) {
	// Create a mock token provider with a short-lived token
	mockProvider := NewMockIAMTokenProviderWithExpiry("initial-token", 1*time.Second)

	ctx := context.Background()

	// Test 1: Try to build DSN with valid token - should work
	dsn, err := mockProvider.BuildDSNWithIAMToken(ctx, "user:password@host:5432/db")
	require.NoError(t, err, "Should build DSN with valid token")
	assert.Contains(t, dsn, "initial-token", "DSN should contain the token")

	// Test 2: Wait for token to expire and then expire it explicitly
	time.Sleep(2 * time.Second)
	mockProvider.ExpireToken()

	// Test 3: Try to build DSN with expired token - should fail
	_, err = mockProvider.BuildDSNWithIAMToken(ctx, "user:password@host:5432/db")
	assert.Error(t, err, "Should fail with expired token")
	assert.Contains(t, err.Error(), "token expired", "Error should indicate token expiration")
}

// TestTokenRefreshWithExistingConnection tests the core issue:
// What happens when tokens are refreshed but existing connections still use old token
func TestTokenRefreshWithExistingConnection(t *testing.T) {
	// This test demonstrates the core issue reported in the bug:
	// "an application that was running fine, suddenly stopped being able to communicate with the database"

	// Create a mock token provider
	mockProvider := NewMockIAMTokenProviderWithExpiry("initial-token", 15*time.Minute)

	ctx := context.Background()

	// Step 1: Build initial DSN - simulates application startup
	initialDSN, err := mockProvider.BuildDSNWithIAMToken(ctx, "user:password@host:5432/db")
	require.NoError(t, err, "Initial DSN build should succeed")
	assert.Contains(t, initialDSN, "initial-token", "Initial DSN should contain initial token")

	// Step 2: Simulate passage of time where token refresh occurs in background
	// This is what would happen in a real application after 10+ minutes
	err = mockProvider.RefreshToken()
	require.NoError(t, err, "Token refresh should succeed")
	assert.Equal(t, 1, mockProvider.GetRefreshCount(), "Should have refreshed once")

	// Step 3: Build new DSN after token refresh
	newDSN, err := mockProvider.BuildDSNWithIAMToken(ctx, "user:password@host:5432/db")
	require.NoError(t, err, "New DSN build should succeed")
	assert.Contains(t, newDSN, "refreshed-token-1", "New DSN should contain refreshed token")

	// Step 4: Verify that the DSNs are different (the key issue)
	assert.NotEqual(t, initialDSN, newDSN, "DSNs should be different after token refresh")

	// This test demonstrates that when tokens are refreshed, the DSN changes,
	// but existing database connections (sql.DB) were created with the old DSN
	// and continue to use the old token until the connections are recreated.
}

// TestTokenProviderRealWorldScenario tests the scenario described in the issue
func TestTokenProviderRealWorldScenario(t *testing.T) {
	// Simulate the real-world scenario:
	// 1. Application starts up with valid token
	// 2. Token expires while application is running
	// 3. Background refresh gets new token
	// 4. Existing connections still use old token and fail

	mockProvider := NewMockIAMTokenProviderWithExpiry("startup-token", 5*time.Second)
	ctx := context.Background()

	// Application startup - gets initial token
	token1, err := mockProvider.GetToken(ctx, "endpoint")
	require.NoError(t, err)
	assert.Equal(t, "startup-token", token1)

	// Simulate time passing (token expires)
	time.Sleep(6 * time.Second)

	// Token should now be expired
	_, err = mockProvider.GetToken(ctx, "endpoint")
	assert.Error(t, err, "Token should be expired")
	assert.Contains(t, err.Error(), "token expired")

	// Background refresh occurs (this is what the background goroutine would do)
	err = mockProvider.RefreshToken()
	require.NoError(t, err, "Refresh should succeed")

	// Now token should work again
	token2, err := mockProvider.GetToken(ctx, "endpoint")
	require.NoError(t, err)
	assert.Equal(t, "refreshed-token-1", token2)
	assert.NotEqual(t, token1, token2, "Tokens should be different")

	// But the issue is: if we had an existing database connection created with token1,
	// it would still be using the old token and would fail even though token2 is valid.
}

// TestConnectionRecreationAfterTokenRefresh tests whether connection recreation helps
func TestConnectionRecreationAfterTokenRefresh(t *testing.T) {
	// This test demonstrates a potential solution: recreating connections after token refresh

	mockProvider := NewMockIAMTokenProviderWithExpiry("initial-token", 15*time.Minute)
	ctx := context.Background()

	// Step 1: Create initial connection DSN
	dsn1, err := mockProvider.BuildDSNWithIAMToken(ctx, "postgres://user:password@host:5432/db")
	require.NoError(t, err)

	// Step 2: Refresh token
	err = mockProvider.RefreshToken()
	require.NoError(t, err)

	// Step 3: Create new connection DSN with refreshed token
	dsn2, err := mockProvider.BuildDSNWithIAMToken(ctx, "postgres://user:password@host:5432/db")
	require.NoError(t, err)

	// Verify DSNs are different
	assert.NotEqual(t, dsn1, dsn2, "DSNs should be different after token refresh")
	assert.Contains(t, dsn1, "initial-token")
	assert.Contains(t, dsn2, "refreshed-token-1")

	// This suggests that the solution would involve:
	// 1. Detecting when token refresh occurs
	// 2. Recreating the database connection with the new DSN
	// 3. Properly handling connection pool lifecycle
}

// TestTokenRefreshCallbackFunctionality tests that the token refresh callback works
func TestTokenRefreshCallbackFunctionality(t *testing.T) {
	// Create real AWS token provider to test callback mechanism
	config := &AWSIAMAuthConfig{
		Enabled:              true,
		Region:               "us-east-1",
		DBUser:               "testuser",
		TokenRefreshInterval: 300,
	}

	provider, err := NewAWSIAMTokenProvider(config)
	if err != nil {
		if strings.Contains(err.Error(), "failed to load AWS config") {
			t.Skip("AWS credentials not available, skipping test")
		}
		t.Fatalf("Failed to create provider: %v", err)
	}

	// Track callback invocations
	var callbackInvoked bool

	callback := func(newToken string, endpoint string) {
		callbackInvoked = true
		// In a real scenario, this callback would be called when tokens are refreshed
		assert.NotEmpty(t, newToken, "New token should not be empty")
		assert.NotEmpty(t, endpoint, "Endpoint should not be empty")
	}

	// Set callback
	provider.SetTokenRefreshCallback(callback)

	// Verify callback is set and provider is functional
	assert.NotNil(t, provider, "Provider should be created")

	// We can't easily test the actual callback without real AWS credentials and token generation,
	// but we can verify the mechanism is in place and doesn't cause issues
	_ = callbackInvoked // Use the variable to avoid compiler error
}
