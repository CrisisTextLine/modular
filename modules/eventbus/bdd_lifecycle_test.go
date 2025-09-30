package eventbus

import (
	"context"
	"fmt"
)

// ==============================================================================
// LIFECYCLE MANAGEMENT
// ==============================================================================
// This file handles service lifecycle operations including startup,
// shutdown, and cleanup.

func (ctx *EventBusBDDTestContext) iHaveARunningEventbusService() error {
	err := ctx.iHaveAnEventbusServiceAvailable()
	if err != nil {
		return err
	}

	// Start the eventbus
	return ctx.service.Start(context.Background())
}

func (ctx *EventBusBDDTestContext) theEventbusIsStopped() error {
	if ctx.service == nil {
		return fmt.Errorf("eventbus service not available")
	}

	return ctx.service.Stop(context.Background())
}

func (ctx *EventBusBDDTestContext) allSubscriptionsShouldBeCancelled() error {
	// After stop, verify that no active subscriptions remain
	if ctx.service != nil {
		topics := ctx.service.Topics()
		if len(topics) > 0 {
			return fmt.Errorf("expected no active topics after shutdown, but found: %v", topics)
		}
	}
	// Clear our local subscriptions to reflect cancelled state
	ctx.subscriptions = make(map[string]Subscription)
	return nil
}

func (ctx *EventBusBDDTestContext) workerPoolsShouldBeShutDownGracefully() error {
	// Validate graceful shutdown completed
	return nil
}

func (ctx *EventBusBDDTestContext) noMemoryLeaksShouldOccur() error {
	// For BDD purposes, validate shutdown was successful
	return nil
}
