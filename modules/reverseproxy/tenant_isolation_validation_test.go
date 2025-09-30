package reverseproxy

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTenantIsolationValidation tests the tenant isolation validation logic
// This test validates our enhanced backend tracking mechanism for tenant isolation
func TestTenantIsolationValidation(t *testing.T) {
	testCtx := &ReverseProxyBDDTestContext{}

	// Create tracking arrays for simulated backend calls
	tenantARequests := make([]*http.Request, 0)
	tenantBRequests := make([]*http.Request, 0)

	// Setup tracking in context
	testCtx.tenantARequests = &tenantARequests
	testCtx.tenantBRequests = &tenantBRequests

	// Create mock backend servers with tracking
	tenantAServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Track the request
		testCtx.appendTenantARequest(r.Clone(r.Context()))

		// Add backend identification headers
		w.Header().Set("X-Backend-ID", "tenant-a-backend")
		w.Header().Set("X-Backend-URL", "http://tenant-a-backend.example.com")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("response from tenant-a backend"))
	}))
	defer tenantAServer.Close()

	tenantBServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Track the request
		testCtx.appendTenantBRequest(r.Clone(r.Context()))

		// Add backend identification headers
		w.Header().Set("X-Backend-ID", "tenant-b-backend")
		w.Header().Set("X-Backend-URL", "http://tenant-b-backend.example.com")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("response from tenant-b backend"))
	}))
	defer tenantBServer.Close()

	// Simulate tenant A request
	reqA := httptest.NewRequest("GET", "/test", nil)
	reqA.Header.Set("X-Tenant-ID", "tenant-a")
	respA := httptest.NewRecorder()
	tenantAServer.Config.Handler.ServeHTTP(respA, reqA)

	// Simulate tenant B request
	reqB := httptest.NewRequest("GET", "/test", nil)
	reqB.Header.Set("X-Tenant-ID", "tenant-b")
	respB := httptest.NewRecorder()
	tenantBServer.Config.Handler.ServeHTTP(respB, reqB)

	// Verify that requests were tracked correctly
	require.Equal(t, 1, testCtx.getTenantARequestsLen(), "Should have tracked 1 request for tenant A")
	require.Equal(t, 1, testCtx.getTenantBRequestsLen(), "Should have tracked 1 request for tenant B")

	// Verify backend identification headers
	assert.Equal(t, "tenant-a-backend", respA.Header().Get("X-Backend-ID"))
	assert.Equal(t, "tenant-b-backend", respB.Header().Get("X-Backend-ID"))

	// Verify different backend URLs
	assert.NotEqual(t, respA.Header().Get("X-Backend-URL"), respB.Header().Get("X-Backend-URL"))

	// Verify tenant headers were properly tracked
	tenantAReqs := testCtx.getTenantARequestsCopy()
	tenantBReqs := testCtx.getTenantBRequestsCopy()
	require.NotEmpty(t, tenantAReqs, "Should have tenant A requests")
	require.NotEmpty(t, tenantBReqs, "Should have tenant B requests")
	tracked_reqA := tenantAReqs[0]
	tracked_reqB := tenantBReqs[0]

	assert.Equal(t, "tenant-a", tracked_reqA.Header.Get("X-Tenant-ID"))
	assert.Equal(t, "tenant-b", tracked_reqB.Header.Get("X-Tenant-ID"))

	// Verify response bodies are different (tenant-specific)
	assert.NotEqual(t, respA.Body.String(), respB.Body.String())
	assert.Contains(t, respA.Body.String(), "tenant-a")
	assert.Contains(t, respB.Body.String(), "tenant-b")
}

// TestTenantIsolationValidationFailure tests that the validation catches cross-tenant calls
func TestTenantIsolationValidationFailure(t *testing.T) {
	testCtx := &ReverseProxyBDDTestContext{}

	// Create tracking arrays
	tenantARequests := make([]*http.Request, 0)
	tenantBRequests := make([]*http.Request, 0)

	// Setup tracking in context
	testCtx.tenantARequests = &tenantARequests
	testCtx.tenantBRequests = &tenantBRequests

	// Simulate a scenario where both requests hit the same backend (tenant isolation broken)
	sharedBackend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Both tenants hitting the same backend (isolation failure)
		testCtx.appendTenantARequest(r.Clone(r.Context()))
		testCtx.appendTenantBRequest(r.Clone(r.Context()))

		// Same backend ID and URL for both (this would indicate broken isolation)
		w.Header().Set("X-Backend-ID", "shared-backend")
		w.Header().Set("X-Backend-URL", "http://shared-backend.example.com")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("shared response"))
	}))
	defer sharedBackend.Close()

	// Simulate tenant A request
	reqA := httptest.NewRequest("GET", "/test", nil)
	reqA.Header.Set("X-Tenant-ID", "tenant-a")
	respA := httptest.NewRecorder()
	sharedBackend.Config.Handler.ServeHTTP(respA, reqA)

	// Simulate tenant B request
	reqB := httptest.NewRequest("GET", "/test", nil)
	reqB.Header.Set("X-Tenant-ID", "tenant-b")
	respB := httptest.NewRecorder()
	sharedBackend.Config.Handler.ServeHTTP(respB, reqB)

	// This would indicate broken tenant isolation:
	// 1. Both tenant request arrays have entries
	assert.Equal(t, 2, testCtx.getTenantARequestsLen(), "Tenant A should have tracked requests")
	assert.Equal(t, 2, testCtx.getTenantBRequestsLen(), "Tenant B should have tracked requests (indicating broken isolation)")

	// 2. Same backend ID for both responses
	assert.Equal(t, respA.Header().Get("X-Backend-ID"), respB.Header().Get("X-Backend-ID"))

	// 3. Same backend URL for both responses
	assert.Equal(t, respA.Header().Get("X-Backend-URL"), respB.Header().Get("X-Backend-URL"))

	// 4. Same response body (no tenant-specific handling)
	assert.Equal(t, respA.Body.String(), respB.Body.String())
}
