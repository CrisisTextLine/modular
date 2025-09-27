package reverseproxy

import (
	"context"
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

	// Don't clear events yet - we need to ensure the observer is properly connected first
	// Clear events to focus on circuit breaker isolation test
	if ctx.eventObserver != nil {
		ctx.eventObserver.ClearEvents()
	}

	// CRITICAL: Ensure circuit breakers have proper event emission
	if ctx.module != nil {
		// Re-establish event emission for all circuit breakers
		for backendID, cb := range ctx.module.circuitBreakers {
			if cb != nil {
				cb.eventEmitter = func(eventType string, data map[string]interface{}) {
					ctx.module.emitEvent(context.Background(), eventType, data)
				}
				if ctx.app != nil && ctx.app.Logger() != nil {
					ctx.app.Logger().Info("Re-established event emitter for circuit breaker", "backend", backendID)
				}
			}
		}
	}

	// Make requests to failing backend to trigger its circuit breaker
	// Use the routes configured by differentBackendsFailAtDifferentRates(): /api/fail -> failing-backend
	// The failing-backend has a failure threshold of 2, so we need at least 2 failures to open it
	for i := 0; i < 6; i++ { // Make 6 requests to ensure circuit breaker opens (threshold is 2)
		resp, err := ctx.makeRequestThroughModule("GET", "/api/fail", nil)
		if err == nil && resp != nil {
			resp.Body.Close()
		}
		time.Sleep(50 * time.Millisecond) // Longer delay to ensure events are processed
	}

	// Wait for circuit breaker events to be processed
	time.Sleep(300 * time.Millisecond)

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
		// Accept both open and half-open circuit breaker events as valid
		if event.Type() == EventTypeCircuitBreakerOpen || event.Type() == EventTypeCircuitBreakerHalfOpen {
			var data map[string]interface{}
			if err := event.DataAs(&data); err != nil {
				continue
			}
			backend, hasBackend := data["backend"]
			if hasBackend {
				circuitBreakerEvents = append(circuitBreakerEvents, fmt.Sprintf("CB_%s:%s",
					map[bool]string{true: "OPEN", false: "HALFOPEN"}[event.Type() == EventTypeCircuitBreakerOpen], backend))
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
		// Debug: Let's see ALL events, not just circuit breaker events
		allEventTypes := make([]string, len(events))
		for i, event := range events {
			allEventTypes[i] = event.Type()
		}
		return fmt.Errorf("expected circuit breaker to open for failing-backend, but no CB open event found. Circuit breaker events: %v, All events: %v", circuitBreakerEvents, allEventTypes)
	}

	if !workingBackendUnaffected {
		return fmt.Errorf("expected healthy-backend to remain unaffected by failing-backend's circuit breaker")
	}

	return nil
}
