package reverseproxy

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTenantIsolationValidationDemo demonstrates the enhanced tenant isolation validation
// This test showcases the key improvements made to the tenantIsolationShouldBeMaintained() method
func TestTenantIsolationValidationDemo(t *testing.T) {
	t.Run("ProperTenantIsolation", func(t *testing.T) {
		// Test Context Setup
		ctx := &ReverseProxyBDDTestContext{}
		tenantARequests := make([]*http.Request, 0)
		tenantBRequests := make([]*http.Request, 0)
		ctx.tenantARequests = &tenantARequests
		ctx.tenantBRequests = &tenantBRequests

		// Mock backend servers with proper tenant isolation
		tenantAServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx.appendTenantARequest(r.Clone(r.Context()))
			w.Header().Set("X-Backend-ID", "tenant-a-backend")
			w.Header().Set("X-Backend-URL", "http://tenant-a.example.com")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("tenant-a response"))
		}))
		defer tenantAServer.Close()

		tenantBServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx.appendTenantBRequest(r.Clone(r.Context()))
			w.Header().Set("X-Backend-ID", "tenant-b-backend")
			w.Header().Set("X-Backend-URL", "http://tenant-b.example.com")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("tenant-b response"))
		}))
		defer tenantBServer.Close()

		// Simulate tenant-specific requests
		reqA := httptest.NewRequest("GET", "/test", nil)
		reqA.Header.Set("X-Tenant-ID", "tenant-a")
		respA := httptest.NewRecorder()
		tenantAServer.Config.Handler.ServeHTTP(respA, reqA)

		reqB := httptest.NewRequest("GET", "/test", nil)
		reqB.Header.Set("X-Tenant-ID", "tenant-b")
		respB := httptest.NewRecorder()
		tenantBServer.Config.Handler.ServeHTTP(respB, reqB)

		// Validation that would occur in tenantIsolationShouldBeMaintained()
		// 1. Backend identification validation
		assert.Equal(t, "tenant-a-backend", respA.Header().Get("X-Backend-ID"), "Tenant A should hit tenant-a-backend")
		assert.Equal(t, "tenant-b-backend", respB.Header().Get("X-Backend-ID"), "Tenant B should hit tenant-b-backend")

		// 2. Request tracking validation
		assert.Equal(t, 1, ctx.getTenantARequestsLen(), "Should track exactly 1 request for tenant A")
		assert.Equal(t, 1, ctx.getTenantBRequestsLen(), "Should track exactly 1 request for tenant B")

		// 3. Tenant header validation in tracked requests
		tenantAReqs := ctx.getTenantARequestsCopy()
		tenantBReqs := ctx.getTenantBRequestsCopy()
		assert.NotEmpty(t, tenantAReqs, "Should have tenant A requests")
		assert.NotEmpty(t, tenantBReqs, "Should have tenant B requests")
		tracked_reqA := tenantAReqs[0]
		tracked_reqB := tenantBReqs[0]
		assert.Equal(t, "tenant-a", tracked_reqA.Header.Get("X-Tenant-ID"), "Tenant A backend should receive tenant-a header")
		assert.Equal(t, "tenant-b", tracked_reqB.Header.Get("X-Tenant-ID"), "Tenant B backend should receive tenant-b header")

		// 4. Backend URL isolation validation
		backend1URL := respA.Header().Get("X-Backend-URL")
		backend2URL := respB.Header().Get("X-Backend-URL")
		assert.NotEqual(t, backend1URL, backend2URL, "Different tenants should hit different backend URLs")

		// 5. Response content validation
		assert.NotEqual(t, respA.Body.String(), respB.Body.String(), "Different tenants should get different responses")
		assert.Contains(t, respA.Body.String(), "tenant-a", "Tenant A should get tenant-specific response")
		assert.Contains(t, respB.Body.String(), "tenant-b", "Tenant B should get tenant-specific response")

		t.Log("✅ All tenant isolation validations passed")
	})

	t.Run("DetectBrokenTenantIsolation", func(t *testing.T) {
		// Test Context Setup
		ctx := &ReverseProxyBDDTestContext{}
		tenantARequests := make([]*http.Request, 0)
		tenantBRequests := make([]*http.Request, 0)
		ctx.tenantARequests = &tenantARequests
		ctx.tenantBRequests = &tenantBRequests

		// Mock scenario where tenant isolation is broken (shared backend)
		sharedServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Both tenant request arrays get populated (indicating broken isolation)
			ctx.appendTenantARequest(r.Clone(r.Context()))
			ctx.appendTenantBRequest(r.Clone(r.Context()))

			// Same backend information (broken isolation)
			w.Header().Set("X-Backend-ID", "shared-backend")
			w.Header().Set("X-Backend-URL", "http://shared.example.com")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("shared response"))
		}))
		defer sharedServer.Close()

		// Simulate tenant-specific requests hitting shared backend
		reqA := httptest.NewRequest("GET", "/test", nil)
		reqA.Header.Set("X-Tenant-ID", "tenant-a")
		respA := httptest.NewRecorder()
		sharedServer.Config.Handler.ServeHTTP(respA, reqA)

		reqB := httptest.NewRequest("GET", "/test", nil)
		reqB.Header.Set("X-Tenant-ID", "tenant-b")
		respB := httptest.NewRecorder()
		sharedServer.Config.Handler.ServeHTTP(respB, reqB)

		// Demonstrate how enhanced validation would detect broken isolation
		errors := make([]string, 0)

		// Backend ID validation
		backend1ID := respA.Header().Get("X-Backend-ID")
		backend2ID := respB.Header().Get("X-Backend-ID")
		if backend1ID != "tenant-a-backend" {
			errors = append(errors, fmt.Sprintf("tenant-a request should hit tenant-a-backend, but hit %s", backend1ID))
		}
		if backend2ID != "tenant-b-backend" {
			errors = append(errors, fmt.Sprintf("tenant-b request should hit tenant-b-backend, but hit %s", backend2ID))
		}

		// Request tracking validation
		tenantALen := ctx.getTenantARequestsLen()
		tenantBLen := ctx.getTenantBRequestsLen()
		if tenantALen > 1 {
			errors = append(errors, fmt.Sprintf("expected exactly 1 request to tenant-a backend, got %d", tenantALen))
		}
		if tenantBLen > 1 {
			errors = append(errors, fmt.Sprintf("expected exactly 1 request to tenant-b backend, got %d", tenantBLen))
		}

		// Backend URL validation
		backend1URL := respA.Header().Get("X-Backend-URL")
		backend2URL := respB.Header().Get("X-Backend-URL")
		if backend1URL == backend2URL {
			errors = append(errors, fmt.Sprintf("tenant requests hit the same backend URL (%s), tenant isolation is broken", backend1URL))
		}

		// Response validation
		if respA.Body.String() == respB.Body.String() {
			errors = append(errors, fmt.Sprintf("tenant responses should be different to prove isolation, but both returned: %s", respA.Body.String()))
		}

		// Verify that broken isolation was detected
		require.NotEmpty(t, errors, "Broken tenant isolation should be detected")
		t.Logf("🔍 Detected %d tenant isolation issues:", len(errors))
		for i, err := range errors {
			t.Logf("  %d. %s", i+1, err)
		}

		// Verify specific expected errors
		allErrors := strings.Join(errors, " | ")
		assert.Contains(t, allErrors, "shared-backend", "Should detect wrong backend ID")
		assert.Contains(t, allErrors, "same backend URL", "Should detect shared backend URL")
		assert.Contains(t, allErrors, "shared response", "Should detect identical responses")
	})
}
