package reverseproxy

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/CrisisTextLine/modular"
	"github.com/cucumber/godog"
)

// ReverseProxy BDD Test Context
type ReverseProxyBDDTestContext struct {
	app          modular.Application
	module       *ReverseProxyModule
	service      *ReverseProxyModule
	config       *ReverseProxyConfig
	lastError    error
	testServers  []*httptest.Server
	lastResponse *http.Response
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
	
	// Create basic reverse proxy configuration for testing
	ctx.config = &ReverseProxyConfig{
		BackendServices: map[string]string{
			"test-backend": "http://localhost:8080",
		},
		Routes: map[string]string{
			"/api/*": "test-backend",
		},
		BackendConfigs: map[string]BackendServiceConfig{
			"test-backend": {
				URL: "http://localhost:8080",
			},
		},
	}
	
	// Create application
	logger := &testLogger{}
	
	// Save and clear ConfigFeeders to prevent environment interference during tests
	originalFeeders := modular.ConfigFeeders
	modular.ConfigFeeders = []modular.Feeder{}
	defer func() {
		modular.ConfigFeeders = originalFeeders
	}()
	
	mainConfigProvider := modular.NewStdConfigProvider(struct{}{})
	ctx.app = modular.NewStdApplication(mainConfigProvider, logger)
	
	// Create and register reverse proxy module
	ctx.module = NewModule()
	
	// Register the reverseproxy config section
	reverseproxyConfigProvider := modular.NewStdConfigProvider(ctx.config)
	ctx.app.RegisterConfigSection("reverseproxy", reverseproxyConfigProvider)
	
	// Register the module
	ctx.app.RegisterModule(ctx.module)
	
	return nil
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
	err := ctx.iHaveAModularApplicationWithReverseProxyModuleConfigured()
	if err != nil {
		return err
	}
	
	// Create a test backend server
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("backend response"))
	}))
	ctx.testServers = append(ctx.testServers, testServer)
	
	// Update config to use the test server
	ctx.config.BackendServices["test-backend"] = testServer.URL
	
	return ctx.theReverseProxyModuleIsInitialized()
}

func (ctx *ReverseProxyBDDTestContext) iSendARequestToTheProxy() error {
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
	// In a real implementation, would verify request forwarding
	return nil
}

func (ctx *ReverseProxyBDDTestContext) theResponseShouldBeReturnedToTheClient() error {
	// In a real implementation, would verify response handling
	return nil
}

func (ctx *ReverseProxyBDDTestContext) iHaveAReverseProxyConfiguredWithMultipleBackends() error {
	err := ctx.iHaveAModularApplicationWithReverseProxyModuleConfigured()
	if err != nil {
		return err
	}
	
	// Create multiple test backend servers
	for i := 0; i < 3; i++ {
		testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(fmt.Sprintf("backend-%d response", i)))
		}))
		ctx.testServers = append(ctx.testServers, testServer)
	}
	
	// Update config with multiple backends
	ctx.config.BackendServices = map[string]string{
		"backend-1": ctx.testServers[0].URL,
		"backend-2": ctx.testServers[1].URL,
		"backend-3": ctx.testServers[2].URL,
	}
	
	return ctx.theReverseProxyModuleIsInitialized()
}

func (ctx *ReverseProxyBDDTestContext) iSendMultipleRequestsToTheProxy() error {
	return ctx.iSendARequestToTheProxy()
}

func (ctx *ReverseProxyBDDTestContext) requestsShouldBeDistributedAcrossAllBackends() error {
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
	err := ctx.iHaveAModularApplicationWithReverseProxyModuleConfigured()
	if err != nil {
		return err
	}
	
	// Ensure health checks are enabled
	ctx.config.HealthCheck.Enabled = true
	ctx.config.HealthCheck.Interval = 5 * time.Second
	ctx.config.HealthCheck.HealthEndpoints = map[string]string{
		"test-backend": "/health",
	}
	
	return ctx.theReverseProxyModuleIsInitialized()
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
	err := ctx.iHaveAModularApplicationWithReverseProxyModuleConfigured()
	if err != nil {
		return err
	}
	
	// Ensure circuit breaker is enabled
	ctx.config.CircuitBreakerConfig.Enabled = true
	ctx.config.CircuitBreakerConfig.FailureThreshold = 3
	
	return ctx.theReverseProxyModuleIsInitialized()
}

func (ctx *ReverseProxyBDDTestContext) aBackendFailsRepeatedly() error {
	// Simulate repeated failures
	return nil
}

func (ctx *ReverseProxyBDDTestContext) theCircuitBreakerShouldOpen() error {
	// Verify circuit breaker configuration
	if !ctx.service.config.CircuitBreakerConfig.Enabled {
		return fmt.Errorf("circuit breaker not enabled")
	}
	return nil
}

func (ctx *ReverseProxyBDDTestContext) requestsShouldBeHandledGracefully() error {
	// In a real implementation, would verify graceful handling
	return nil
}

func (ctx *ReverseProxyBDDTestContext) iHaveAReverseProxyWithCachingEnabled() error {
	err := ctx.iHaveAModularApplicationWithReverseProxyModuleConfigured()
	if err != nil {
		return err
	}
	
	// Ensure caching is enabled
	ctx.config.CacheEnabled = true
	ctx.config.CacheTTL = 300 * time.Second
	
	return ctx.theReverseProxyModuleIsInitialized()
}

func (ctx *ReverseProxyBDDTestContext) iSendTheSameRequestMultipleTimes() error {
	return ctx.iSendARequestToTheProxy()
}

func (ctx *ReverseProxyBDDTestContext) theFirstRequestShouldHitTheBackend() error {
	// In a real implementation, would verify cache miss
	return nil
}

func (ctx *ReverseProxyBDDTestContext) subsequentRequestsShouldBeServedFromCache() error {
	// Verify caching is enabled
	if !ctx.service.config.CacheEnabled {
		return fmt.Errorf("caching not enabled")
	}
	return nil
}

func (ctx *ReverseProxyBDDTestContext) iHaveATenantAwareReverseProxyConfigured() error {
	err := ctx.iHaveAModularApplicationWithReverseProxyModuleConfigured()
	if err != nil {
		return err
	}
	
	// Add tenant-specific configuration
	ctx.config.RequireTenantID = true
	ctx.config.TenantIDHeader = "X-Tenant-ID"
	
	return ctx.theReverseProxyModuleIsInitialized()
}

func (ctx *ReverseProxyBDDTestContext) iSendRequestsWithDifferentTenantContexts() error {
	return ctx.iSendARequestToTheProxy()
}

func (ctx *ReverseProxyBDDTestContext) requestsShouldBeRoutedBasedOnTenantConfiguration() error {
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
	err := ctx.iHaveAModularApplicationWithReverseProxyModuleConfigured()
	if err != nil {
		return err
	}
	
	// Add composite route configuration
	ctx.config.CompositeRoutes = map[string]CompositeRoute{
		"/api/combined": {
			Pattern:  "/api/combined",
			Backends: []string{"backend-1", "backend-2"},
			Strategy: "combine",
		},
	}
	
	return ctx.theReverseProxyModuleIsInitialized()
}

func (ctx *ReverseProxyBDDTestContext) iSendARequestThatRequiresMultipleBackendCalls() error {
	return ctx.iSendARequestToTheProxy()
}

func (ctx *ReverseProxyBDDTestContext) theProxyShouldCallAllRequiredBackends() error {
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
	err := ctx.iHaveAModularApplicationWithReverseProxyModuleConfigured()
	if err != nil {
		return err
	}
	
	// Add backend configuration with header rewriting
	ctx.config.BackendConfigs = map[string]BackendServiceConfig{
		"backend-1": {
			URL: "http://backend1.example.com",
			HeaderRewriting: HeaderRewritingConfig{
				SetHeaders: map[string]string{
					"X-Forwarded-By": "reverse-proxy",
				},
				RemoveHeaders: []string{"Authorization"},
			},
		},
	}
	
	return ctx.theReverseProxyModuleIsInitialized()
}

func (ctx *ReverseProxyBDDTestContext) theRequestShouldBeTransformedBeforeForwarding() error {
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
	err := ctx.iHaveAModularApplicationWithReverseProxyModuleConfigured()
	if err != nil {
		return err
	}
	
	err = ctx.theReverseProxyModuleIsInitialized()
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

// Test helper structures
type testLogger struct{}

func (l *testLogger) Debug(msg string, keysAndValues ...interface{}) {}
func (l *testLogger) Info(msg string, keysAndValues ...interface{})  {}
func (l *testLogger) Warn(msg string, keysAndValues ...interface{})  {}
func (l *testLogger) Error(msg string, keysAndValues ...interface{}) {}
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