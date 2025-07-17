package reverseproxy

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/CrisisTextLine/modular"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBasePathRewriting tests the base path rewriting functionality
func TestBasePathRewriting(t *testing.T) {
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

	// Test Case 1: Strip base path
	t.Run("StripBasePath", func(t *testing.T) {
		// Reset received path
		receivedPath = ""

		// Configure the module to strip /api/v1 from all requests
		module.config = &ReverseProxyConfig{
			BackendServices: map[string]string{
				"api": backendServer.URL,
			},
			DefaultBackend: "api",
			TenantIDHeader: "X-Tenant-ID",
			PathRewriting: PathRewritingConfig{
				StripBasePath: "/api/v1",
			},
		}

		// Create the reverse proxy
		backendURL, err := url.Parse(backendServer.URL)
		require.NoError(t, err)
		proxy := module.createReverseProxy(backendURL)

		// Create a request with the base path that should be stripped
		req := httptest.NewRequest("GET", "http://client.example.com/api/v1/users/123", nil)
		req.Host = "client.example.com"

		// Process the request through the proxy
		w := httptest.NewRecorder()
		proxy.ServeHTTP(w, req)

		// Verify the response
		resp := w.Result()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// The backend should receive the path with /api/v1 stripped
		assert.Equal(t, "/users/123", receivedPath,
			"Backend should receive path with base path stripped")
	})

	// Test Case 2: Base path rewrite
	t.Run("BasePathRewrite", func(t *testing.T) {
		// Reset received path
		receivedPath = ""

		// Configure the module to rewrite base path
		module.config = &ReverseProxyConfig{
			BackendServices: map[string]string{
				"api": backendServer.URL,
			},
			DefaultBackend: "api",
			TenantIDHeader: "X-Tenant-ID",
			PathRewriting: PathRewritingConfig{
				BasePathRewrite: "/internal/api",
			},
		}

		// Create the reverse proxy
		backendURL, err := url.Parse(backendServer.URL)
		require.NoError(t, err)
		proxy := module.createReverseProxy(backendURL)

		// Create a request
		req := httptest.NewRequest("GET", "http://client.example.com/users/123", nil)
		req.Host = "client.example.com"

		// Process the request through the proxy
		w := httptest.NewRecorder()
		proxy.ServeHTTP(w, req)

		// Verify the response
		resp := w.Result()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// The backend should receive the path with base path prepended
		assert.Equal(t, "/internal/api/users/123", receivedPath,
			"Backend should receive path with base path prepended")
	})

	// Test Case 3: Strip base path AND rewrite
	t.Run("StripAndRewrite", func(t *testing.T) {
		// Reset received path
		receivedPath = ""

		// Configure the module to strip /api/v1 and rewrite to /internal/api
		module.config = &ReverseProxyConfig{
			BackendServices: map[string]string{
				"api": backendServer.URL,
			},
			DefaultBackend: "api",
			TenantIDHeader: "X-Tenant-ID",
			PathRewriting: PathRewritingConfig{
				StripBasePath:   "/api/v1",
				BasePathRewrite: "/internal/api",
			},
		}

		// Create the reverse proxy
		backendURL, err := url.Parse(backendServer.URL)
		require.NoError(t, err)
		proxy := module.createReverseProxy(backendURL)

		// Create a request with the base path that should be stripped and rewritten
		req := httptest.NewRequest("GET", "http://client.example.com/api/v1/users/123", nil)
		req.Host = "client.example.com"

		// Process the request through the proxy
		w := httptest.NewRecorder()
		proxy.ServeHTTP(w, req)

		// Verify the response
		resp := w.Result()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// The backend should receive the path with /api/v1 stripped and /internal/api prepended
		assert.Equal(t, "/internal/api/users/123", receivedPath,
			"Backend should receive path with base path stripped and rewritten")
	})
}

// TestEndpointPathRewriting tests the per-endpoint path rewriting functionality
func TestEndpointPathRewriting(t *testing.T) {
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

	// Test Case 1: Exact pattern match
	t.Run("ExactPatternMatch", func(t *testing.T) {
		// Reset received path
		receivedPath = ""

		// Configure the module with endpoint-specific rewriting
		module.config = &ReverseProxyConfig{
			BackendServices: map[string]string{
				"api": backendServer.URL,
			},
			DefaultBackend: "api",
			TenantIDHeader: "X-Tenant-ID",
			PathRewriting: PathRewritingConfig{
				EndpointRewrites: map[string]EndpointRewriteRule{
					"users-exact": {
						Pattern:     "/api/users",
						Replacement: "/internal/users",
					},
				},
			},
		}

		// Create the reverse proxy
		backendURL, err := url.Parse(backendServer.URL)
		require.NoError(t, err)
		proxy := module.createReverseProxy(backendURL)

		// Create a request that exactly matches the pattern
		req := httptest.NewRequest("GET", "http://client.example.com/api/users", nil)
		req.Host = "client.example.com"

		// Process the request through the proxy
		w := httptest.NewRecorder()
		proxy.ServeHTTP(w, req)

		// Verify the response
		resp := w.Result()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// The backend should receive the rewritten path
		assert.Equal(t, "/internal/users", receivedPath,
			"Backend should receive rewritten path for exact match")
	})

	// Test Case 2: Wildcard pattern match
	t.Run("WildcardPatternMatch", func(t *testing.T) {
		// Reset received path
		receivedPath = ""

		// Configure the module with wildcard endpoint rewriting
		module.config = &ReverseProxyConfig{
			BackendServices: map[string]string{
				"api": backendServer.URL,
			},
			DefaultBackend: "api",
			TenantIDHeader: "X-Tenant-ID",
			PathRewriting: PathRewritingConfig{
				EndpointRewrites: map[string]EndpointRewriteRule{
					"users-wildcard": {
						Pattern:     "/api/users/*",
						Replacement: "/internal/users",
					},
				},
			},
		}

		// Create the reverse proxy
		backendURL, err := url.Parse(backendServer.URL)
		require.NoError(t, err)
		proxy := module.createReverseProxy(backendURL)

		// Create a request that matches the wildcard pattern
		req := httptest.NewRequest("GET", "http://client.example.com/api/users/123/profile", nil)
		req.Host = "client.example.com"

		// Process the request through the proxy
		w := httptest.NewRecorder()
		proxy.ServeHTTP(w, req)

		// Verify the response
		resp := w.Result()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// The backend should receive the rewritten path with suffix preserved
		assert.Equal(t, "/internal/users/123/profile", receivedPath,
			"Backend should receive rewritten path with suffix preserved")
	})

	// Test Case 3: Multiple rules - first match wins
	t.Run("MultipleRulesFirstWins", func(t *testing.T) {
		// Reset received path
		receivedPath = ""

		// Configure the module with multiple endpoint rewriting rules
		module.config = &ReverseProxyConfig{
			BackendServices: map[string]string{
				"api": backendServer.URL,
			},
			DefaultBackend: "api",
			TenantIDHeader: "X-Tenant-ID",
			PathRewriting: PathRewritingConfig{
				EndpointRewrites: map[string]EndpointRewriteRule{
					"users-specific": {
						Pattern:     "/api/users/123",
						Replacement: "/special/user",
					},
					"users-general": {
						Pattern:     "/api/users/*",
						Replacement: "/internal/users",
					},
				},
			},
		}

		// Create the reverse proxy
		backendURL, err := url.Parse(backendServer.URL)
		require.NoError(t, err)
		proxy := module.createReverseProxy(backendURL)

		// Create a request that matches both patterns
		req := httptest.NewRequest("GET", "http://client.example.com/api/users/123", nil)
		req.Host = "client.example.com"

		// Process the request through the proxy
		w := httptest.NewRecorder()
		proxy.ServeHTTP(w, req)

		// Verify the response
		resp := w.Result()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// The backend should receive the first matching rule's replacement
		// Note: The exact behavior depends on map iteration order, but one of them should win
		assert.True(t, receivedPath == "/special/user" || receivedPath == "/internal/users/123",
			"Backend should receive path from one of the matching rules")
	})

	// Test Case 4: No pattern match - path unchanged
	t.Run("NoPatternMatch", func(t *testing.T) {
		// Reset received path
		receivedPath = ""

		// Configure the module with endpoint rewriting that won't match
		module.config = &ReverseProxyConfig{
			BackendServices: map[string]string{
				"api": backendServer.URL,
			},
			DefaultBackend: "api",
			TenantIDHeader: "X-Tenant-ID",
			PathRewriting: PathRewritingConfig{
				EndpointRewrites: map[string]EndpointRewriteRule{
					"users-only": {
						Pattern:     "/api/users/*",
						Replacement: "/internal/users",
					},
				},
			},
		}

		// Create the reverse proxy
		backendURL, err := url.Parse(backendServer.URL)
		require.NoError(t, err)
		proxy := module.createReverseProxy(backendURL)

		// Create a request that doesn't match any pattern
		req := httptest.NewRequest("GET", "http://client.example.com/api/orders/456", nil)
		req.Host = "client.example.com"

		// Process the request through the proxy
		w := httptest.NewRecorder()
		proxy.ServeHTTP(w, req)

		// Verify the response
		resp := w.Result()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// The backend should receive the original path unchanged
		assert.Equal(t, "/api/orders/456", receivedPath,
			"Backend should receive original path when no pattern matches")
	})
}

// TestCombinedPathRewriting tests combining base path rewriting with endpoint rewriting
func TestCombinedPathRewriting(t *testing.T) {
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

	// Test Case 1: Base path stripping + endpoint rewriting
	t.Run("BaseStripAndEndpointRewrite", func(t *testing.T) {
		// Reset received path
		receivedPath = ""

		// Configure the module with both base path stripping and endpoint rewriting
		module.config = &ReverseProxyConfig{
			BackendServices: map[string]string{
				"api": backendServer.URL,
			},
			DefaultBackend: "api",
			TenantIDHeader: "X-Tenant-ID",
			PathRewriting: PathRewritingConfig{
				StripBasePath: "/api/v1",
				EndpointRewrites: map[string]EndpointRewriteRule{
					"users-rewrite": {
						Pattern:     "/users/*",
						Replacement: "/internal/users",
					},
				},
			},
		}

		// Create the reverse proxy
		backendURL, err := url.Parse(backendServer.URL)
		require.NoError(t, err)
		proxy := module.createReverseProxy(backendURL)

		// Create a request
		req := httptest.NewRequest("GET", "http://client.example.com/api/v1/users/123", nil)
		req.Host = "client.example.com"

		// Process the request through the proxy
		w := httptest.NewRecorder()
		proxy.ServeHTTP(w, req)

		// Verify the response
		resp := w.Result()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// The backend should receive: /api/v1/users/123 -> /users/123 (strip) -> /internal/users/123 (rewrite)
		assert.Equal(t, "/internal/users/123", receivedPath,
			"Backend should receive path with base path stripped and then endpoint rewritten")
	})

	// Test Case 2: Base path rewrite + endpoint rewriting
	t.Run("BaseRewriteAndEndpointRewrite", func(t *testing.T) {
		// Reset received path
		receivedPath = ""

		// Configure the module with both base path rewriting and endpoint rewriting
		module.config = &ReverseProxyConfig{
			BackendServices: map[string]string{
				"api": backendServer.URL,
			},
			DefaultBackend: "api",
			TenantIDHeader: "X-Tenant-ID",
			PathRewriting: PathRewritingConfig{
				BasePathRewrite: "/service",
				EndpointRewrites: map[string]EndpointRewriteRule{
					"users-rewrite": {
						Pattern:     "/service/users/*",
						Replacement: "/internal/users",
					},
				},
			},
		}

		// Create the reverse proxy
		backendURL, err := url.Parse(backendServer.URL)
		require.NoError(t, err)
		proxy := module.createReverseProxy(backendURL)

		// Create a request
		req := httptest.NewRequest("GET", "http://client.example.com/users/123", nil)
		req.Host = "client.example.com"

		// Process the request through the proxy
		w := httptest.NewRecorder()
		proxy.ServeHTTP(w, req)

		// Verify the response
		resp := w.Result()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// The backend should receive: /users/123 -> /service/users/123 (base rewrite) -> /internal/users/123 (endpoint rewrite)
		assert.Equal(t, "/internal/users/123", receivedPath,
			"Backend should receive path with base path rewritten and then endpoint rewritten")
	})
}

// TestTenantPathRewriting tests that tenant-specific path rewriting works correctly
func TestTenantPathRewriting(t *testing.T) {
	// Track what path the backend receives
	var receivedPath string
	var receivedTenantHeader string

	// Create mock backend servers
	globalBackendServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		receivedTenantHeader = r.Header.Get("X-Tenant-ID")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		response := map[string]string{
			"message": "global backend response",
			"path":    r.URL.Path,
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer globalBackendServer.Close()

	tenantBackendServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		receivedTenantHeader = r.Header.Get("X-Tenant-ID")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		response := map[string]string{
			"message": "tenant backend response",
			"path":    r.URL.Path,
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer tenantBackendServer.Close()

	// Create a reverse proxy module
	module := NewModule()

	// Set up global configuration
	module.config = &ReverseProxyConfig{
		BackendServices: map[string]string{
			"api": globalBackendServer.URL,
		},
		DefaultBackend: "api",
		TenantIDHeader: "X-Tenant-ID",
		PathRewriting: PathRewritingConfig{
			StripBasePath: "/api/v1",
			EndpointRewrites: map[string]EndpointRewriteRule{
				"global-users": {
					Pattern:     "/users/*",
					Replacement: "/global/users",
				},
			},
		},
	}

	// Set up tenant-specific configuration
	tenantID := modular.TenantID("tenant-123")
	module.tenants = make(map[modular.TenantID]*ReverseProxyConfig)
	module.tenants[tenantID] = &ReverseProxyConfig{
		BackendServices: map[string]string{
			"api": tenantBackendServer.URL,
		},
		DefaultBackend: "api",
		TenantIDHeader: "X-Tenant-ID",
		PathRewriting: PathRewritingConfig{
			StripBasePath: "/api/v2", // Different base path for tenant
			EndpointRewrites: map[string]EndpointRewriteRule{
				"tenant-users": {
					Pattern:     "/users/*",
					Replacement: "/tenant/users",
				},
			},
		},
	}

	// Test Case 1: Request without tenant header uses global configuration
	t.Run("GlobalPathRewriting", func(t *testing.T) {
		// Reset received values
		receivedPath = ""
		receivedTenantHeader = ""

		// Create the reverse proxy for global backend
		globalURL, err := url.Parse(globalBackendServer.URL)
		require.NoError(t, err)
		proxy := module.createReverseProxy(globalURL)

		// Create a request without tenant header
		req := httptest.NewRequest("GET", "http://client.example.com/api/v1/users/123", nil)
		req.Host = "client.example.com"

		// Process the request through the proxy
		w := httptest.NewRecorder()
		proxy.ServeHTTP(w, req)

		// Verify the response
		resp := w.Result()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// The backend should receive path processed by global configuration
		assert.Equal(t, "/global/users/123", receivedPath,
			"Global backend should receive path processed by global configuration")
		assert.Equal(t, "", receivedTenantHeader,
			"Global backend should not receive tenant header")
	})

	// Test Case 2: Request with tenant header uses tenant-specific configuration
	t.Run("TenantPathRewriting", func(t *testing.T) {
		// Reset received values
		receivedPath = ""
		receivedTenantHeader = ""

		// Create the reverse proxy for tenant backend
		tenantURL, err := url.Parse(tenantBackendServer.URL)
		require.NoError(t, err)
		proxy := module.createReverseProxy(tenantURL)

		// Create a request with tenant header
		req := httptest.NewRequest("GET", "http://client.example.com/api/v2/users/456", nil)
		req.Host = "client.example.com"
		req.Header.Set("X-Tenant-ID", string(tenantID))

		// Process the request through the proxy
		w := httptest.NewRecorder()
		proxy.ServeHTTP(w, req)

		// Verify the response
		resp := w.Result()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// The backend should receive path processed by tenant-specific configuration
		assert.Equal(t, "/tenant/users/456", receivedPath,
			"Tenant backend should receive path processed by tenant configuration")
		assert.Equal(t, string(tenantID), receivedTenantHeader,
			"Tenant backend should receive tenant header")
	})

	// Test Case 3: Request with tenant header but different base path
	t.Run("TenantPathRewritingDifferentBasePath", func(t *testing.T) {
		// Reset received values
		receivedPath = ""
		receivedTenantHeader = ""

		// Create the reverse proxy for tenant backend
		tenantURL, err := url.Parse(tenantBackendServer.URL)
		require.NoError(t, err)
		proxy := module.createReverseProxy(tenantURL)

		// Create a request with tenant header but using global base path
		req := httptest.NewRequest("GET", "http://client.example.com/api/v1/users/789", nil)
		req.Host = "client.example.com"
		req.Header.Set("X-Tenant-ID", string(tenantID))

		// Process the request through the proxy
		w := httptest.NewRecorder()
		proxy.ServeHTTP(w, req)

		// Verify the response
		resp := w.Result()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// The backend should receive path processed by tenant config, but since the base path doesn't match
		// the tenant's StripBasePath (/api/v2), it should just keep the original path since no endpoint rewrite matches
		assert.Equal(t, "/api/v1/users/789", receivedPath,
			"Tenant backend should receive original path when base path doesn't match and no endpoint rewrite applies")
	})
}
