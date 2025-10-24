package reverseproxy

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCircuitBreakerConfiguredTimeout verifies that the circuit breaker uses
// the configured request timeout instead of the hardcoded 5 second default.
func TestCircuitBreakerConfiguredTimeout(t *testing.T) {
	tests := []struct {
		name            string
		configTimeout   time.Duration
		expectedTimeout time.Duration
	}{
		{
			name:            "uses configured 30s timeout",
			configTimeout:   30 * time.Second,
			expectedTimeout: 30 * time.Second,
		},
		{
			name:            "uses configured 10s timeout",
			configTimeout:   10 * time.Second,
			expectedTimeout: 10 * time.Second,
		},
		{
			name:            "defaults to 30s when not configured",
			configTimeout:   0,
			expectedTimeout: 30 * time.Second,
		},
		{
			name:            "uses configured 60s timeout",
			configTimeout:   60 * time.Second,
			expectedTimeout: 60 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create circuit breaker config with the specified timeout
			config := CircuitBreakerConfig{
				Enabled:          true,
				FailureThreshold: 5,
				OpenTimeout:      30 * time.Second,
				RequestTimeout:   tt.configTimeout,
			}

			// Create circuit breaker
			cb := NewCircuitBreakerWithConfig("test-backend", config, nil)

			// Verify the request timeout is set correctly
			assert.Equal(t, tt.expectedTimeout, cb.requestTimeout,
				"Circuit breaker should use the configured request timeout")
		})
	}
}

// TestCircuitBreakerRespectsConfiguredTimeout verifies that the circuit breaker
// actually applies the configured timeout when executing requests.
func TestCircuitBreakerRespectsConfiguredTimeout(t *testing.T) {
	// Create a slow backend that takes 2 seconds to respond
	slowBackend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("slow response"))
	}))
	defer slowBackend.Close()

	t.Run("request succeeds with 5s timeout", func(t *testing.T) {
		// Create circuit breaker with 5 second timeout (should succeed)
		config := CircuitBreakerConfig{
			Enabled:          true,
			FailureThreshold: 5,
			OpenTimeout:      30 * time.Second,
			RequestTimeout:   5 * time.Second,
		}
		cb := NewCircuitBreakerWithConfig("test-backend", config, nil)

		// Create a request
		req, err := http.NewRequest(http.MethodGet, slowBackend.URL, nil)
		require.NoError(t, err)

		// Execute the request through circuit breaker
		startTime := time.Now()
		resp, err := cb.Execute(req, func(r *http.Request) (*http.Response, error) {
			return http.DefaultClient.Do(r)
		})
		elapsed := time.Since(startTime)

		// Request should succeed (backend responds in 2s, timeout is 5s)
		require.NoError(t, err, "Request should succeed with 5s timeout")
		require.NotNil(t, resp, "Response should not be nil")
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Less(t, elapsed, 4*time.Second, "Request should complete in ~2 seconds")
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
	})

	t.Run("request fails with 1s timeout", func(t *testing.T) {
		// Create circuit breaker with 1 second timeout (should fail)
		config := CircuitBreakerConfig{
			Enabled:          true,
			FailureThreshold: 5,
			OpenTimeout:      30 * time.Second,
			RequestTimeout:   1 * time.Second,
		}
		cb := NewCircuitBreakerWithConfig("test-backend", config, nil)

		// Create a request
		req, err := http.NewRequest(http.MethodGet, slowBackend.URL, nil)
		require.NoError(t, err)

		// Execute the request through circuit breaker
		startTime := time.Now()
		resp, err := cb.Execute(req, func(r *http.Request) (*http.Response, error) {
			return http.DefaultClient.Do(r)
		})
		elapsed := time.Since(startTime)

		// Request should fail due to timeout (backend takes 2s, timeout is 1s)
		require.Error(t, err, "Request should fail with 1s timeout")
		assert.Contains(t, err.Error(), "context deadline exceeded", "Error should indicate timeout")
		assert.Nil(t, resp, "Response should be nil on timeout")
		assert.Less(t, elapsed, 2*time.Second, "Request should timeout in ~1 second")
	})

	t.Run("request succeeds with 30s timeout", func(t *testing.T) {
		// Create circuit breaker with 30 second timeout (should succeed)
		config := CircuitBreakerConfig{
			Enabled:          true,
			FailureThreshold: 5,
			OpenTimeout:      30 * time.Second,
			RequestTimeout:   30 * time.Second,
		}
		cb := NewCircuitBreakerWithConfig("test-backend", config, nil)

		// Create a request
		req, err := http.NewRequest(http.MethodGet, slowBackend.URL, nil)
		require.NoError(t, err)

		// Execute the request through circuit breaker
		startTime := time.Now()
		resp, err := cb.Execute(req, func(r *http.Request) (*http.Response, error) {
			return http.DefaultClient.Do(r)
		})
		elapsed := time.Since(startTime)

		// Request should succeed (backend responds in 2s, timeout is 30s)
		require.NoError(t, err, "Request should succeed with 30s timeout")
		require.NotNil(t, resp, "Response should not be nil")
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Less(t, elapsed, 4*time.Second, "Request should complete in ~2 seconds")
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
	})
}

// TestCircuitBreakerDoesNotOverrideParentContext verifies that when the parent
// context already has a shorter deadline, the circuit breaker respects it.
func TestCircuitBreakerDoesNotOverrideParentContext(t *testing.T) {
	// Create a backend that takes 2 seconds to respond
	slowBackend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer slowBackend.Close()

	// Create circuit breaker with 10 second timeout
	config := CircuitBreakerConfig{
		Enabled:          true,
		FailureThreshold: 5,
		OpenTimeout:      30 * time.Second,
		RequestTimeout:   10 * time.Second,
	}
	cb := NewCircuitBreakerWithConfig("test-backend", config, nil)

	// Create a request with a parent context that has a 1 second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	req, err := http.NewRequest(http.MethodGet, slowBackend.URL, nil)
	require.NoError(t, err)
	req = req.WithContext(ctx)

	// Execute the request through circuit breaker
	startTime := time.Now()
	resp, err := cb.Execute(req, func(r *http.Request) (*http.Response, error) {
		return http.DefaultClient.Do(r)
	})
	elapsed := time.Since(startTime)

	// Request should fail due to parent context timeout (1s, not 10s)
	require.Error(t, err, "Request should fail with parent context timeout")
	assert.Contains(t, err.Error(), "context deadline exceeded", "Error should indicate timeout")
	assert.Nil(t, resp, "Response should be nil on timeout")
	assert.Less(t, elapsed, 2*time.Second, "Request should timeout in ~1 second due to parent context")
}

// TestCircuitBreakerWithRequestTimeoutMethod verifies the WithRequestTimeout method works
func TestCircuitBreakerWithRequestTimeoutMethod(t *testing.T) {
	// Create a circuit breaker with default config
	cb := NewCircuitBreaker("test-backend", nil)

	// Default timeout should be 5s (from NewCircuitBreaker)
	assert.Equal(t, 5*time.Second, cb.requestTimeout)

	// Use WithRequestTimeout to change it
	cb = cb.WithRequestTimeout(45 * time.Second)
	assert.Equal(t, 45*time.Second, cb.requestTimeout)
}
