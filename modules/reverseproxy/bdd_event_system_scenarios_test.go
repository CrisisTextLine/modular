package reverseproxy

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/CrisisTextLine/modular"
)

// Event System BDD Step Implementations
// This file implements unique BDD steps for event system scenarios that are not implemented elsewhere

// circuitBreakerBehaviorShouldBeIsolatedPerBackend validates per-backend circuit breaker isolation
func (ctx *ReverseProxyBDDTestContext) circuitBreakerBehaviorShouldBeIsolatedPerBackend() error {
	// Set up multiple backends with circuit breakers
	ctx.resetContext()

	// Create application with reverse proxy config - use ObservableApplication for event support
	logger := &testLogger{}
	mainConfigProvider := modular.NewStdConfigProvider(struct{}{})
	ctx.app = modular.NewObservableApplication(mainConfigProvider, logger)

	// Apply per-app empty feeders to avoid mutating global modular.ConfigFeeders
	if cfSetter, ok := ctx.app.(interface{ SetConfigFeeders([]modular.Feeder) }); ok {
		cfSetter.SetConfigFeeders([]modular.Feeder{})
	}

	// Register router service
	mockRouter := &testRouter{routes: make(map[string]http.HandlerFunc)}
	ctx.app.RegisterService("router", mockRouter)

	// Create two backend servers - one working, one failing
	workingServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("working backend"))
	}))
	ctx.testServers = append(ctx.testServers, workingServer)

	failingServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("failing backend"))
	}))
	ctx.testServers = append(ctx.testServers, failingServer)

	// Configure reverse proxy with both backends and circuit breakers
	ctx.config = &ReverseProxyConfig{
		BackendServices: map[string]string{
			"working-backend": workingServer.URL,
			"failing-backend": failingServer.URL,
		},
		Routes: map[string]string{
			"/working/*": "working-backend",
			"/failing/*": "failing-backend",
		},
		DefaultBackend:       "working-backend",
		CircuitBreakerConfig: CircuitBreakerConfig{Enabled: true, FailureThreshold: 2, OpenTimeout: 100 * time.Millisecond},
	}

	// Create and setup module
	ctx.module = NewModule()
	ctx.service = ctx.module
	ctx.eventObserver = newTestEventObserver()

	// Register observer before module to capture all events
	if err := ctx.app.(modular.Subject).RegisterObserver(ctx.eventObserver); err != nil {
		return fmt.Errorf("failed to register test observer: %w", err)
	}

	ctx.app.RegisterModule(ctx.module)

	// Register config
	reverseproxyConfigProvider := modular.NewStdConfigProvider(ctx.config)
	ctx.app.RegisterConfigSection("reverseproxy", reverseproxyConfigProvider)

	// Initialize and start
	if err := ctx.app.Init(); err != nil {
		return fmt.Errorf("failed to initialize app: %v", err)
	}

	if err := ctx.app.Start(); err != nil {
		return fmt.Errorf("failed to start app: %v", err)
	}

	// Clear events to focus on circuit breaker isolation test
	ctx.eventObserver.ClearEvents()

	// Make requests to failing backend to trigger its circuit breaker
	for i := 0; i < 3; i++ {
		resp, err := ctx.makeRequestThroughModule("GET", "/failing/test", nil)
		if err == nil && resp != nil {
			resp.Body.Close()
		}
		time.Sleep(10 * time.Millisecond) // Small delay between requests
	}

	// Make request to working backend (should still work)
	resp, err := ctx.makeRequestThroughModule("GET", "/working/test", nil)
	if err == nil && resp != nil {
		resp.Body.Close()
	}

	time.Sleep(200 * time.Millisecond) // Allow events to be processed

	// Validate circuit breaker isolation
	events := ctx.eventObserver.GetEvents()
	var circuitBreakerEvents []string
	var failingBackendCBOpen, workingBackendUnaffected bool

	for _, event := range events {
		if event.Type() == EventTypeCircuitBreakerOpen {
			var data map[string]interface{}
			if err := event.DataAs(&data); err != nil {
				continue
			}
			backend, hasBackend := data["backend"]
			if hasBackend {
				circuitBreakerEvents = append(circuitBreakerEvents, fmt.Sprintf("CB_OPEN:%s", backend))
				if backend == "failing-backend" {
					failingBackendCBOpen = true
				}
			}
		}
		if event.Type() == EventTypeRequestProxied {
			var data map[string]interface{}
			if err := event.DataAs(&data); err != nil {
				continue
			}
			backend, hasBackend := data["backend"]
			if hasBackend && backend == "working-backend" {
				workingBackendUnaffected = true
			}
		}
	}

	if !failingBackendCBOpen {
		return fmt.Errorf("expected circuit breaker to open for failing-backend, but no CB open event found. Events: %v", circuitBreakerEvents)
	}

	if !workingBackendUnaffected {
		return fmt.Errorf("expected working-backend to remain unaffected by failing-backend's circuit breaker")
	}

	return nil
}
