package cache

import (
	"context"
	"fmt"
	"time"
)

// Event observation BDD test steps for expiration scenarios

func (ctx *CacheBDDTestContext) theCacheCleanupProcessRuns() error {
	// Wait for the natural cleanup process to run
	// With the configured cleanup interval of 500ms, we wait for 3+ cycles to ensure it runs reliably
	time.Sleep(1600 * time.Millisecond)

	// Additionally, proactively trigger cleanup on the in-memory engine to reduce test flakiness
	// and accelerate emission of expiration events in CI environments.
	if ctx.service != nil {
		if mem, ok := ctx.service.cacheEngine.(*MemoryCache); ok {
			// Poll a few times, triggering cleanup and checking if the expired event appeared
			deadline := time.Now().Add(2 * time.Second)
			for time.Now().Before(deadline) {
				mem.CleanupNow(context.Background())
				// Small delay to allow async event emission to propagate
				time.Sleep(50 * time.Millisecond)
				for _, ev := range ctx.eventObserver.GetEvents() {
					if ev.Type() == EventTypeCacheExpired {
						return nil
					}
				}
			}
		}
	}

	return nil
}

func (ctx *CacheBDDTestContext) aCacheExpiredEventShouldBeEmitted() error {
	time.Sleep(100 * time.Millisecond) // Give time for async event emission

	events := ctx.eventObserver.GetEvents()
	for _, event := range events {
		if event.Type() == EventTypeCacheExpired {
			return nil
		}
	}

	eventTypes := make([]string, len(events))
	for i, event := range events {
		eventTypes[i] = event.Type()
	}

	return fmt.Errorf("cache expired event not found. Captured events: %v", eventTypes)
}

func (ctx *CacheBDDTestContext) theExpiredEventShouldContainTheExpiredKey(key string) error {
	events := ctx.eventObserver.GetEvents()
	for _, event := range events {
		if event.Type() == EventTypeCacheExpired {
			// Check if the event data contains the expired key
			data := event.Data()
			if data != nil {
				// Parse the JSON data
				var eventData map[string]interface{}
				if err := event.DataAs(&eventData); err == nil {
					if cacheKey, exists := eventData["cache_key"]; exists && cacheKey == key {
						// Also validate other expected fields
						if _, hasExpiredAt := eventData["expired_at"]; hasExpiredAt {
							if reason, hasReason := eventData["reason"]; hasReason && reason == "ttl_expired" {
								return nil
							}
						}
					}
				}
			}
		}
	}
	return fmt.Errorf("expired event does not contain expected expired key '%s' with proper data structure", key)
}
