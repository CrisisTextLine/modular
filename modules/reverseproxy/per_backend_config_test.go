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

// TestPerBackendPathRewriting tests path rewriting configuration per backend
func TestPerBackendPathRewriting(t *testing.T) {
	// Track what path each backend receives
	var apiReceivedPath, userReceivedPath string

	// Create mock backend servers
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiReceivedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		response := map[string]string{"service": "api", "path": r.URL.Path}
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer apiServer.Close()

	userServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userReceivedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		response := map[string]string{"service": "user", "path": r.URL.Path}
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer userServer.Close()

	// Create a reverse proxy module
	module := NewModule()

	// Configure per-backend path rewriting
	module.config = &ReverseProxyConfig{
		BackendServices: map[string]string{
			"api":  apiServer.URL,
			"user": userServer.URL,
		},
		BackendConfigs: map[string]BackendServiceConfig{
			"api": {
				URL: apiServer.URL,
				PathRewriting: PathRewritingConfig{
					StripBasePath:   "/api/v1",
					BasePathRewrite: "/internal/api",
				},
			},
			"user": {
				URL: userServer.URL,
				PathRewriting: PathRewritingConfig{
					StripBasePath:   "/user/v1",
					BasePathRewrite: "/internal/user",
				},
			},
		},
		TenantIDHeader: "X-Tenant-ID",
	}

	t.Run("API Backend Path Rewriting", func(t *testing.T) {
		// Reset received path
		apiReceivedPath = ""

		// Create the reverse proxy for API backend
		apiURL, err := url.Parse(apiServer.URL)
		require.NoError(t, err)
		proxy := module.createReverseProxyForBackend(apiURL, "api", "")

		// Create a request that should be rewritten
		req := httptest.NewRequest("GET", "http://client.example.com/api/v1/products/123", nil)
		req.Host = "client.example.com"

		// Process the request through the proxy
		w := httptest.NewRecorder()
		proxy.ServeHTTP(w, req)

		// Verify the response
		resp := w.Result()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// The API backend should receive the path rewritten as /internal/api/products/123
		assert.Equal(t, "/internal/api/products/123", apiReceivedPath,
			"API backend should receive path with /api/v1 stripped and /internal/api prepended")
	})

	t.Run("User Backend Path Rewriting", func(t *testing.T) {
		// Reset received path
		userReceivedPath = ""

		// Create the reverse proxy for User backend
		userURL, err := url.Parse(userServer.URL)
		require.NoError(t, err)
		proxy := module.createReverseProxyForBackend(userURL, "user", "")

		// Create a request that should be rewritten
		req := httptest.NewRequest("GET", "http://client.example.com/user/v1/profile/456", nil)
		req.Host = "client.example.com"

		// Process the request through the proxy
		w := httptest.NewRecorder()
		proxy.ServeHTTP(w, req)

		// Verify the response
		resp := w.Result()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// The User backend should receive the path rewritten as /internal/user/profile/456
		assert.Equal(t, "/internal/user/profile/456", userReceivedPath,
			"User backend should receive path with /user/v1 stripped and /internal/user prepended")
	})
}

// TestPerBackendHostnameHandling tests hostname handling configuration per backend
func TestPerBackendHostnameHandling(t *testing.T) {
	// Track what hostname each backend receives
	var apiReceivedHost, userReceivedHost string

	// Create mock backend servers
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiReceivedHost = r.Host
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		response := map[string]string{"service": "api", "host": r.Host}
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer apiServer.Close()

	userServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userReceivedHost = r.Host
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		response := map[string]string{"service": "user", "host": r.Host}
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer userServer.Close()

	// Create a reverse proxy module
	module := NewModule()

	// Configure per-backend hostname handling
	module.config = &ReverseProxyConfig{
		BackendServices: map[string]string{
			"api":  apiServer.URL,
			"user": userServer.URL,
		},
		BackendConfigs: map[string]BackendServiceConfig{
			"api": {
				URL: apiServer.URL,
				HeaderRewriting: HeaderRewritingConfig{
					HostnameHandling: HostnamePreserveOriginal, // Default behavior
				},
			},
			"user": {
				URL: userServer.URL,
				HeaderRewriting: HeaderRewritingConfig{
					HostnameHandling: HostnameUseBackend, // Use backend hostname
				},
			},
		},
		TenantIDHeader: "X-Tenant-ID",
	}

	t.Run("API Backend Preserves Original Hostname", func(t *testing.T) {
		// Reset received host
		apiReceivedHost = ""

		// Create the reverse proxy for API backend
		apiURL, err := url.Parse(apiServer.URL)
		require.NoError(t, err)
		proxy := module.createReverseProxyForBackend(apiURL, "api", "")

		// Create a request with original hostname
		req := httptest.NewRequest("GET", "http://client.example.com/api/products", nil)
		req.Host = "client.example.com"

		// Process the request through the proxy
		w := httptest.NewRecorder()
		proxy.ServeHTTP(w, req)

		// Verify the response
		resp := w.Result()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// The API backend should receive the original hostname
		assert.Equal(t, "client.example.com", apiReceivedHost,
			"API backend should receive original client hostname")
	})

	t.Run("User Backend Uses Backend Hostname", func(t *testing.T) {
		// Reset received host
		userReceivedHost = ""

		// Create the reverse proxy for User backend
		userURL, err := url.Parse(userServer.URL)
		require.NoError(t, err)
		proxy := module.createReverseProxyForBackend(userURL, "user", "")

		// Create a request with original hostname
		req := httptest.NewRequest("GET", "http://client.example.com/user/profile", nil)
		req.Host = "client.example.com"

		// Process the request through the proxy
		w := httptest.NewRecorder()
		proxy.ServeHTTP(w, req)

		// Verify the response
		resp := w.Result()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// The User backend should receive the backend hostname
		expectedHost := userURL.Host
		assert.Equal(t, expectedHost, userReceivedHost,
			"User backend should receive backend hostname")
	})
}

// TestPerBackendCustomHostname tests custom hostname configuration per backend
func TestPerBackendCustomHostname(t *testing.T) {
	// Track what hostname the backend receives
	var receivedHost string

	// Create mock backend server
	backendServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHost = r.Host
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		response := map[string]string{"service": "api", "host": r.Host}
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer backendServer.Close()

	// Create a reverse proxy module
	module := NewModule()

	// Configure custom hostname handling
	module.config = &ReverseProxyConfig{
		BackendServices: map[string]string{
			"api": backendServer.URL,
		},
		BackendConfigs: map[string]BackendServiceConfig{
			"api": {
				URL: backendServer.URL,
				HeaderRewriting: HeaderRewritingConfig{
					HostnameHandling: HostnameUseCustom,
					CustomHostname:   "custom.internal.com",
				},
			},
		},
		TenantIDHeader: "X-Tenant-ID",
	}

	t.Run("Backend Uses Custom Hostname", func(t *testing.T) {
		// Reset received host
		receivedHost = ""

		// Create the reverse proxy for API backend
		apiURL, err := url.Parse(backendServer.URL)
		require.NoError(t, err)
		proxy := module.createReverseProxyForBackend(apiURL, "api", "")

		// Create a request with original hostname
		req := httptest.NewRequest("GET", "http://client.example.com/api/products", nil)
		req.Host = "client.example.com"

		// Process the request through the proxy
		w := httptest.NewRecorder()
		proxy.ServeHTTP(w, req)

		// Verify the response
		resp := w.Result()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// The backend should receive the custom hostname
		assert.Equal(t, "custom.internal.com", receivedHost,
			"Backend should receive custom hostname")
	})
}

// TestPerBackendHeaderRewriting tests header rewriting configuration per backend
func TestPerBackendHeaderRewriting(t *testing.T) {
	// Track what headers the backend receives
	var receivedHeaders map[string]string

	// Create mock backend server
	backendServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = make(map[string]string)
		for name, values := range r.Header {
			if len(values) > 0 {
				receivedHeaders[name] = values[0]
			}
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		response := map[string]interface{}{
			"service": "api",
			"headers": receivedHeaders,
		}
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer backendServer.Close()

	// Create a reverse proxy module
	module := NewModule()

	// Configure header rewriting
	module.config = &ReverseProxyConfig{
		BackendServices: map[string]string{
			"api": backendServer.URL,
		},
		BackendConfigs: map[string]BackendServiceConfig{
			"api": {
				URL: backendServer.URL,
				HeaderRewriting: HeaderRewritingConfig{
					SetHeaders: map[string]string{
						"X-API-Key":     "secret-key",
						"X-Custom-Auth": "bearer-token",
					},
					RemoveHeaders: []string{"X-Client-Version"},
				},
			},
		},
		TenantIDHeader: "X-Tenant-ID",
	}

	t.Run("Backend Receives Modified Headers", func(t *testing.T) {
		// Reset received headers
		receivedHeaders = nil

		// Create the reverse proxy for API backend
		apiURL, err := url.Parse(backendServer.URL)
		require.NoError(t, err)
		proxy := module.createReverseProxyForBackend(apiURL, "api", "")

		// Create a request with original headers
		req := httptest.NewRequest("GET", "http://client.example.com/api/products", nil)
		req.Host = "client.example.com"
		req.Header.Set("X-Client-Version", "1.0.0")
		req.Header.Set("X-Original-Header", "original-value")

		// Process the request through the proxy
		w := httptest.NewRecorder()
		proxy.ServeHTTP(w, req)

		// Verify the response
		resp := w.Result()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// The backend should receive the modified headers
		assert.Equal(t, "secret-key", receivedHeaders["X-Api-Key"],
			"Backend should receive set X-API-Key header")
		assert.Equal(t, "bearer-token", receivedHeaders["X-Custom-Auth"],
			"Backend should receive set X-Custom-Auth header")
		assert.Equal(t, "original-value", receivedHeaders["X-Original-Header"],
			"Backend should receive original header that wasn't modified")
		assert.Empty(t, receivedHeaders["X-Client-Version"],
			"Backend should not receive removed X-Client-Version header")
	})
}

// TestPerEndpointConfiguration tests endpoint-specific configuration
func TestPerEndpointConfiguration(t *testing.T) {
	// Track what the backend receives
	var receivedPath, receivedHost string

	// Create mock backend server
	backendServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		receivedHost = r.Host
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		response := map[string]string{
			"service": "api",
			"path":    r.URL.Path,
			"host":    r.Host,
		}
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer backendServer.Close()

	// Create a reverse proxy module
	module := NewModule()

	// Configure endpoint-specific configuration
	module.config = &ReverseProxyConfig{
		BackendServices: map[string]string{
			"api": backendServer.URL,
		},
		BackendConfigs: map[string]BackendServiceConfig{
			"api": {
				URL: backendServer.URL,
				PathRewriting: PathRewritingConfig{
					StripBasePath: "/api/v1",
				},
				HeaderRewriting: HeaderRewritingConfig{
					HostnameHandling: HostnamePreserveOriginal,
				},
				Endpoints: map[string]EndpointConfig{
					"users": {
						Pattern: "/users/*",
						PathRewriting: PathRewritingConfig{
							BasePathRewrite: "/internal/users",
						},
						HeaderRewriting: HeaderRewritingConfig{
							HostnameHandling: HostnameUseCustom,
							CustomHostname:   "users.internal.com",
						},
					},
				},
			},
		},
		TenantIDHeader: "X-Tenant-ID",
	}

	t.Run("Users Endpoint Uses Specific Configuration", func(t *testing.T) {
		// Reset received values
		receivedPath = ""
		receivedHost = ""

		// Create the reverse proxy for API backend with users endpoint
		apiURL, err := url.Parse(backendServer.URL)
		require.NoError(t, err)
		proxy := module.createReverseProxyForBackend(apiURL, "api", "users")

		// Create a request to users endpoint
		req := httptest.NewRequest("GET", "http://client.example.com/api/v1/users/123", nil)
		req.Host = "client.example.com"

		// Process the request through the proxy
		w := httptest.NewRecorder()
		proxy.ServeHTTP(w, req)

		// Verify the response
		resp := w.Result()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// The backend should receive endpoint-specific configuration
		assert.Equal(t, "/internal/users/users/123", receivedPath,
			"Backend should receive endpoint-specific path rewriting")
		assert.Equal(t, "users.internal.com", receivedHost,
			"Backend should receive endpoint-specific hostname")
	})
}
