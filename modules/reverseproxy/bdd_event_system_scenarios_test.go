package reverseproxy

import (
	"fmt"
	"time"
)

// Event System BDD Step Implementations
// This file implements unique BDD steps for event system scenarios that are not implemented elsewhere

// circuitBreakerBehaviorShouldBeIsolatedPerBackend validates per-backend circuit breaker isolation
func (ctx *ReverseProxyBDDTestContext) circuitBreakerBehaviorShouldBeIsolatedPerBackend() error {
	// Use the existing configuration set up by differentBackendsFailAtDifferentRates()
	// which should have configured failing-backend, intermittent-backend, and healthy-backend

	if ctx.service == nil {
		return fmt.Errorf("service not available - ensure previous steps set up the service")
	}

	if ctx.eventObserver == nil {
		return fmt.Errorf("event observer not available - ensure previous steps set up event observation")
	}

	// Clear events to focus on circuit breaker isolation test
	ctx.eventObserver.ClearEvents()

	// Make requests to failing backend to trigger its circuit breaker
	// Use the routes configured by differentBackendsFailAtDifferentRates(): /api/fail -> failing-backend
	for i := 0; i < 5; i++ { // Make 5 requests (threshold is 2, so this should trigger circuit breaker)
		resp, err := ctx.makeRequestThroughModule("GET", "/api/fail", nil)
		if err == nil && resp != nil {
			resp.Body.Close()
		}
		time.Sleep(20 * time.Millisecond) // Small delay between requests
	}

	// Make request to healthy backend (should still work)
	// Use /api/healthy -> healthy-backend
	resp, err := ctx.makeRequestThroughModule("GET", "/api/healthy", nil)
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
			if hasBackend && backend == "healthy-backend" {
				workingBackendUnaffected = true
			}
		}
	}

	if !failingBackendCBOpen {
		return fmt.Errorf("expected circuit breaker to open for failing-backend, but no CB open event found. Events: %v", circuitBreakerEvents)
	}

	if !workingBackendUnaffected {
		return fmt.Errorf("expected healthy-backend to remain unaffected by failing-backend's circuit breaker")
	}

	return nil
}
