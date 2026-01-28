package reverseproxy

import (
	"testing"
	"time"
)

// TestConfigLeakDetection runs the suspect tests in sequence and logs CacheTTL values
func TestConfigLeakDetection(t *testing.T) {
	t.Run("Before_MergeConfigs", func(t *testing.T) {
		t.Logf("=== BEFORE TestMergeConfigs ===")
	})

	t.Run("MergeConfigs", func(t *testing.T) {
		// Run TestMergeConfigs inline
		TestMergeConfigs(t)
		t.Logf("=== AFTER TestMergeConfigs ===")
	})

	t.Run("PartialTenantConfig", func(t *testing.T) {
		// Run TestPartialTenantConfig inline
		TestPartialTenantConfig(t)
		t.Logf("=== AFTER TestPartialTenantConfig ===")
	})

	t.Run("BDD_CacheTTL_Scenario", func(t *testing.T) {
		t.Logf("=== BEFORE BDD Cache TTL scenario ===")

		// Create a BDD context and run the Given step
		ctx := &ReverseProxyBDDTestContext{}

		// Call the Given step
		err := ctx.iHaveAReverseProxyWithSpecificCacheTTLConfigured()
		if err != nil {
			t.Fatalf("Given step failed: %v", err)
		}

		// Log the actual CacheTTL value
		if ctx.config != nil {
			t.Logf("=== BDD Context CacheTTL: %v ===", ctx.config.CacheTTL)

			// Also check what the sleep time would be
			waitTime := ctx.config.CacheTTL + (500 * time.Millisecond)
			t.Logf("=== Would sleep for: %v ===", waitTime)

			if ctx.config.CacheTTL != 1*time.Second {
				t.Errorf("❌ LEAK DETECTED: Expected CacheTTL=1s, got %v", ctx.config.CacheTTL)
			} else {
				t.Logf("✅ CacheTTL is correct (1s)")
			}
		} else {
			t.Error("ctx.config is nil!")
		}

		// Cleanup
		ctx.resetContext()
	})
}

// TestCacheTTLInWhenStep verifies the When step reads the correct config
func TestCacheTTLInWhenStep(t *testing.T) {
	ctx := &ReverseProxyBDDTestContext{}

	// Setup with 1 second TTL
	err := ctx.iHaveAReverseProxyWithSpecificCacheTTLConfigured()
	if err != nil {
		t.Fatalf("Given step failed: %v", err)
	}

	t.Logf("After Given: CacheTTL = %v", ctx.config.CacheTTL)

	// Now check what cachedResponsesAgeBeyondTTL would see
	if ctx.config == nil || ctx.config.CacheTTL <= 0 {
		t.Fatal("Cache TTL not configured properly in Given step")
	}

	ttl := ctx.config.CacheTTL
	waitTime := ttl + (500 * time.Millisecond)

	t.Logf("When step would sleep for: %v (TTL=%v + 500ms)", waitTime, ttl)

	if waitTime > 10*time.Second {
		t.Errorf("❌ PROBLEM: Would sleep for %v which is > 10s", waitTime)
		t.Errorf("This explains why tests hang!")
	} else {
		t.Logf("✅ Sleep time is reasonable: %v", waitTime)
	}

	ctx.resetContext()
}

// TestPrintAllCacheTTLValues prints all CacheTTL values in the test file
func TestPrintAllCacheTTLValues(t *testing.T) {
	t.Logf("Searching for CacheTTL values in codebase...")

	configs := []struct {
		name string
		ttl  time.Duration
	}{
		{"bdd_caching_tenant_test.go:42 (iHaveAReverseProxyWithCachingEnabled)", 300 * time.Second},
		{"bdd_caching_tenant_test.go:168 (iHaveAReverseProxyWithSpecificCacheTTLConfigured)", 1 * time.Second},
		{"config_merge_test.go:34", 120 * time.Second},
		{"module_test.go:812", 120 * time.Second},
	}

	for _, cfg := range configs {
		t.Logf("%s: CacheTTL = %v", cfg.name, cfg.ttl)
		if cfg.ttl >= 60*time.Second {
			t.Logf("  ⚠️  This is a long TTL that could cause test hangs")
		}
	}

	t.Logf("\nIf BDD test uses wrong config, it could sleep for 120s or 300s!")
}
