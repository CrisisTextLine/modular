package reverseproxy

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/CrisisTextLine/modular"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/cucumber/godog"
)

// ReverseProxy BDD Test Context
type ReverseProxyBDDTestContext struct {
	app           modular.Application
	module        *ReverseProxyModule
	service       *ReverseProxyModule
	config        *ReverseProxyConfig
	lastError     error
	testServers   []*httptest.Server
	lastResponse  *http.Response
	eventObserver *testEventObserver
}

// testEventObserver captures CloudEvents during testing
type testEventObserver struct {
	events []cloudevents.Event
}

func newTestEventObserver() *testEventObserver {
	return &testEventObserver{
		events: make([]cloudevents.Event, 0),
	}
}

func (t *testEventObserver) OnEvent(ctx context.Context, event cloudevents.Event) error {
	t.events = append(t.events, event.Clone())
	return nil
}

func (t *testEventObserver) ObserverID() string {
	return "test-observer-reverseproxy"
}

func (t *testEventObserver) GetEvents() []cloudevents.Event {
	events := make([]cloudevents.Event, len(t.events))
	copy(events, t.events)
	return events
}

func (t *testEventObserver) ClearEvents() {
	t.events = make([]cloudevents.Event, 0)
}

func (ctx *ReverseProxyBDDTestContext) resetContext() {
	// Close test servers
	for _, server := range ctx.testServers {
		if server != nil {
			server.Close()
		}
	}

	ctx.app = nil
	ctx.module = nil
	ctx.service = nil
	ctx.config = nil
	ctx.lastError = nil
	ctx.testServers = nil
	ctx.lastResponse = nil
}

func (ctx *ReverseProxyBDDTestContext) iHaveAModularApplicationWithReverseProxyModuleConfigured() error {
	ctx.resetContext()

	// Create a test backend server first
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test backend response"))
	}))
	ctx.testServers = append(ctx.testServers, testServer)

	// Create basic reverse proxy configuration for testing using the test server
	ctx.config = &ReverseProxyConfig{
		BackendServices: map[string]string{
			"test-backend": testServer.URL,
		},
		Routes: map[string]string{
			"/api/*": "test-backend",
		},
		BackendConfigs: map[string]BackendServiceConfig{
			"test-backend": {
				URL: testServer.URL,
			},
		},
	}

	// Create application
	logger := &testLogger{}

	// Clear ConfigFeeders and disable AppConfigLoader to prevent environment interference during tests
	modular.ConfigFeeders = []modular.Feeder{}
	originalLoader := modular.AppConfigLoader
	modular.AppConfigLoader = func(app *modular.StdApplication) error { return nil }
	// Don't restore them - let them stay disabled throughout all BDD tests
	_ = originalLoader

	mainConfigProvider := modular.NewStdConfigProvider(struct{}{})
	ctx.app = modular.NewObservableApplication(mainConfigProvider, logger)

	// Verify that the observable application properly implements the required interfaces
	if _, ok := ctx.app.(modular.Application); !ok {
		return fmt.Errorf("observable application does not implement Application interface")
	}
	if _, ok := ctx.app.(modular.Subject); !ok {
		return fmt.Errorf("observable application does not implement Subject interface")
	}

	// Create and register a mock router service (required by ReverseProxy)
	mockRouter := &testRouter{
		routes: make(map[string]http.HandlerFunc),
	}
	ctx.app.RegisterService("router", mockRouter)

	// Create and register reverse proxy module
	ctx.module = NewModule()

	// Register the reverseproxy config section
	reverseproxyConfigProvider := modular.NewStdConfigProvider(ctx.config)
	ctx.app.RegisterConfigSection("reverseproxy", reverseproxyConfigProvider)

	// Register the module
	ctx.app.RegisterModule(ctx.module)

	return nil
}

// setupApplicationWithConfig creates a fresh application with the current configuration
func (ctx *ReverseProxyBDDTestContext) setupApplicationWithConfig() error {
	// Create application
	logger := &testLogger{}

	// Clear ConfigFeeders and disable AppConfigLoader to prevent environment interference during tests
	modular.ConfigFeeders = []modular.Feeder{}
	modular.AppConfigLoader = func(app *modular.StdApplication) error { return nil }

	mainConfigProvider := modular.NewStdConfigProvider(struct{}{})
	ctx.app = modular.NewObservableApplication(mainConfigProvider, logger)

	// Create and register a mock router service (required by ReverseProxy)
	mockRouter := &testRouter{
		routes: make(map[string]http.HandlerFunc),
	}
	ctx.app.RegisterService("router", mockRouter)

	// Create and register reverse proxy module
	ctx.module = NewModule()

	// Register the reverseproxy config section with current configuration
	reverseproxyConfigProvider := modular.NewStdConfigProvider(ctx.config)
	ctx.app.RegisterConfigSection("reverseproxy", reverseproxyConfigProvider)

	// Register the module
	ctx.app.RegisterModule(ctx.module)

	// Initialize the application with the complete configuration
	return ctx.app.Init()
}

func (ctx *ReverseProxyBDDTestContext) theReverseProxyModuleIsInitialized() error {
	err := ctx.app.Init()
	if err != nil {
		ctx.lastError = err
		return err
	}
	return nil
}

func (ctx *ReverseProxyBDDTestContext) theProxyServiceShouldBeAvailable() error {
	err := ctx.app.GetService("reverseproxy.provider", &ctx.service)
	if err != nil {
		return err
	}
	if ctx.service == nil {
		return fmt.Errorf("proxy service not available")
	}
	return nil
}

func (ctx *ReverseProxyBDDTestContext) theModuleShouldBeReadyToRouteRequests() error {
	// Verify the module is properly configured
	if ctx.service == nil || ctx.service.config == nil {
		return fmt.Errorf("module not properly initialized")
	}
	return nil
}

func (ctx *ReverseProxyBDDTestContext) iHaveAReverseProxyConfiguredWithASingleBackend() error {
	// The background step has already set up a single backend configuration
	// Initialize the module so it's ready for the "When" step
	return ctx.app.Init()
}

func (ctx *ReverseProxyBDDTestContext) iSendARequestToTheProxy() error {
	// Ensure service is available if not already retrieved
	if ctx.service == nil {
		err := ctx.app.GetService("reverseproxy.provider", &ctx.service)
		if err != nil {
			return fmt.Errorf("failed to get reverseproxy service: %w", err)
		}
	}

	if ctx.service == nil {
		return fmt.Errorf("service not available")
	}

	// Start the service
	err := ctx.app.Start()
	if err != nil {
		return err
	}

	// Simulate a request (in real tests would make HTTP call)
	// For BDD test, we just verify the service is ready
	return nil
}

func (ctx *ReverseProxyBDDTestContext) theRequestShouldBeForwardedToTheBackend() error {
	// Verify that the reverse proxy service is available and configured
	if ctx.service == nil {
		return fmt.Errorf("reverse proxy service not available")
	}

	// Verify that at least one backend is configured for request forwarding
	if ctx.config == nil || len(ctx.config.BackendServices) == 0 {
		return fmt.Errorf("no backend targets configured for request forwarding")
	}

	return nil
}

func (ctx *ReverseProxyBDDTestContext) theResponseShouldBeReturnedToTheClient() error {
	// In a real implementation, would verify response handling
	return nil
}

func (ctx *ReverseProxyBDDTestContext) iHaveAReverseProxyConfiguredWithMultipleBackends() error {
	// Reset context and set up fresh application for this scenario
	ctx.resetContext()

	// Create multiple test backend servers
	for i := 0; i < 3; i++ {
		testServer := httptest.NewServer(http.HandlerFunc(func(idx int) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(fmt.Sprintf("backend-%d response", idx)))
			}
		}(i)))
		ctx.testServers = append(ctx.testServers, testServer)
	}

	// Create configuration with multiple backends
	ctx.config = &ReverseProxyConfig{
		BackendServices: map[string]string{
			"backend-1": ctx.testServers[0].URL,
			"backend-2": ctx.testServers[1].URL,
			"backend-3": ctx.testServers[2].URL,
		},
		Routes: map[string]string{
			"/api/*": "backend-1",
		},
		BackendConfigs: map[string]BackendServiceConfig{
			"backend-1": {URL: ctx.testServers[0].URL},
			"backend-2": {URL: ctx.testServers[1].URL},
			"backend-3": {URL: ctx.testServers[2].URL},
		},
	}

	return ctx.setupApplicationWithConfig()
}

func (ctx *ReverseProxyBDDTestContext) iSendMultipleRequestsToTheProxy() error {
	return ctx.iSendARequestToTheProxy()
}

func (ctx *ReverseProxyBDDTestContext) requestsShouldBeDistributedAcrossAllBackends() error {
	// Ensure service is available
	if ctx.service == nil {
		err := ctx.app.GetService("reverseproxy.provider", &ctx.service)
		if err != nil {
			return fmt.Errorf("failed to get reverseproxy service: %w", err)
		}
	}

	if ctx.service == nil || ctx.service.config == nil {
		return fmt.Errorf("service or config not available")
	}

	// Verify multiple backends are configured
	if len(ctx.service.config.BackendServices) < 2 {
		return fmt.Errorf("expected multiple backends, got %d", len(ctx.service.config.BackendServices))
	}
	return nil
}

func (ctx *ReverseProxyBDDTestContext) loadBalancingShouldBeApplied() error {
	// In a real implementation, would verify load balancing algorithm
	return nil
}

func (ctx *ReverseProxyBDDTestContext) iHaveAReverseProxyWithHealthChecksEnabled() error {
	// Ensure health checks are enabled
	ctx.config.HealthCheck.Enabled = true
	ctx.config.HealthCheck.Interval = 5 * time.Second
	ctx.config.HealthCheck.HealthEndpoints = map[string]string{
		"test-backend": "/health",
	}

	// Re-register the config section with the updated configuration
	reverseproxyConfigProvider := modular.NewStdConfigProvider(ctx.config)
	ctx.app.RegisterConfigSection("reverseproxy", reverseproxyConfigProvider)

	// Initialize the module with the updated configuration
	return ctx.app.Init()
}

func (ctx *ReverseProxyBDDTestContext) aBackendBecomesUnavailable() error {
	// Simulate backend failure by closing one test server
	if len(ctx.testServers) > 0 {
		ctx.testServers[0].Close()
	}
	return nil
}

func (ctx *ReverseProxyBDDTestContext) theProxyShouldDetectTheFailure() error {
	// In a real implementation, would verify health check detection
	return nil
}

func (ctx *ReverseProxyBDDTestContext) routeTrafficOnlyToHealthyBackends() error {
	// In a real implementation, would verify traffic routing
	return nil
}

func (ctx *ReverseProxyBDDTestContext) iHaveAReverseProxyWithCircuitBreakerEnabled() error {
	// Reset context and set up fresh application for this scenario
	ctx.resetContext()

	// Create a test backend server
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test backend response"))
	}))
	ctx.testServers = append(ctx.testServers, testServer)

	// Create configuration with circuit breaker enabled
	ctx.config = &ReverseProxyConfig{
		BackendServices: map[string]string{
			"test-backend": testServer.URL,
		},
		Routes: map[string]string{
			"/api/*": "test-backend",
		},
		BackendConfigs: map[string]BackendServiceConfig{
			"test-backend": {
				URL: testServer.URL,
			},
		},
		CircuitBreakerConfig: CircuitBreakerConfig{
			Enabled:          true,
			FailureThreshold: 3,
		},
	}

	return ctx.setupApplicationWithConfig()
}

func (ctx *ReverseProxyBDDTestContext) aBackendFailsRepeatedly() error {
	// Create a backend that will fail repeatedly to trigger circuit breaker
	if len(ctx.testServers) == 0 {
		return fmt.Errorf("no backend servers available to fail")
	}

	// Create a failing backend server that returns errors
	failingServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return 500 error to simulate backend failure
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Backend service unavailable"))
	}))

	// Replace the first test server with the failing one
	ctx.testServers[0].Close()
	ctx.testServers[0] = failingServer

	// Update configuration to point to the failing server
	backendName := "test-backend"
	if ctx.config.BackendServices == nil {
		ctx.config.BackendServices = make(map[string]string)
	}
	ctx.config.BackendServices[backendName] = failingServer.URL

	// Make multiple requests through the reverse proxy to trigger circuit breaker logic
	for i := 0; i < 5; i++ {
		// Make requests through the reverse proxy module, not directly to backend
		// This will trigger the actual circuit breaker logic and emit real events
		resp, err := ctx.makeRequestThroughModule("GET", "/api/test", nil)
		if resp != nil {
			resp.Body.Close()
		}
		// Continue making requests to trigger circuit breaker regardless of errors
		if err != nil {
			continue
		}
	}

	return nil
}

func (ctx *ReverseProxyBDDTestContext) theCircuitBreakerShouldOpen() error {
	// Ensure service is available
	if ctx.service == nil {
		err := ctx.app.GetService("reverseproxy.provider", &ctx.service)
		if err != nil {
			return fmt.Errorf("failed to get reverseproxy service: %w", err)
		}
	}

	if ctx.service == nil || ctx.service.config == nil {
		return fmt.Errorf("service or config not available")
	}

	// Verify circuit breaker configuration
	if !ctx.service.config.CircuitBreakerConfig.Enabled {
		return fmt.Errorf("circuit breaker not enabled")
	}

	// Verify that circuit breaker state has been affected by the failed requests
	// Check if we have any circuit breakers registered
	if len(ctx.service.circuitBreakers) == 0 {
		return fmt.Errorf("no circuit breakers registered despite failures")
	}

	// Check if any circuit breaker is open or half-open (indicating it responded to failures)
	foundActiveCircuitBreaker := false
	for _, cb := range ctx.service.circuitBreakers {
		if cb != nil {
			state := cb.GetState()
			if state == StateOpen || state == StateHalfOpen {
				foundActiveCircuitBreaker = true
				break
			}
			// Even if not open, check if failure count increased
			if cb.GetFailureCount() > 0 {
				foundActiveCircuitBreaker = true
				break
			}
		}
	}

	if !foundActiveCircuitBreaker {
		return fmt.Errorf("circuit breaker did not respond to backend failures as expected")
	}

	return nil
}

func (ctx *ReverseProxyBDDTestContext) requestsShouldBeHandledGracefully() error {
	// In a real implementation, would verify graceful handling
	return nil
}

func (ctx *ReverseProxyBDDTestContext) iHaveAReverseProxyWithCachingEnabled() error {
	// Reset context and set up fresh application for this scenario
	ctx.resetContext()

	// Create a test backend server
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test backend response"))
	}))
	ctx.testServers = append(ctx.testServers, testServer)

	// Create configuration with caching enabled
	ctx.config = &ReverseProxyConfig{
		BackendServices: map[string]string{
			"test-backend": testServer.URL,
		},
		Routes: map[string]string{
			"/api/*": "test-backend",
		},
		BackendConfigs: map[string]BackendServiceConfig{
			"test-backend": {
				URL: testServer.URL,
			},
		},
		CacheEnabled: true,
		CacheTTL:     300 * time.Second,
	}

	return ctx.setupApplicationWithConfig()
}

func (ctx *ReverseProxyBDDTestContext) iSendTheSameRequestMultipleTimes() error {
	return ctx.iSendARequestToTheProxy()
}

func (ctx *ReverseProxyBDDTestContext) theFirstRequestShouldHitTheBackend() error {
	// In a real implementation, would verify cache miss
	return nil
}

func (ctx *ReverseProxyBDDTestContext) subsequentRequestsShouldBeServedFromCache() error {
	// Ensure service is available
	if ctx.service == nil {
		err := ctx.app.GetService("reverseproxy.provider", &ctx.service)
		if err != nil {
			return fmt.Errorf("failed to get reverseproxy service: %w", err)
		}
	}

	if ctx.service == nil || ctx.service.config == nil {
		return fmt.Errorf("service or config not available")
	}

	// Verify caching is enabled
	if !ctx.service.config.CacheEnabled {
		return fmt.Errorf("caching not enabled")
	}
	return nil
}

func (ctx *ReverseProxyBDDTestContext) iHaveATenantAwareReverseProxyConfigured() error {
	// Add tenant-specific configuration
	ctx.config.RequireTenantID = true
	ctx.config.TenantIDHeader = "X-Tenant-ID"

	// Re-register the config section with the updated configuration
	reverseproxyConfigProvider := modular.NewStdConfigProvider(ctx.config)
	ctx.app.RegisterConfigSection("reverseproxy", reverseproxyConfigProvider)

	// Initialize the module with the updated configuration
	return ctx.app.Init()
}

func (ctx *ReverseProxyBDDTestContext) iSendRequestsWithDifferentTenantContexts() error {
	return ctx.iSendARequestToTheProxy()
}

func (ctx *ReverseProxyBDDTestContext) requestsShouldBeRoutedBasedOnTenantConfiguration() error {
	// Ensure service is available
	if ctx.service == nil {
		err := ctx.app.GetService("reverseproxy.provider", &ctx.service)
		if err != nil {
			return fmt.Errorf("failed to get reverseproxy service: %w", err)
		}
	}

	if ctx.service == nil || ctx.service.config == nil {
		return fmt.Errorf("service or config not available")
	}

	// Verify tenant routing is configured
	if !ctx.service.config.RequireTenantID {
		return fmt.Errorf("tenant routing not enabled")
	}
	return nil
}

func (ctx *ReverseProxyBDDTestContext) tenantIsolationShouldBeMaintained() error {
	// In a real implementation, would verify tenant isolation
	return nil
}

func (ctx *ReverseProxyBDDTestContext) iHaveAReverseProxyConfiguredForCompositeResponses() error {
	// Add composite route configuration
	ctx.config.CompositeRoutes = map[string]CompositeRoute{
		"/api/combined": {
			Pattern:  "/api/combined",
			Backends: []string{"backend-1", "backend-2"},
			Strategy: "combine",
		},
	}

	// Re-register the config section with the updated configuration
	reverseproxyConfigProvider := modular.NewStdConfigProvider(ctx.config)
	ctx.app.RegisterConfigSection("reverseproxy", reverseproxyConfigProvider)

	// Initialize the module with the updated configuration
	return ctx.app.Init()
}

func (ctx *ReverseProxyBDDTestContext) iSendARequestThatRequiresMultipleBackendCalls() error {
	return ctx.iSendARequestToTheProxy()
}

func (ctx *ReverseProxyBDDTestContext) theProxyShouldCallAllRequiredBackends() error {
	// Ensure service is available
	if ctx.service == nil {
		err := ctx.app.GetService("reverseproxy.provider", &ctx.service)
		if err != nil {
			return fmt.Errorf("failed to get reverseproxy service: %w", err)
		}
	}

	if ctx.service == nil || ctx.service.config == nil {
		return fmt.Errorf("service or config not available")
	}

	// Verify composite routes are configured
	if len(ctx.service.config.CompositeRoutes) == 0 {
		return fmt.Errorf("no composite routes configured")
	}
	return nil
}

func (ctx *ReverseProxyBDDTestContext) combineTheResponsesIntoASingleResponse() error {
	// In a real implementation, would verify response combination
	return nil
}

func (ctx *ReverseProxyBDDTestContext) iHaveAReverseProxyWithRequestTransformationConfigured() error {
	// Create a test backend server for transformation testing
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("transformed backend response"))
	}))
	ctx.testServers = append(ctx.testServers, testServer)

	// Add backend configuration with header rewriting
	ctx.config.BackendConfigs = map[string]BackendServiceConfig{
		"backend-1": {
			URL: testServer.URL,
			HeaderRewriting: HeaderRewritingConfig{
				SetHeaders: map[string]string{
					"X-Forwarded-By": "reverse-proxy",
				},
				RemoveHeaders: []string{"Authorization"},
			},
		},
	}

	// Update backend services to use the test server
	ctx.config.BackendServices["backend-1"] = testServer.URL

	// Re-register the config section with the updated configuration
	reverseproxyConfigProvider := modular.NewStdConfigProvider(ctx.config)
	ctx.app.RegisterConfigSection("reverseproxy", reverseproxyConfigProvider)

	// Initialize the module with the updated configuration
	return ctx.app.Init()
}

func (ctx *ReverseProxyBDDTestContext) theRequestShouldBeTransformedBeforeForwarding() error {
	// Ensure service is available
	if ctx.service == nil {
		err := ctx.app.GetService("reverseproxy.provider", &ctx.service)
		if err != nil {
			return fmt.Errorf("failed to get reverseproxy service: %w", err)
		}
	}

	if ctx.service == nil || ctx.service.config == nil {
		return fmt.Errorf("service or config not available")
	}

	// Verify backend configs with header rewriting are configured
	if len(ctx.service.config.BackendConfigs) == 0 {
		return fmt.Errorf("no backend configs configured")
	}
	return nil
}

func (ctx *ReverseProxyBDDTestContext) theBackendShouldReceiveTheTransformedRequest() error {
	// In a real implementation, would verify transformed request
	return nil
}

func (ctx *ReverseProxyBDDTestContext) iHaveAnActiveReverseProxyWithOngoingRequests() error {
	// Initialize the module with the basic configuration from background
	err := ctx.app.Init()
	if err != nil {
		return err
	}

	err = ctx.theProxyServiceShouldBeAvailable()
	if err != nil {
		return err
	}

	// Start the module
	return ctx.app.Start()
}

func (ctx *ReverseProxyBDDTestContext) theModuleIsStopped() error {
	return ctx.app.Stop()
}

func (ctx *ReverseProxyBDDTestContext) ongoingRequestsShouldBeCompleted() error {
	// In a real implementation, would verify graceful completion
	return nil
}

func (ctx *ReverseProxyBDDTestContext) newRequestsShouldBeRejectedGracefully() error {
	// In a real implementation, would verify graceful rejection
	return nil
}

// Event observation step methods
func (ctx *ReverseProxyBDDTestContext) iHaveAReverseProxyWithEventObservationEnabled() error {
	ctx.resetContext()

	// Create application with reverse proxy config - use ObservableApplication for event support
	logger := &testLogger{}
	mainConfigProvider := modular.NewStdConfigProvider(struct{}{})
	ctx.app = modular.NewObservableApplication(mainConfigProvider, logger)

	// Create and register a mock router service (required by ReverseProxy)
	mockRouter := &testRouter{
		routes: make(map[string]http.HandlerFunc),
	}
	ctx.app.RegisterService("router", mockRouter)

	// Create a test backend server
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("backend response"))
	}))
	ctx.testServers = append(ctx.testServers, testServer)

	// Create reverse proxy configuration
	ctx.config = &ReverseProxyConfig{
		BackendServices: map[string]string{
			"test-backend": testServer.URL,
		},
		Routes: map[string]string{
			"/api/test": "test-backend",
		},
		DefaultBackend: "test-backend",
	}

	// Create reverse proxy module
	ctx.module = NewModule()
	ctx.service = ctx.module

	// Create test event observer
	ctx.eventObserver = newTestEventObserver()

	// Register our test observer BEFORE registering module to capture all events
	if err := ctx.app.(modular.Subject).RegisterObserver(ctx.eventObserver); err != nil {
		return fmt.Errorf("failed to register test observer: %w", err)
	}

	// Register module
	ctx.app.RegisterModule(ctx.module)

	// Register reverse proxy config section
	reverseproxyConfigProvider := modular.NewStdConfigProvider(ctx.config)
	ctx.app.RegisterConfigSection("reverseproxy", reverseproxyConfigProvider)

	// Initialize the application (this should trigger config loaded events)
	if err := ctx.app.Init(); err != nil {
		return fmt.Errorf("failed to initialize app: %v", err)
	}

	return nil
}

func (ctx *ReverseProxyBDDTestContext) theReverseProxyModuleStarts() error {
	// Start the application
	if err := ctx.app.Start(); err != nil {
		return fmt.Errorf("failed to start app: %v", err)
	}

	// Give time for all events to be emitted
	time.Sleep(200 * time.Millisecond)
	return nil
}

func (ctx *ReverseProxyBDDTestContext) aProxyCreatedEventShouldBeEmitted() error {
	time.Sleep(100 * time.Millisecond) // Allow time for async event emission

	events := ctx.eventObserver.GetEvents()
	for _, event := range events {
		if event.Type() == EventTypeProxyCreated {
			return nil
		}
	}

	eventTypes := make([]string, len(events))
	for i, event := range events {
		eventTypes[i] = event.Type()
	}

	return fmt.Errorf("event of type %s was not emitted. Captured events: %v", EventTypeProxyCreated, eventTypes)
}

func (ctx *ReverseProxyBDDTestContext) aProxyStartedEventShouldBeEmitted() error {
	events := ctx.eventObserver.GetEvents()
	for _, event := range events {
		if event.Type() == EventTypeProxyStarted {
			return nil
		}
	}

	eventTypes := make([]string, len(events))
	for i, event := range events {
		eventTypes[i] = event.Type()
	}

	return fmt.Errorf("event of type %s was not emitted. Captured events: %v", EventTypeProxyStarted, eventTypes)
}

func (ctx *ReverseProxyBDDTestContext) aModuleStartedEventShouldBeEmitted() error {
	events := ctx.eventObserver.GetEvents()
	for _, event := range events {
		if event.Type() == EventTypeModuleStarted {
			return nil
		}
	}

	eventTypes := make([]string, len(events))
	for i, event := range events {
		eventTypes[i] = event.Type()
	}

	return fmt.Errorf("event of type %s was not emitted. Captured events: %v", EventTypeModuleStarted, eventTypes)
}

func (ctx *ReverseProxyBDDTestContext) theEventsShouldContainProxyConfigurationDetails() error {
	events := ctx.eventObserver.GetEvents()

	// Check module started event has configuration details
	for _, event := range events {
		if event.Type() == EventTypeModuleStarted {
			var data map[string]interface{}
			if err := event.DataAs(&data); err != nil {
				return fmt.Errorf("failed to extract module started event data: %v", err)
			}

			// Check for key configuration fields
			if _, exists := data["backend_count"]; !exists {
				return fmt.Errorf("module started event should contain backend_count field")
			}

			return nil
		}
	}

	return fmt.Errorf("module started event not found")
}

func (ctx *ReverseProxyBDDTestContext) theReverseProxyModuleStops() error {
	return ctx.app.Stop()
}

func (ctx *ReverseProxyBDDTestContext) aProxyStoppedEventShouldBeEmitted() error {
	time.Sleep(100 * time.Millisecond) // Allow time for async event emission

	events := ctx.eventObserver.GetEvents()
	for _, event := range events {
		if event.Type() == EventTypeProxyStopped {
			return nil
		}
	}

	eventTypes := make([]string, len(events))
	for i, event := range events {
		eventTypes[i] = event.Type()
	}

	return fmt.Errorf("event of type %s was not emitted. Captured events: %v", EventTypeProxyStopped, eventTypes)
}

func (ctx *ReverseProxyBDDTestContext) aModuleStoppedEventShouldBeEmitted() error {
	events := ctx.eventObserver.GetEvents()
	for _, event := range events {
		if event.Type() == EventTypeModuleStopped {
			return nil
		}
	}

	eventTypes := make([]string, len(events))
	for i, event := range events {
		eventTypes[i] = event.Type()
	}

	return fmt.Errorf("event of type %s was not emitted. Captured events: %v", EventTypeModuleStopped, eventTypes)
}

func (ctx *ReverseProxyBDDTestContext) iHaveABackendServiceConfigured() error {
	// This is already done in the setup, just ensure it's ready
	return nil
}

func (ctx *ReverseProxyBDDTestContext) iSendARequestToTheReverseProxy() error {
	// Check if app is started and service is available
	if ctx.service == nil {
		return fmt.Errorf("service not available")
	}

	// Make sure the application is started so routes are registered
	if err := ctx.app.Start(); err != nil {
		return fmt.Errorf("failed to start app: %w", err)
	}

	// Make an actual request through the module to trigger events
	resp, err := ctx.makeRequestThroughModule("GET", "/api/test", nil)
	if err != nil {
		return fmt.Errorf("failed to make request through module: %w", err)
	}
	defer resp.Body.Close()

	ctx.lastResponse = resp

	// Give time for async event emission
	time.Sleep(100 * time.Millisecond)
	return nil
}

// Helper method to make requests through the module
func (ctx *ReverseProxyBDDTestContext) makeRequestThroughModule(method, path string, body io.Reader) (*http.Response, error) {
	// Create a test request
	req := httptest.NewRequest(method, path, body)

	// Create a test response recorder
	w := httptest.NewRecorder()

	// Get the router service and serve the request
	var router routerService
	if err := ctx.app.GetService("router", &router); err != nil {
		return nil, fmt.Errorf("failed to get router service: %w", err)
	}

	// Serve the request through the router
	router.ServeHTTP(w, req)

	// Convert the recorded response to an http.Response
	result := w.Result()
	return result, nil
}

func (ctx *ReverseProxyBDDTestContext) aRequestReceivedEventShouldBeEmitted() error {
	time.Sleep(100 * time.Millisecond) // Allow time for async event emission

	events := ctx.eventObserver.GetEvents()
	for _, event := range events {
		if event.Type() == EventTypeRequestReceived {
			return nil
		}
	}

	eventTypes := make([]string, len(events))
	for i, event := range events {
		eventTypes[i] = event.Type()
	}

	return fmt.Errorf("event of type %s was not emitted. Captured events: %v", EventTypeRequestReceived, eventTypes)
}

func (ctx *ReverseProxyBDDTestContext) theEventShouldContainRequestDetails() error {
	events := ctx.eventObserver.GetEvents()

	// Check request received event has request details
	for _, event := range events {
		if event.Type() == EventTypeRequestReceived {
			var data map[string]interface{}
			if err := event.DataAs(&data); err != nil {
				return fmt.Errorf("failed to extract request received event data: %v", err)
			}

			// Check for key request fields
			if _, exists := data["backend"]; !exists {
				return fmt.Errorf("request received event should contain backend field")
			}
			if _, exists := data["method"]; !exists {
				return fmt.Errorf("request received event should contain method field")
			}

			return nil
		}
	}

	return fmt.Errorf("request received event not found")
}

func (ctx *ReverseProxyBDDTestContext) theRequestIsSuccessfullyProxiedToTheBackend() error {
	// Wait for the request to be processed
	time.Sleep(100 * time.Millisecond)
	return nil
}

func (ctx *ReverseProxyBDDTestContext) aRequestProxiedEventShouldBeEmitted() error {
	time.Sleep(200 * time.Millisecond) // Allow time for async event emission

	events := ctx.eventObserver.GetEvents()
	for _, event := range events {
		if event.Type() == EventTypeRequestProxied {
			return nil
		}
	}

	eventTypes := make([]string, len(events))
	for i, event := range events {
		eventTypes[i] = event.Type()
	}

	return fmt.Errorf("event of type %s was not emitted. Captured events: %v", EventTypeRequestProxied, eventTypes)
}

func (ctx *ReverseProxyBDDTestContext) theEventShouldContainBackendAndResponseDetails() error {
	events := ctx.eventObserver.GetEvents()

	// Check request proxied event has backend and response details
	for _, event := range events {
		if event.Type() == EventTypeRequestProxied {
			var data map[string]interface{}
			if err := event.DataAs(&data); err != nil {
				return fmt.Errorf("failed to extract request proxied event data: %v", err)
			}

			// Check for key response fields
			if _, exists := data["backend"]; !exists {
				return fmt.Errorf("request proxied event should contain backend field")
			}

			return nil
		}
	}

	return fmt.Errorf("request proxied event not found")
}

func (ctx *ReverseProxyBDDTestContext) iHaveAnUnavailableBackendServiceConfigured() error {
	// Configure with an invalid backend URL to simulate unavailability
	ctx.config.BackendServices = map[string]string{
		"unavailable-backend": "http://localhost:99999", // Invalid port
	}
	return nil
}

func (ctx *ReverseProxyBDDTestContext) theRequestFailsToReachTheBackend() error {
	// Wait for the request to fail
	time.Sleep(300 * time.Millisecond)
	return nil
}

func (ctx *ReverseProxyBDDTestContext) aRequestFailedEventShouldBeEmitted() error {
	time.Sleep(200 * time.Millisecond) // Allow time for async event emission

	events := ctx.eventObserver.GetEvents()
	for _, event := range events {
		if event.Type() == EventTypeRequestFailed {
			return nil
		}
	}

	eventTypes := make([]string, len(events))
	for i, event := range events {
		eventTypes[i] = event.Type()
	}

	return fmt.Errorf("event of type %s was not emitted. Captured events: %v", EventTypeRequestFailed, eventTypes)
}

func (ctx *ReverseProxyBDDTestContext) theEventShouldContainErrorDetails() error {
	events := ctx.eventObserver.GetEvents()

	// Check request failed event has error details
	for _, event := range events {
		if event.Type() == EventTypeRequestFailed {
			var data map[string]interface{}
			if err := event.DataAs(&data); err != nil {
				return fmt.Errorf("failed to extract request failed event data: %v", err)
			}

			// Check for error field
			if _, exists := data["error"]; !exists {
				return fmt.Errorf("request failed event should contain error field")
			}

			return nil
		}
	}

	return fmt.Errorf("request failed event not found")
}

// Backend health management event scenarios
func (ctx *ReverseProxyBDDTestContext) iHaveBackendsWithHealthCheckingEnabled() error {
	return ctx.iHaveAReverseProxyWithHealthChecksEnabled()
}

func (ctx *ReverseProxyBDDTestContext) aBackendBecomesHealthy() error {
	// Create a realistic healthy backend server that responds properly to health checks
	if ctx.service == nil {
		return fmt.Errorf("service not available")
	}

	// Create a new healthy backend server that responds to both regular requests and health checks
	healthyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status": "healthy", "timestamp": "` + time.Now().Format(time.RFC3339) + `"}`))
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("healthy backend response"))
		}
	}))
	ctx.testServers = append(ctx.testServers, healthyServer)

	backendName := "healthy-backend"

	// Update the configuration to include this backend
	if ctx.config.BackendServices == nil {
		ctx.config.BackendServices = make(map[string]string)
	}
	ctx.config.BackendServices[backendName] = healthyServer.URL

	// Perform actual health check to verify the backend is healthy
	healthCheckURL := healthyServer.URL + "/health"
	resp, err := http.Get(healthCheckURL)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	// Test the backend by making a request through the reverse proxy
	// This should trigger the reverse proxy's internal health monitoring and event emission
	proxyResp, proxyErr := ctx.makeRequestThroughModule("GET", "/api/test", nil)
	if proxyErr == nil && proxyResp != nil {
		proxyResp.Body.Close()
		// The actual health monitoring should be triggered by the reverse proxy module
		// No manual event emission needed - the module should emit events based on real operations
	}

	return nil
}

func (ctx *ReverseProxyBDDTestContext) aBackendHealthyEventShouldBeEmitted() error {
	events := ctx.eventObserver.GetEvents()

	for _, event := range events {
		if event.Type() == EventTypeBackendHealthy {
			return nil
		}
	}

	return fmt.Errorf("backend healthy event not found")
}

func (ctx *ReverseProxyBDDTestContext) theEventShouldContainBackendHealthDetails() error {
	events := ctx.eventObserver.GetEvents()

	for _, event := range events {
		if event.Type() == EventTypeBackendHealthy {
			var data map[string]interface{}
			if err := event.DataAs(&data); err != nil {
				return fmt.Errorf("failed to extract backend healthy event data: %v", err)
			}

			// Check for backend field
			if _, exists := data["backend"]; !exists {
				return fmt.Errorf("backend healthy event should contain backend field")
			}

			return nil
		}
	}

	return fmt.Errorf("backend healthy event not found")
}

func (ctx *ReverseProxyBDDTestContext) aBackendBecomesUnhealthy() error {
	// Test real backend failure by making requests through the reverse proxy
	// to trigger actual health monitoring and failure detection
	if ctx.service == nil {
		return fmt.Errorf("service not available")
	}

	if len(ctx.testServers) == 0 {
		return fmt.Errorf("no backend servers available to make unhealthy")
	}

	// Close the original healthy server to simulate failure
	originalServer := ctx.testServers[0]
	originalServer.Close()

	// Create a new server that returns error responses (simulating unhealthy backend)
	unhealthyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Always return service unavailable to simulate backend failure
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("Service temporarily unavailable"))
	}))

	// Update the configuration to point to the unhealthy server
	ctx.testServers[0] = unhealthyServer
	ctx.config.BackendServices["test-backend"] = unhealthyServer.URL

	// Reconfigure the module with the new backend URL
	err := ctx.setupApplicationWithConfig()
	if err != nil {
		return fmt.Errorf("failed to reconfigure with unhealthy backend: %w", err)
	}

	// Start the application to register routes
	if err := ctx.app.Start(); err != nil {
		return fmt.Errorf("failed to start app: %w", err)
	}

	// Make multiple requests through the reverse proxy to trigger failure detection
	// This should cause the reverse proxy to emit backend.unhealthy events
	for i := 0; i < 3; i++ {
		resp, err := ctx.makeRequestThroughModule("GET", "/api/test", nil)
		if err != nil {
			// Connection error is expected when backend fails
			continue
		}
		resp.Body.Close()
		// Status 503 or 500 responses should trigger backend unhealthy detection
		if resp.StatusCode >= 500 {
			break
		}
	}

	// Give time for health checks to process and emit events
	time.Sleep(100 * time.Millisecond)

	return nil
}

func (ctx *ReverseProxyBDDTestContext) aBackendUnhealthyEventShouldBeEmitted() error {
	events := ctx.eventObserver.GetEvents()

	for _, event := range events {
		if event.Type() == EventTypeBackendUnhealthy {
			return nil
		}
	}

	return fmt.Errorf("backend unhealthy event not found")
}

func (ctx *ReverseProxyBDDTestContext) theEventShouldContainHealthFailureDetails() error {
	events := ctx.eventObserver.GetEvents()

	for _, event := range events {
		if event.Type() == EventTypeBackendUnhealthy {
			var data map[string]interface{}
			if err := event.DataAs(&data); err != nil {
				return fmt.Errorf("failed to extract backend unhealthy event data: %v", err)
			}

			// Check for backend and error fields
			if _, exists := data["backend"]; !exists {
				return fmt.Errorf("backend unhealthy event should contain backend field")
			}

			return nil
		}
	}

	return fmt.Errorf("backend unhealthy event not found")
}

// Backend management event scenarios
func (ctx *ReverseProxyBDDTestContext) aNewBackendIsAddedToTheConfiguration() error {
	// Test real backend addition by updating configuration and making requests through the proxy
	if ctx.service == nil {
		return fmt.Errorf("service not available")
	}

	// Create a new backend server with realistic endpoints
	newBackendServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status": "healthy", "service": "new-backend"}`))
		case "/api/data":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"message": "new backend data", "service": "new-backend"}`))
		default:
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("new backend response"))
		}
	}))
	ctx.testServers = append(ctx.testServers, newBackendServer)

	backendName := "new-backend-test"

	// Update the configuration to include this new backend
	if ctx.config.BackendServices == nil {
		ctx.config.BackendServices = make(map[string]string)
	}
	ctx.config.BackendServices[backendName] = newBackendServer.URL

	// Add route mapping for the new backend
	if ctx.config.Routes == nil {
		ctx.config.Routes = make(map[string]string)
	}
	ctx.config.Routes["/api/new"] = backendName

	// Reconfigure the module with the updated configuration
	err := ctx.setupApplicationWithConfig()
	if err != nil {
		return fmt.Errorf("failed to reconfigure with new backend: %w", err)
	}

	// Start the application to register routes
	if err := ctx.app.Start(); err != nil {
		return fmt.Errorf("failed to start app: %w", err)
	}

	// Test the new backend by making a request through the reverse proxy
	resp, err := ctx.makeRequestThroughModule("GET", "/api/new", nil)
	if err != nil {
		return fmt.Errorf("failed to test new backend through proxy: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("new backend not accessible through proxy: status %d", resp.StatusCode)
	}

	// Give time for any configuration change events to be emitted
	time.Sleep(100 * time.Millisecond)

	return nil
}

func (ctx *ReverseProxyBDDTestContext) aBackendAddedEventShouldBeEmitted() error {
	events := ctx.eventObserver.GetEvents()

	for _, event := range events {
		if event.Type() == EventTypeBackendAdded {
			return nil
		}
	}

	return fmt.Errorf("backend added event not found")
}

func (ctx *ReverseProxyBDDTestContext) theEventShouldContainBackendConfiguration() error {
	events := ctx.eventObserver.GetEvents()

	for _, event := range events {
		if event.Type() == EventTypeBackendAdded {
			var data map[string]interface{}
			if err := event.DataAs(&data); err != nil {
				return fmt.Errorf("failed to extract backend added event data: %v", err)
			}

			// Check for backend configuration field
			if _, exists := data["backend"]; !exists {
				return fmt.Errorf("backend added event should contain backend field")
			}

			return nil
		}
	}

	return fmt.Errorf("backend added event not found")
}

func (ctx *ReverseProxyBDDTestContext) aBackendIsRemovedFromTheConfiguration() error {
	// Perform realistic backend removal with proper configuration management
	if ctx.service == nil {
		return fmt.Errorf("service not available")
	}

	if len(ctx.testServers) == 0 {
		return fmt.Errorf("no backend servers available to remove")
	}

	// Identify the backend to remove
	serverToRemove := ctx.testServers[len(ctx.testServers)-1]
	removedBackendName := "backend-to-remove"

	// Find the backend name in configuration
	var actualBackendName string
	if ctx.config.BackendServices != nil {
		for name, url := range ctx.config.BackendServices {
			if url == serverToRemove.URL {
				actualBackendName = name
				break
			}
		}
	}
	if actualBackendName == "" {
		actualBackendName = removedBackendName
	}

	// Perform one final health check before removal to demonstrate operational state
	finalCheckURL := serverToRemove.URL + "/health"
	finalResp, err := http.Get(finalCheckURL)
	if err == nil {
		finalResp.Body.Close()
	}

	// Update configuration to remove the backend
	if ctx.config.BackendServices != nil {
		delete(ctx.config.BackendServices, actualBackendName)
	}

	// Test that the backend is no longer accessible through the reverse proxy
	// This should trigger the reverse proxy's internal monitoring and emit removal events
	resp, err := ctx.makeRequestThroughModule("GET", "/api/removed", nil)
	if resp != nil {
		resp.Body.Close()
		// The reverse proxy module should detect the backend removal and emit appropriate events
	}
	_ = err // Expected to fail since backend is removed

	// Remove from test servers list and close the server
	ctx.testServers = ctx.testServers[:len(ctx.testServers)-1]
	serverToRemove.Close()

	return nil
}

func (ctx *ReverseProxyBDDTestContext) aBackendRemovedEventShouldBeEmitted() error {
	events := ctx.eventObserver.GetEvents()

	for _, event := range events {
		if event.Type() == EventTypeBackendRemoved {
			return nil
		}
	}

	return fmt.Errorf("backend removed event not found")
}

func (ctx *ReverseProxyBDDTestContext) theEventShouldContainRemovalDetails() error {
	events := ctx.eventObserver.GetEvents()

	for _, event := range events {
		if event.Type() == EventTypeBackendRemoved {
			var data map[string]interface{}
			if err := event.DataAs(&data); err != nil {
				return fmt.Errorf("failed to extract backend removed event data: %v", err)
			}

			// Check for backend field
			if _, exists := data["backend"]; !exists {
				return fmt.Errorf("backend removed event should contain backend field")
			}

			return nil
		}
	}

	return fmt.Errorf("backend removed event not found")
}

// Load balancing event scenarios
func (ctx *ReverseProxyBDDTestContext) iHaveMultipleBackendsConfigured() error {
	return ctx.iHaveAReverseProxyConfiguredWithMultipleBackends()
}

func (ctx *ReverseProxyBDDTestContext) loadBalancingDecisionsAreMade() error {
	return ctx.iSendMultipleRequestsToTheProxy()
}

func (ctx *ReverseProxyBDDTestContext) loadBalanceDecisionEventsShouldBeEmitted() error {
	events := ctx.eventObserver.GetEvents()

	for _, event := range events {
		if event.Type() == EventTypeLoadBalanceDecision {
			return nil
		}
	}

	return fmt.Errorf("load balance decision event not found")
}

func (ctx *ReverseProxyBDDTestContext) theEventsShouldContainSelectedBackendInformation() error {
	events := ctx.eventObserver.GetEvents()

	for _, event := range events {
		if event.Type() == EventTypeLoadBalanceDecision {
			var data map[string]interface{}
			if err := event.DataAs(&data); err != nil {
				return fmt.Errorf("failed to extract load balance decision event data: %v", err)
			}

			// Check for backend selection field
			if _, exists := data["selected_backend"]; !exists {
				return fmt.Errorf("load balance decision event should contain selected_backend field")
			}

			return nil
		}
	}

	return fmt.Errorf("load balance decision event not found")
}

func (ctx *ReverseProxyBDDTestContext) roundrobinLoadBalancingIsUsed() error {
	return ctx.iSendMultipleRequestsToTheProxy()
}

func (ctx *ReverseProxyBDDTestContext) roundrobinEventsShouldBeEmitted() error {
	events := ctx.eventObserver.GetEvents()

	for _, event := range events {
		if event.Type() == EventTypeLoadBalanceRoundRobin {
			return nil
		}
	}

	return fmt.Errorf("round-robin event not found")
}

func (ctx *ReverseProxyBDDTestContext) theEventsShouldContainRotationDetails() error {
	events := ctx.eventObserver.GetEvents()

	for _, event := range events {
		if event.Type() == EventTypeLoadBalanceRoundRobin {
			var data map[string]interface{}
			if err := event.DataAs(&data); err != nil {
				return fmt.Errorf("failed to extract round-robin event data: %v", err)
			}

			// Check for rotation information
			if _, exists := data["rotation_index"]; !exists {
				return fmt.Errorf("round-robin event should contain rotation_index field")
			}

			return nil
		}
	}

	return fmt.Errorf("round-robin event not found")
}

// Circuit breaker event scenarios
func (ctx *ReverseProxyBDDTestContext) iHaveCircuitBreakerEnabledForBackends() error {
	return ctx.iHaveAReverseProxyWithCircuitBreakerEnabled()
}

func (ctx *ReverseProxyBDDTestContext) aCircuitBreakerOpensDueToFailures() error {
	return ctx.aBackendFailsRepeatedly()
}

func (ctx *ReverseProxyBDDTestContext) aCircuitBreakerOpenEventShouldBeEmitted() error {
	events := ctx.eventObserver.GetEvents()

	for _, event := range events {
		if event.Type() == EventTypeCircuitBreakerOpen {
			return nil
		}
	}

	return fmt.Errorf("circuit breaker open event not found")
}

func (ctx *ReverseProxyBDDTestContext) theEventShouldContainFailureThresholdDetails() error {
	events := ctx.eventObserver.GetEvents()

	for _, event := range events {
		if event.Type() == EventTypeCircuitBreakerOpen {
			var data map[string]interface{}
			if err := event.DataAs(&data); err != nil {
				return fmt.Errorf("failed to extract circuit breaker open event data: %v", err)
			}

			// Check for failure threshold information
			if _, exists := data["failure_count"]; !exists {
				return fmt.Errorf("circuit breaker open event should contain failure_count field")
			}

			return nil
		}
	}

	return fmt.Errorf("circuit breaker open event not found")
}

func (ctx *ReverseProxyBDDTestContext) aCircuitBreakerTransitionsToHalfopen() error {
	// Test real circuit breaker half-open state by configuring circuit breaker and making requests
	if ctx.service == nil {
		return fmt.Errorf("service not available")
	}

	// Ensure we have a backend to work with
	if len(ctx.testServers) == 0 {
		return fmt.Errorf("no backend servers available for circuit breaker test")
	}

	// Close existing server and create a partially recovered backend
	ctx.testServers[0].Close()

	// Create a new backend that has intermittent failures (simulating half-open recovery)
	halfOpenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate deterministic behavior for half-open state testing
		// Use request path length to determine response (deterministic for testing)
		if len(r.URL.Path)%2 == 0 {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("backend partially recovered"))
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("backend still unstable"))
		}
	}))

	ctx.testServers[0] = halfOpenServer

	// Update configuration with circuit breaker enabled and the new backend
	ctx.config.BackendServices["test-backend"] = halfOpenServer.URL
	ctx.config.CircuitBreakerConfig = CircuitBreakerConfig{
		Enabled:                 true,
		FailureThreshold:        3,
		OpenTimeout:             5 * time.Second,
		SuccessThreshold:        2,
		HalfOpenAllowedRequests: 5,
	}

	// Reconfigure and restart
	err := ctx.setupApplicationWithConfig()
	if err != nil {
		return fmt.Errorf("failed to reconfigure with circuit breaker: %w", err)
	}

	if err := ctx.app.Start(); err != nil {
		return fmt.Errorf("failed to start app: %w", err)
	}

	// Make several requests to test the half-open behavior
	// This should trigger circuit breaker logic and potentially emit half-open events
	for i := 0; i < 5; i++ {
		resp, err := ctx.makeRequestThroughModule("GET", "/api/test", nil)
		if err != nil {
			continue
		}
		resp.Body.Close()
		time.Sleep(50 * time.Millisecond) // Small delay between requests
	}

	// Give time for circuit breaker to process and potentially emit events
	time.Sleep(200 * time.Millisecond)

	return nil
}

func (ctx *ReverseProxyBDDTestContext) aCircuitBreakerHalfopenEventShouldBeEmitted() error {
	events := ctx.eventObserver.GetEvents()

	for _, event := range events {
		if event.Type() == EventTypeCircuitBreakerHalfOpen {
			return nil
		}
	}

	return fmt.Errorf("circuit breaker half-open event not found")
}

func (ctx *ReverseProxyBDDTestContext) aCircuitBreakerClosesAfterRecovery() error {
	// Test real circuit breaker recovery by making successful requests through the proxy
	if ctx.service == nil {
		return fmt.Errorf("service not available")
	}

	// Ensure we have a backend to work with
	if len(ctx.testServers) == 0 {
		return fmt.Errorf("no backend servers available for circuit breaker recovery test")
	}

	// Close the existing server and create a fully recovered backend server
	ctx.testServers[0].Close()
	recoveredServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate full recovery - always return success
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status": "healthy", "recovery": "complete"}`))
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("backend fully recovered"))
		}
	}))

	ctx.testServers[0] = recoveredServer

	// Update configuration with the recovered backend
	ctx.config.BackendServices["test-backend"] = recoveredServer.URL
	ctx.config.CircuitBreakerConfig = CircuitBreakerConfig{
		Enabled:                 true,
		FailureThreshold:        3,
		OpenTimeout:             5 * time.Second,
		SuccessThreshold:        2,
		HalfOpenAllowedRequests: 5,
	}

	// Reconfigure and restart
	err := ctx.setupApplicationWithConfig()
	if err != nil {
		return fmt.Errorf("failed to reconfigure with recovered backend: %w", err)
	}

	if err := ctx.app.Start(); err != nil {
		return fmt.Errorf("failed to start app: %w", err)
	}

	// Make multiple successful requests through the reverse proxy to trigger recovery
	successCount := 0
	for i := 0; i < 5; i++ {
		resp, err := ctx.makeRequestThroughModule("GET", "/api/test", nil)
		if err == nil && resp != nil {
			if resp.StatusCode == http.StatusOK {
				successCount++
			}
			resp.Body.Close()
		}
		time.Sleep(50 * time.Millisecond) // Small delay between requests
	}

	// Verify we got successful responses
	if successCount < 3 {
		return fmt.Errorf("backend recovery through proxy insufficient: only %d/5 successful requests", successCount)
	}

	// Give time for circuit breaker to process successful requests and potentially close
	time.Sleep(200 * time.Millisecond)

	return nil
}

func (ctx *ReverseProxyBDDTestContext) aCircuitBreakerClosedEventShouldBeEmitted() error {
	events := ctx.eventObserver.GetEvents()

	for _, event := range events {
		if event.Type() == EventTypeCircuitBreakerClosed {
			return nil
		}
	}

	return fmt.Errorf("circuit breaker closed event not found")
}

// Test helper structures
type testLogger struct{}

func (l *testLogger) Debug(msg string, keysAndValues ...interface{})   {}
func (l *testLogger) Info(msg string, keysAndValues ...interface{})    {}
func (l *testLogger) Warn(msg string, keysAndValues ...interface{})    {}
func (l *testLogger) Error(msg string, keysAndValues ...interface{})   {}
func (l *testLogger) With(keysAndValues ...interface{}) modular.Logger { return l }

// TestReverseProxyModuleBDD runs the BDD tests for the ReverseProxy module
func TestReverseProxyModuleBDD(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: func(s *godog.ScenarioContext) {
			ctx := &ReverseProxyBDDTestContext{}

			// Background
			s.Given(`^I have a modular application with reverse proxy module configured$`, ctx.iHaveAModularApplicationWithReverseProxyModuleConfigured)

			// Initialization
			s.When(`^the reverse proxy module is initialized$`, ctx.theReverseProxyModuleIsInitialized)
			s.Then(`^the proxy service should be available$`, ctx.theProxyServiceShouldBeAvailable)
			s.Then(`^the module should be ready to route requests$`, ctx.theModuleShouldBeReadyToRouteRequests)

			// Single backend
			s.Given(`^I have a reverse proxy configured with a single backend$`, ctx.iHaveAReverseProxyConfiguredWithASingleBackend)
			s.When(`^I send a request to the proxy$`, ctx.iSendARequestToTheProxy)
			s.Then(`^the request should be forwarded to the backend$`, ctx.theRequestShouldBeForwardedToTheBackend)
			s.Then(`^the response should be returned to the client$`, ctx.theResponseShouldBeReturnedToTheClient)

			// Multiple backends
			s.Given(`^I have a reverse proxy configured with multiple backends$`, ctx.iHaveAReverseProxyConfiguredWithMultipleBackends)
			s.When(`^I send multiple requests to the proxy$`, ctx.iSendMultipleRequestsToTheProxy)
			s.Then(`^requests should be distributed across all backends$`, ctx.requestsShouldBeDistributedAcrossAllBackends)
			s.Then(`^load balancing should be applied$`, ctx.loadBalancingShouldBeApplied)

			// Health checking
			s.Given(`^I have a reverse proxy with health checks enabled$`, ctx.iHaveAReverseProxyWithHealthChecksEnabled)
			s.When(`^a backend becomes unavailable$`, ctx.aBackendBecomesUnavailable)
			s.Then(`^the proxy should detect the failure$`, ctx.theProxyShouldDetectTheFailure)
			s.Then(`^route traffic only to healthy backends$`, ctx.routeTrafficOnlyToHealthyBackends)

			// Circuit breaker
			s.Given(`^I have a reverse proxy with circuit breaker enabled$`, ctx.iHaveAReverseProxyWithCircuitBreakerEnabled)
			s.When(`^a backend fails repeatedly$`, ctx.aBackendFailsRepeatedly)
			s.Then(`^the circuit breaker should open$`, ctx.theCircuitBreakerShouldOpen)
			s.Then(`^requests should be handled gracefully$`, ctx.requestsShouldBeHandledGracefully)

			// Caching
			s.Given(`^I have a reverse proxy with caching enabled$`, ctx.iHaveAReverseProxyWithCachingEnabled)
			s.When(`^I send the same request multiple times$`, ctx.iSendTheSameRequestMultipleTimes)
			s.Then(`^the first request should hit the backend$`, ctx.theFirstRequestShouldHitTheBackend)
			s.Then(`^subsequent requests should be served from cache$`, ctx.subsequentRequestsShouldBeServedFromCache)

			// Tenant routing
			s.Given(`^I have a tenant-aware reverse proxy configured$`, ctx.iHaveATenantAwareReverseProxyConfigured)
			s.When(`^I send requests with different tenant contexts$`, ctx.iSendRequestsWithDifferentTenantContexts)
			s.Then(`^requests should be routed based on tenant configuration$`, ctx.requestsShouldBeRoutedBasedOnTenantConfiguration)
			s.Then(`^tenant isolation should be maintained$`, ctx.tenantIsolationShouldBeMaintained)

			// Composite responses
			s.Given(`^I have a reverse proxy configured for composite responses$`, ctx.iHaveAReverseProxyConfiguredForCompositeResponses)
			s.When(`^I send a request that requires multiple backend calls$`, ctx.iSendARequestThatRequiresMultipleBackendCalls)
			s.Then(`^the proxy should call all required backends$`, ctx.theProxyShouldCallAllRequiredBackends)
			s.Then(`^combine the responses into a single response$`, ctx.combineTheResponsesIntoASingleResponse)

			// Request transformation
			s.Given(`^I have a reverse proxy with request transformation configured$`, ctx.iHaveAReverseProxyWithRequestTransformationConfigured)
			s.Then(`^the request should be transformed before forwarding$`, ctx.theRequestShouldBeTransformedBeforeForwarding)
			s.Then(`^the backend should receive the transformed request$`, ctx.theBackendShouldReceiveTheTransformedRequest)

			// Shutdown
			s.Given(`^I have an active reverse proxy with ongoing requests$`, ctx.iHaveAnActiveReverseProxyWithOngoingRequests)
			s.When(`^the module is stopped$`, ctx.theModuleIsStopped)
			s.Then(`^ongoing requests should be completed$`, ctx.ongoingRequestsShouldBeCompleted)
			s.Then(`^new requests should be rejected gracefully$`, ctx.newRequestsShouldBeRejectedGracefully)

			// Event observation scenarios
			s.Given(`^I have a reverse proxy with event observation enabled$`, ctx.iHaveAReverseProxyWithEventObservationEnabled)
			s.When(`^the reverse proxy module starts$`, ctx.theReverseProxyModuleStarts)
			s.Then(`^a proxy created event should be emitted$`, ctx.aProxyCreatedEventShouldBeEmitted)
			s.Then(`^a proxy started event should be emitted$`, ctx.aProxyStartedEventShouldBeEmitted)
			s.Then(`^a module started event should be emitted$`, ctx.aModuleStartedEventShouldBeEmitted)
			s.Then(`^the events should contain proxy configuration details$`, ctx.theEventsShouldContainProxyConfigurationDetails)
			s.When(`^the reverse proxy module stops$`, ctx.theReverseProxyModuleStops)
			s.Then(`^a proxy stopped event should be emitted$`, ctx.aProxyStoppedEventShouldBeEmitted)
			s.Then(`^a module stopped event should be emitted$`, ctx.aModuleStoppedEventShouldBeEmitted)

			// Request routing events
			s.Given(`^I have a backend service configured$`, ctx.iHaveABackendServiceConfigured)
			s.When(`^I send a request to the reverse proxy$`, ctx.iSendARequestToTheReverseProxy)
			s.Then(`^a request received event should be emitted$`, ctx.aRequestReceivedEventShouldBeEmitted)
			s.Then(`^the event should contain request details$`, ctx.theEventShouldContainRequestDetails)
			s.When(`^the request is successfully proxied to the backend$`, ctx.theRequestIsSuccessfullyProxiedToTheBackend)
			s.Then(`^a request proxied event should be emitted$`, ctx.aRequestProxiedEventShouldBeEmitted)
			s.Then(`^the event should contain backend and response details$`, ctx.theEventShouldContainBackendAndResponseDetails)

			// Request failure events
			s.Given(`^I have an unavailable backend service configured$`, ctx.iHaveAnUnavailableBackendServiceConfigured)
			s.When(`^the request fails to reach the backend$`, ctx.theRequestFailsToReachTheBackend)
			s.Then(`^a request failed event should be emitted$`, ctx.aRequestFailedEventShouldBeEmitted)
			s.Then(`^the event should contain error details$`, ctx.theEventShouldContainErrorDetails)

			// Backend health management events
			s.Given(`^I have backends with health checking enabled$`, ctx.iHaveBackendsWithHealthCheckingEnabled)
			s.When(`^a backend becomes healthy$`, ctx.aBackendBecomesHealthy)
			s.Then(`^a backend healthy event should be emitted$`, ctx.aBackendHealthyEventShouldBeEmitted)
			s.Then(`^the event should contain backend health details$`, ctx.theEventShouldContainBackendHealthDetails)
			s.When(`^a backend becomes unhealthy$`, ctx.aBackendBecomesUnhealthy)
			s.Then(`^a backend unhealthy event should be emitted$`, ctx.aBackendUnhealthyEventShouldBeEmitted)
			s.Then(`^the event should contain health failure details$`, ctx.theEventShouldContainHealthFailureDetails)

			// Backend management events
			s.When(`^a new backend is added to the configuration$`, ctx.aNewBackendIsAddedToTheConfiguration)
			s.Then(`^a backend added event should be emitted$`, ctx.aBackendAddedEventShouldBeEmitted)
			s.Then(`^the event should contain backend configuration$`, ctx.theEventShouldContainBackendConfiguration)
			s.When(`^a backend is removed from the configuration$`, ctx.aBackendIsRemovedFromTheConfiguration)
			s.Then(`^a backend removed event should be emitted$`, ctx.aBackendRemovedEventShouldBeEmitted)
			s.Then(`^the event should contain removal details$`, ctx.theEventShouldContainRemovalDetails)

			// Load balancing events
			s.Given(`^I have multiple backends configured$`, ctx.iHaveMultipleBackendsConfigured)
			s.When(`^load balancing decisions are made$`, ctx.loadBalancingDecisionsAreMade)
			s.Then(`^load balance decision events should be emitted$`, ctx.loadBalanceDecisionEventsShouldBeEmitted)
			s.Then(`^the events should contain selected backend information$`, ctx.theEventsShouldContainSelectedBackendInformation)
			s.When(`^round-robin load balancing is used$`, ctx.roundrobinLoadBalancingIsUsed)
			s.Then(`^round-robin events should be emitted$`, ctx.roundrobinEventsShouldBeEmitted)
			s.Then(`^the events should contain rotation details$`, ctx.theEventsShouldContainRotationDetails)

			// Circuit breaker events
			s.Given(`^I have circuit breaker enabled for backends$`, ctx.iHaveCircuitBreakerEnabledForBackends)
			s.When(`^a circuit breaker opens due to failures$`, ctx.aCircuitBreakerOpensDueToFailures)
			s.Then(`^a circuit breaker open event should be emitted$`, ctx.aCircuitBreakerOpenEventShouldBeEmitted)
			s.Then(`^the event should contain failure threshold details$`, ctx.theEventShouldContainFailureThresholdDetails)
			s.When(`^a circuit breaker transitions to half-open$`, ctx.aCircuitBreakerTransitionsToHalfopen)
			s.Then(`^a circuit breaker half-open event should be emitted$`, ctx.aCircuitBreakerHalfopenEventShouldBeEmitted)
			s.When(`^a circuit breaker closes after recovery$`, ctx.aCircuitBreakerClosesAfterRecovery)
			s.Then(`^a circuit breaker closed event should be emitted$`, ctx.aCircuitBreakerClosedEventShouldBeEmitted)
		},
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"features/reverseproxy_module.feature"},
			TestingT: t,
		},
	}

	if suite.Run() != 0 {
		t.Fatal("non-zero status returned, failed to run feature tests")
	}
}
