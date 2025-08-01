package reverseproxy

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/CrisisTextLine/modular"
)

// TestDryRunWithStaticRouteConfiguration tests the exact scenario described in the issue:
// When a route is statically configured with dry run enabled, the user should be able to:
// 1. Hit the endpoint and get the result from the "legacy" backend
// 2. Compare and log the result of both "v2" and "legacy" backends
func TestDryRunWithStaticRouteConfiguration(t *testing.T) {
	// Create mock backends that return different responses
	legacyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Backend", "legacy")
		w.WriteHeader(http.StatusOK)
		response := map[string]interface{}{
			"backend":  "legacy",
			"endpoint": "/api/some/endpoint",
			"version":  "1.0",
			"data":     "legacy data",
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Errorf("Failed to encode response: %v", err)
		}
	}))
	defer legacyServer.Close()

	v2Server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Backend", "v2")
		w.WriteHeader(http.StatusOK)
		response := map[string]interface{}{
			"backend":  "v2",
			"endpoint": "/api/some/endpoint",
			"version":  "2.0",
			"data":     "v2 data",
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Errorf("Failed to encode response: %v", err)
		}
	}))
	defer v2Server.Close()

	// Create the configuration exactly as described in the issue
	config := &ReverseProxyConfig{
		BackendServices: map[string]string{
			"legacy": legacyServer.URL,
			"v2":     v2Server.URL,
		},
		Routes: map[string]string{
			"/api/some/endpoint": "v2", // Primary route goes to v2
		},
		RouteConfigs: map[string]RouteConfig{
			"/api/some/endpoint": {
				FeatureFlagID:      "v2-endpoint",
				AlternativeBackend: "legacy",
				DryRun:             true,
				DryRunBackend:      "v2",
			},
		},
		DryRun: DryRunConfig{
			Enabled:         true,
			LogResponses:    true,
			MaxResponseSize: 1048576,
		},
		// Feature flags configuration with the flag disabled
		FeatureFlags: FeatureFlagsConfig{
			Enabled: true,
			Flags: map[string]bool{
				"v2-endpoint": false, // Feature flag is disabled, should use alternative backend
			},
		},
	}

	// Create mock application and services
	app := NewMockTenantApplication()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	// Create mock router
	mockRouter := &testRouter{routes: make(map[string]http.HandlerFunc)}

	// Register tenant service for proper configuration management
	tenantService := modular.NewStandardTenantService(logger)
	if err := app.RegisterService("tenantService", tenantService); err != nil {
		t.Fatalf("Failed to register tenant service: %v", err)
	}

	// Register the configuration
	app.RegisterConfigSection("reverseproxy", modular.NewStdConfigProvider(config))

	// Create feature flag evaluator
	featureFlagEvaluator, err := NewFileBasedFeatureFlagEvaluator(app, logger)
	if err != nil {
		t.Fatalf("Failed to create feature flag evaluator: %v", err)
	}

	// Create and configure the reverse proxy module
	module := NewModule()
	
	// Register config
	if err := module.RegisterConfig(app); err != nil {
		t.Fatalf("Failed to register config: %v", err)
	}

	// Initialize with services
	services := map[string]any{
		"router":               mockRouter,
		"featureFlagEvaluator": featureFlagEvaluator,
	}

	constructedModule, err := module.Constructor()(app, services)
	if err != nil {
		t.Fatalf("Failed to construct module: %v", err)
	}

	reverseProxyModule := constructedModule.(*ReverseProxyModule)

	// Initialize the module
	if err := reverseProxyModule.Init(app); err != nil {
		t.Fatalf("Failed to initialize module: %v", err)
	}

	// Start the module
	if err := reverseProxyModule.Start(context.Background()); err != nil {
		t.Fatalf("Failed to start module: %v", err)
	}
	defer func() {
		if err := reverseProxyModule.Stop(context.Background()); err != nil {
			t.Errorf("Failed to stop module: %v", err)
		}
	}()

	// Create a test request to the configured endpoint
	req := httptest.NewRequest("GET", "/api/some/endpoint", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Request-ID", "test-dry-run-123")

	// Create a response recorder
	recorder := httptest.NewRecorder()

	// Get the handler for our specific route or catch-all
	t.Logf("Available routes: %v", mockRouter.routes)
	var handler http.HandlerFunc
	if h, exists := mockRouter.routes["/api/some/endpoint"]; exists {
		handler = h
	} else if h, exists := mockRouter.routes["/*"]; exists {
		handler = h
	} else {
		// Debug: list all registered routes
		for pattern := range mockRouter.routes {
			t.Logf("Registered route: %s", pattern)
		}
		t.Fatal("No handler found for the test route")
	}

	// Execute the handler
	handler.ServeHTTP(recorder, req)

	// Verify the response
	resp := recorder.Result()
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Errorf("Failed to close response body: %v", err)
		}
	}()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	t.Logf("Response status: %d", resp.StatusCode)
	t.Logf("Response body: %s", string(body))
	t.Logf("Response headers: %v", resp.Header)

	// Expected behavior based on the issue description:
	// 1. Since the feature flag "v2-endpoint" is disabled, it should use the alternative backend ("legacy")
	// 2. Since dry_run is true, it should also call the dry_run_backend ("v2") for comparison
	// 3. The response returned should be from the "legacy" backend

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Parse the response to verify it came from the legacy backend
	var responseData map[string]interface{}
	if err := json.Unmarshal(body, &responseData); err != nil {
		t.Fatalf("Failed to parse response JSON: %v", err)
	}

	backend, ok := responseData["backend"].(string)
	if !ok {
		t.Error("Response does not contain backend field")
	} else if backend != "legacy" {
		t.Errorf("Expected response from legacy backend, got response from %s backend", backend)
	}

	// Check that we got the legacy data
	data, ok := responseData["data"].(string)
	if !ok {
		t.Error("Response does not contain data field")
	} else if data != "legacy data" {
		t.Errorf("Expected legacy data, got: %s", data)
	}

	// Note: In the current implementation, we would expect dry run comparison to happen
	// but since we're using a test recorder, we can't easily verify the logging.
	// The important thing is that the correct backend response is returned.
}

// TestDryRunWithFeatureFlagEnabled tests what happens when the feature flag is enabled
func TestDryRunWithFeatureFlagEnabled(t *testing.T) {
	// Create mock backends
	legacyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`{"backend":"legacy","data":"legacy data"}`)); err != nil {
			t.Errorf("Failed to write response: %v", err)
		}
	}))
	defer legacyServer.Close()

	v2Server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`{"backend":"v2","data":"v2 data"}`)); err != nil {
			t.Errorf("Failed to write response: %v", err)
		}
	}))
	defer v2Server.Close()

	config := &ReverseProxyConfig{
		BackendServices: map[string]string{
			"legacy": legacyServer.URL,
			"v2":     v2Server.URL,
		},
		Routes: map[string]string{
			"/api/some/endpoint": "v2",
		},
		RouteConfigs: map[string]RouteConfig{
			"/api/some/endpoint": {
				FeatureFlagID:      "v2-endpoint",
				AlternativeBackend: "legacy",
				DryRun:             true,
				DryRunBackend:      "legacy", // Compare against legacy when using v2
			},
		},
		DryRun: DryRunConfig{
			Enabled:         true,
			LogResponses:    true,
			MaxResponseSize: 1048576,
		},
		FeatureFlags: FeatureFlagsConfig{
			Enabled: true,
			Flags: map[string]bool{
				"v2-endpoint": true, // Feature flag is enabled, should use primary backend (v2)
			},
		},
	}

	// Create mock application and services
	app := NewMockTenantApplication()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	// Create mock router
	mockRouter := &testRouter{routes: make(map[string]http.HandlerFunc)}

	// Register tenant service for proper configuration management
	tenantService := modular.NewStandardTenantService(logger)
	if err := app.RegisterService("tenantService", tenantService); err != nil {
		t.Fatalf("Failed to register tenant service: %v", err)
	}

	// Register the configuration
	app.RegisterConfigSection("reverseproxy", modular.NewStdConfigProvider(config))

	// Create feature flag evaluator
	featureFlagEvaluator, err := NewFileBasedFeatureFlagEvaluator(app, logger)
	if err != nil {
		t.Fatalf("Failed to create feature flag evaluator: %v", err)
	}

	module := NewModule()
	
	// Register config
	if err := module.RegisterConfig(app); err != nil {
		t.Fatalf("Failed to register config: %v", err)
	}

	// Initialize with services
	services := map[string]any{
		"router":               mockRouter,
		"featureFlagEvaluator": featureFlagEvaluator,
	}

	constructedModule, err := module.Constructor()(app, services)
	if err != nil {
		t.Fatalf("Failed to construct module: %v", err)
	}

	reverseProxyModule := constructedModule.(*ReverseProxyModule)

	if err := reverseProxyModule.Init(app); err != nil {
		t.Fatalf("Failed to initialize module: %v", err)
	}

	if err := reverseProxyModule.Start(context.Background()); err != nil {
		t.Fatalf("Failed to start module: %v", err)
	}
	defer func() {
		if err := reverseProxyModule.Stop(context.Background()); err != nil {
			t.Errorf("Failed to stop module: %v", err)
		}
	}()

	req := httptest.NewRequest("GET", "/api/some/endpoint", nil)
	recorder := httptest.NewRecorder()

	// Get the handler for our specific route or catch-all
	var handler http.HandlerFunc
	if h, exists := mockRouter.routes["/api/some/endpoint"]; exists {
		handler = h
	} else if h, exists := mockRouter.routes["/*"]; exists {
		handler = h
	} else {
		t.Fatal("No handler found for the test route")
	}

	// Execute the handler
	handler.ServeHTTP(recorder, req)

	resp := recorder.Result()
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Errorf("Failed to close response body: %v", err)
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	t.Logf("Response status: %d", resp.StatusCode)
	t.Logf("Response body: %s", string(body))

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// When feature flag is enabled, should get response from v2 backend
	if !strings.Contains(string(body), `"backend":"v2"`) {
		t.Errorf("Expected response from v2 backend when feature flag is enabled, got: %s", string(body))
	}
}

// TestDryRunDirectHandler tests the DryRunHandler in isolation
func TestDryRunDirectHandler(t *testing.T) {
	legacyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`{"backend":"legacy","message":"test"}`)); err != nil {
			t.Errorf("Failed to write response: %v", err)
		}
	}))
	defer legacyServer.Close()

	v2Server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`{"backend":"v2","message":"test"}`)); err != nil {
			t.Errorf("Failed to write response: %v", err)
		}
	}))
	defer v2Server.Close()

	config := DryRunConfig{
		Enabled:                true,
		LogResponses:           true,
		MaxResponseSize:        1048576,
		DefaultResponseBackend: "secondary", // Return secondary response (legacy)
	}

	handler := NewDryRunHandler(config, "X-Tenant-ID", NewMockLogger())
	req := httptest.NewRequest("GET", "/api/some/endpoint", nil)

	ctx := context.Background()
	result, err := handler.ProcessDryRun(ctx, req, v2Server.URL, legacyServer.URL)

	if err != nil {
		t.Fatalf("ProcessDryRun failed: %v", err)
	}

	if result == nil {
		t.Fatal("Dry run result is nil")
	}

	// Verify both backends were called
	if result.PrimaryResponse.StatusCode != http.StatusOK {
		t.Errorf("Expected primary response status 200, got %d", result.PrimaryResponse.StatusCode)
	}

	if result.SecondaryResponse.StatusCode != http.StatusOK {
		t.Errorf("Expected secondary response status 200, got %d", result.SecondaryResponse.StatusCode)
	}

	// Verify that secondary response is returned as configured
	if result.ReturnedResponse != "secondary" {
		t.Errorf("Expected returned response to be 'secondary', got %s", result.ReturnedResponse)
	}

	returnedResp := result.GetReturnedResponse()
	if !strings.Contains(returnedResp.Body, `"backend":"legacy"`) {
		t.Errorf("Expected returned response to be from legacy backend, got: %s", returnedResp.Body)
	}

	// Verify comparison detected differences
	if result.Comparison.BodyMatch {
		t.Error("Expected body content to differ between backends")
	}

	if len(result.Comparison.Differences) == 0 {
		t.Error("Expected differences to be reported")
	}

	t.Logf("Dry run completed successfully with %d differences", len(result.Comparison.Differences))
}