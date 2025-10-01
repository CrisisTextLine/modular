package reverseproxy

import (
	"testing"
	"time"
)

// TestSequentialCacheScenarios simulates running Response caching followed by Cache TTL behavior
func TestSequentialCacheScenarios(t *testing.T) {
	ctx := &ReverseProxyBDDTestContext{}

	t.Run("Response_caching_scenario", func(t *testing.T) {
		// This is the Given step from "Response caching" scenario
		err := ctx.iHaveAReverseProxyWithCachingEnabled()
		if err != nil {
			t.Fatalf("Response caching Given failed: %v", err)
		}

		t.Logf("After Response caching Given: CacheTTL = %v", ctx.config.CacheTTL)

		if ctx.config.CacheTTL != 300*time.Second {
			t.Errorf("Expected CacheTTL=300s for Response caching, got %v", ctx.config.CacheTTL)
		}

		// Cleanup after this scenario (simulating what happens between scenarios)
		ctx.resetContext()
		t.Logf("After resetContext: ctx.config = %v", ctx.config)
	})

	t.Run("Cache_TTL_behavior_scenario", func(t *testing.T) {
		// This is the Given step from "Cache TTL behavior" scenario
		err := ctx.iHaveAReverseProxyWithSpecificCacheTTLConfigured()
		if err != nil {
			t.Fatalf("Cache TTL behavior Given failed: %v", err)
		}

		t.Logf("After Cache TTL Given: CacheTTL = %v", ctx.config.CacheTTL)

		if ctx.config.CacheTTL != 1*time.Second {
			t.Errorf("❌ LEAK DETECTED: Expected CacheTTL=1s, got %v", ctx.config.CacheTTL)
			t.Errorf("This means resetContext() didn't fully isolate the scenarios!")
		} else {
			t.Logf("✅ CacheTTL is correct (1s)")
		}

		// Check what the When step would see
		ttl := ctx.config.CacheTTL
		waitTime := ttl + (500 * time.Millisecond)
		t.Logf("When step would sleep for: %v", waitTime)

		if waitTime > 2*time.Second {
			t.Errorf("❌ Would sleep for %v which is too long!", waitTime)
		}

		ctx.resetContext()
	})
}
