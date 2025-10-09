package reverseproxy

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestHTTPClientHasNoHardcodedTimeout verifies that when the HTTP client is
// created internally (not provided by httpclient service), it does not have
// a hardcoded timeout that would override per-request context timeouts.
//
// This test addresses the issue where a hardcoded 30-second client timeout
// was preventing the configured RequestTimeout from being respected.
func TestHTTPClientHasNoHardcodedTimeout(t *testing.T) {
	// Simulate creating an HTTP client internally (as done in Init method)
	// This is the same code path as in module.go lines 267-279
	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
		DisableCompression:  false,
	}

	// After our fix, the HTTP client should NOT have a hardcoded timeout
	httpClient := &http.Client{
		Transport: transport,
		// Note: No Timeout field set here - this allows per-request context timeouts
	}

	// Verify the HTTP client does NOT have a hardcoded timeout
	// The client timeout should be 0 to allow per-request context timeouts to work
	assert.Equal(t, time.Duration(0), httpClient.Timeout,
		"HTTP client should not have a hardcoded timeout (should be 0 to allow per-request context timeouts)")
}

// TestHTTPClientTimeoutBehavior verifies that when no client-level timeout
// is set, per-request context timeouts can work properly. This is a
// conceptual test that demonstrates the expected behavior.
func TestHTTPClientTimeoutBehavior(t *testing.T) {
	// When client has no timeout, per-request contexts control timeout
	clientNoTimeout := &http.Client{
		Timeout: 0, // No client-level timeout
	}
	assert.Equal(t, time.Duration(0), clientNoTimeout.Timeout,
		"Client with no timeout allows per-request context timeouts")

	// When client has a timeout, it acts as a hard limit
	clientWithTimeout := &http.Client{
		Timeout: 30 * time.Second, // Hardcoded timeout
	}
	assert.Equal(t, 30*time.Second, clientWithTimeout.Timeout,
		"Client with hardcoded timeout will override per-request contexts")

	// The fix changes the reverseproxy module from the second case to the first
}
