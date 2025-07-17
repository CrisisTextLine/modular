package reverseproxy

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPathRewritingEdgeCases tests edge cases for path rewriting functionality
func TestPathRewritingEdgeCases(t *testing.T) {
	// Track what path the backend receives
	var receivedPath string

	// Create a mock backend server that captures the path
	backendServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		response := map[string]string{
			"message": "backend response",
			"path":    r.URL.Path,
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer backendServer.Close()

	// Create a reverse proxy module
	module := NewModule()

	// Test Case 1: Empty path rewriting configuration
	t.Run("EmptyPathRewritingConfig", func(t *testing.T) {
		// Reset received path
		receivedPath = ""

		// Configure the module with empty path rewriting
		module.config = &ReverseProxyConfig{
			BackendServices: map[string]string{
				"api": backendServer.URL,
			},
			DefaultBackend: "api",
			TenantIDHeader: "X-Tenant-ID",
			PathRewriting:  PathRewritingConfig{
				// All empty
			},
		}

		// Create the reverse proxy
		backendURL, err := url.Parse(backendServer.URL)
		require.NoError(t, err)
		proxy := module.createReverseProxy(backendURL)

		// Create a request
		req := httptest.NewRequest("GET", "http://client.example.com/api/users/123", nil)
		req.Host = "client.example.com"

		// Process the request through the proxy
		w := httptest.NewRecorder()
		proxy.ServeHTTP(w, req)

		// Verify the response
		resp := w.Result()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// The backend should receive the original path unchanged
		assert.Equal(t, "/api/users/123", receivedPath,
			"Backend should receive original path when no path rewriting is configured")
	})

	// Test Case 2: Strip base path that doesn't match
	t.Run("StripBasePathNoMatch", func(t *testing.T) {
		// Reset received path
		receivedPath = ""

		// Configure the module to strip a base path that won't match
		module.config = &ReverseProxyConfig{
			BackendServices: map[string]string{
				"api": backendServer.URL,
			},
			DefaultBackend: "api",
			TenantIDHeader: "X-Tenant-ID",
			PathRewriting: PathRewritingConfig{
				StripBasePath: "/v2",
			},
		}

		// Create the reverse proxy
		backendURL, err := url.Parse(backendServer.URL)
		require.NoError(t, err)
		proxy := module.createReverseProxy(backendURL)

		// Create a request with a path that doesn't match the strip pattern
		req := httptest.NewRequest("GET", "http://client.example.com/v1/users/123", nil)
		req.Host = "client.example.com"

		// Process the request through the proxy
		w := httptest.NewRecorder()
		proxy.ServeHTTP(w, req)

		// Verify the response
		resp := w.Result()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// The backend should receive the original path unchanged
		assert.Equal(t, "/v1/users/123", receivedPath,
			"Backend should receive original path when strip base path doesn't match")
	})

	// Test Case 3: Root path handling
	t.Run("RootPathHandling", func(t *testing.T) {
		// Reset received path
		receivedPath = ""

		// Configure the module with base path rewriting
		module.config = &ReverseProxyConfig{
			BackendServices: map[string]string{
				"api": backendServer.URL,
			},
			DefaultBackend: "api",
			TenantIDHeader: "X-Tenant-ID",
			PathRewriting: PathRewritingConfig{
				BasePathRewrite: "/api",
			},
		}

		// Create the reverse proxy
		backendURL, err := url.Parse(backendServer.URL)
		require.NoError(t, err)
		proxy := module.createReverseProxy(backendURL)

		// Create a request to the root path
		req := httptest.NewRequest("GET", "http://client.example.com/", nil)
		req.Host = "client.example.com"

		// Process the request through the proxy
		w := httptest.NewRecorder()
		proxy.ServeHTTP(w, req)

		// Verify the response
		resp := w.Result()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// The backend should receive the base path prepended to root
		assert.Equal(t, "/api/", receivedPath,
			"Backend should receive base path prepended to root path")
	})

	// Test Case 4: Multiple slashes normalization
	t.Run("MultipleSlashesNormalization", func(t *testing.T) {
		// Reset received path
		receivedPath = ""

		// Configure the module with base path rewriting
		module.config = &ReverseProxyConfig{
			BackendServices: map[string]string{
				"api": backendServer.URL,
			},
			DefaultBackend: "api",
			TenantIDHeader: "X-Tenant-ID",
			PathRewriting: PathRewritingConfig{
				BasePathRewrite: "/api/",
			},
		}

		// Create the reverse proxy
		backendURL, err := url.Parse(backendServer.URL)
		require.NoError(t, err)
		proxy := module.createReverseProxy(backendURL)

		// Create a request with multiple slashes
		req := httptest.NewRequest("GET", "http://client.example.com//users///123", nil)
		req.Host = "client.example.com"

		// Process the request through the proxy
		w := httptest.NewRecorder()
		proxy.ServeHTTP(w, req)

		// Verify the response
		resp := w.Result()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// The backend should receive the path with base path prepended
		assert.Equal(t, "/api//users///123", receivedPath,
			"Backend should receive base path prepended, preserving original path structure")
	})

	// Test Case 5: Strip entire path
	t.Run("StripEntirePath", func(t *testing.T) {
		// Reset received path
		receivedPath = ""

		// Configure the module to strip a base path that matches the entire path
		module.config = &ReverseProxyConfig{
			BackendServices: map[string]string{
				"api": backendServer.URL,
			},
			DefaultBackend: "api",
			TenantIDHeader: "X-Tenant-ID",
			PathRewriting: PathRewritingConfig{
				StripBasePath: "/api/users",
			},
		}

		// Create the reverse proxy
		backendURL, err := url.Parse(backendServer.URL)
		require.NoError(t, err)
		proxy := module.createReverseProxy(backendURL)

		// Create a request where the entire path matches the strip pattern
		req := httptest.NewRequest("GET", "http://client.example.com/api/users", nil)
		req.Host = "client.example.com"

		// Process the request through the proxy
		w := httptest.NewRecorder()
		proxy.ServeHTTP(w, req)

		// Verify the response
		resp := w.Result()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// The backend should receive root path
		assert.Equal(t, "/", receivedPath,
			"Backend should receive root path when entire path is stripped")
	})

	// Test Case 6: Nil configuration handling
	t.Run("NilConfigHandling", func(t *testing.T) {
		// Reset received path
		receivedPath = ""

		// Set config to nil
		module.config = nil

		// Create the reverse proxy
		backendURL, err := url.Parse(backendServer.URL)
		require.NoError(t, err)
		proxy := module.createReverseProxy(backendURL)

		// Create a request
		req := httptest.NewRequest("GET", "http://client.example.com/api/users/123", nil)
		req.Host = "client.example.com"

		// Process the request through the proxy
		w := httptest.NewRecorder()
		proxy.ServeHTTP(w, req)

		// Verify the response
		resp := w.Result()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// The backend should receive the original path unchanged
		assert.Equal(t, "/api/users/123", receivedPath,
			"Backend should receive original path when config is nil")
	})
}

// TestPathRewritingPatternMatching tests the pattern matching logic
func TestPathRewritingPatternMatching(t *testing.T) {
	// Create a reverse proxy module
	module := NewModule()

	// Test exact matches
	t.Run("ExactMatches", func(t *testing.T) {
		assert.True(t, module.matchesPattern("/api/users", "/api/users"),
			"Should match exact path")
		assert.False(t, module.matchesPattern("/api/users/123", "/api/users"),
			"Should not match exact path with extra suffix")
		assert.False(t, module.matchesPattern("/api/user", "/api/users"),
			"Should not match exact path with different suffix")
	})

	// Test wildcard matches
	t.Run("WildcardMatches", func(t *testing.T) {
		assert.True(t, module.matchesPattern("/api/users/123", "/api/users/*"),
			"Should match wildcard pattern")
		assert.True(t, module.matchesPattern("/api/users/123/profile", "/api/users/*"),
			"Should match wildcard pattern with multiple segments")
		assert.True(t, module.matchesPattern("/api/users/", "/api/users/*"),
			"Should match wildcard pattern with trailing slash")
		assert.False(t, module.matchesPattern("/api/user", "/api/users/*"),
			"Should not match wildcard pattern with different prefix")
	})

	// Test star-only matches
	t.Run("StarOnlyMatches", func(t *testing.T) {
		assert.True(t, module.matchesPattern("/api/users123", "/api/users*"),
			"Should match star pattern")
		assert.True(t, module.matchesPattern("/api/users/123", "/api/users*"),
			"Should match star pattern with slash")
		assert.False(t, module.matchesPattern("/api/user", "/api/users*"),
			"Should not match star pattern with different prefix")
	})
}

// TestPathRewritingPatternReplacement tests the pattern replacement logic
func TestPathRewritingPatternReplacement(t *testing.T) {
	// Create a reverse proxy module
	module := NewModule()

	// Test exact replacements
	t.Run("ExactReplacements", func(t *testing.T) {
		result := module.applyPatternReplacement("/api/users", "/api/users", "/internal/users")
		assert.Equal(t, "/internal/users", result,
			"Should replace exact match completely")

		result = module.applyPatternReplacement("/api/users/123", "/api/users", "/internal/users")
		assert.Equal(t, "/api/users/123", result,
			"Should not replace when pattern doesn't match exactly")
	})

	// Test wildcard replacements
	t.Run("WildcardReplacements", func(t *testing.T) {
		result := module.applyPatternReplacement("/api/users/123", "/api/users/*", "/internal/users")
		assert.Equal(t, "/internal/users/123", result,
			"Should replace wildcard pattern with suffix preserved")

		result = module.applyPatternReplacement("/api/users/123/profile", "/api/users/*", "/internal/users")
		assert.Equal(t, "/internal/users/123/profile", result,
			"Should replace wildcard pattern with full suffix preserved")

		result = module.applyPatternReplacement("/api/users/", "/api/users/*", "/internal/users")
		assert.Equal(t, "/internal/users/", result,
			"Should replace wildcard pattern preserving trailing slash")
	})

	// Test star-only replacements
	t.Run("StarOnlyReplacements", func(t *testing.T) {
		result := module.applyPatternReplacement("/api/users123", "/api/users*", "/internal/users")
		assert.Equal(t, "/internal/users123", result,
			"Should replace star pattern with suffix preserved")

		result = module.applyPatternReplacement("/api/users/123", "/api/users*", "/internal/users")
		assert.Equal(t, "/internal/users/123", result,
			"Should replace star pattern with slash and suffix preserved")
	})

	// Test no match cases
	t.Run("NoMatchCases", func(t *testing.T) {
		result := module.applyPatternReplacement("/api/orders", "/api/users/*", "/internal/users")
		assert.Equal(t, "/api/orders", result,
			"Should return original path when pattern doesn't match")
	})
}

// TestPathRewritingIntegration tests the full integration of path rewriting
func TestPathRewritingIntegration(t *testing.T) {
	// Create a reverse proxy module
	module := NewModule()

	// Test combining all path rewriting features
	t.Run("AllFeaturesIntegration", func(t *testing.T) {
		config := &ReverseProxyConfig{
			PathRewriting: PathRewritingConfig{
				StripBasePath:   "/api/v1",
				BasePathRewrite: "/service",
				EndpointRewrites: map[string]EndpointRewriteRule{
					"users-rewrite": {
						Pattern:     "/service/users/*",
						Replacement: "/internal/users",
					},
				},
			},
		}

		// Test the full path rewriting flow
		result := module.applyPathRewriting("/api/v1/users/123", config)
		assert.Equal(t, "/internal/users/123", result,
			"Should apply strip, base rewrite, and endpoint rewrite in order")

		// Test with no base path match
		result = module.applyPathRewriting("/api/v2/users/123", config)
		assert.Equal(t, "/service/api/v2/users/123", result,
			"Should apply base rewrite when strip doesn't match and no endpoint rewrite matches")

		// Test with no endpoint rewrite match
		result = module.applyPathRewriting("/api/v1/orders/123", config)
		assert.Equal(t, "/service/orders/123", result,
			"Should apply strip and base rewrite when endpoint rewrite doesn't match")
	})
}
