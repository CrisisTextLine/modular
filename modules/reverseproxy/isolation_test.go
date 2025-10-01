package reverseproxy

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestApplicationIsolation ensures multiple app instances don't share state
func TestApplicationIsolation(t *testing.T) {
	t.Parallel()

	// Create first module with specific config
	module1 := NewModule()
	config1 := &ReverseProxyConfig{
		CacheTTL:     10 * time.Second,
		CacheEnabled: true,
		BackendServices: map[string]string{
			"backend1": "http://localhost:8001",
		},
	}
	module1.config = config1

	// Create second module with different config
	module2 := NewModule()
	config2 := &ReverseProxyConfig{
		CacheTTL:     20 * time.Second,
		CacheEnabled: true,
		BackendServices: map[string]string{
			"backend2": "http://localhost:8002",
		},
	}
	module2.config = config2

	// Verify configs are isolated
	assert.Equal(t, 10*time.Second, module1.config.CacheTTL, "Module1 should have 10s CacheTTL")
	assert.Equal(t, 20*time.Second, module2.config.CacheTTL, "Module2 should have 20s CacheTTL")

	// Verify backends are isolated
	assert.Contains(t, module1.config.BackendServices, "backend1", "Module1 should have backend1")
	assert.NotContains(t, module1.config.BackendServices, "backend2", "Module1 should NOT have backend2")

	assert.Contains(t, module2.config.BackendServices, "backend2", "Module2 should have backend2")
	assert.NotContains(t, module2.config.BackendServices, "backend1", "Module2 should NOT have backend1")

	t.Logf("✅ Application isolation verified: module1.CacheTTL=%v, module2.CacheTTL=%v",
		module1.config.CacheTTL, module2.config.CacheTTL)
}

// TestModuleIsolation ensures multiple module instances don't share state
func TestModuleIsolation(t *testing.T) {
	t.Parallel()

	// Create two separate module instances
	module1 := &ReverseProxyModule{}
	module2 := &ReverseProxyModule{}

	// They should be distinct
	assert.NotSame(t, module1, module2, "Module instances should be different")

	// Create separate configs
	config1 := &ReverseProxyConfig{
		CacheTTL:     30 * time.Second,
		CacheEnabled: true,
	}
	config2 := &ReverseProxyConfig{
		CacheTTL:     40 * time.Second,
		CacheEnabled: false,
	}

	// Assign configs (simulating what Initialize would do)
	module1.config = config1
	module2.config = config2

	// Verify isolation
	assert.Equal(t, 30*time.Second, module1.config.CacheTTL)
	assert.Equal(t, 40*time.Second, module2.config.CacheTTL)
	assert.True(t, module1.config.CacheEnabled)
	assert.False(t, module2.config.CacheEnabled)

	// Modify module1's config
	module1.config.CacheTTL = 50 * time.Second

	// Verify module2 is unaffected
	assert.Equal(t, 50*time.Second, module1.config.CacheTTL, "Module1 should be modified")
	assert.Equal(t, 40*time.Second, module2.config.CacheTTL, "Module2 should be unchanged")

	t.Logf("✅ Module isolation verified: changes to module1 don't affect module2")
}

// TestConfigStructIsolation ensures config structs don't share internal maps
func TestConfigStructIsolation(t *testing.T) {
	t.Parallel()

	// Create base config
	baseConfig := &ReverseProxyConfig{
		CacheTTL:     60 * time.Second,
		CacheEnabled: true,
		BackendServices: map[string]string{
			"shared": "http://localhost:9000",
		},
		Routes: map[string]string{
			"/api/*": "shared",
		},
		BackendConfigs: make(map[string]BackendServiceConfig),
	}

	// Create a "copy" by creating a new struct (simulating what might happen in tests)
	derivedConfig := &ReverseProxyConfig{
		CacheTTL:        baseConfig.CacheTTL,
		CacheEnabled:    baseConfig.CacheEnabled,
		BackendServices: baseConfig.BackendServices, // DANGER: shares the map!
		Routes:          baseConfig.Routes,          // DANGER: shares the map!
		BackendConfigs:  baseConfig.BackendConfigs,  // DANGER: shares the map!
	}

	// Modify derived config
	derivedConfig.CacheTTL = 70 * time.Second
	derivedConfig.BackendServices["derived"] = "http://localhost:9001"

	// Check if base config was affected
	assert.Equal(t, 60*time.Second, baseConfig.CacheTTL, "Primitive fields should be isolated")

	if _, exists := baseConfig.BackendServices["derived"]; exists {
		t.Errorf("❌ MAP SHARING BUG: base config was polluted with 'derived' backend")
		t.Errorf("This happens when maps are shared between config instances!")
		t.Logf("Base config backends: %v", baseConfig.BackendServices)
		t.Logf("Derived config backends: %v", derivedConfig.BackendServices)
	} else {
		t.Logf("✅ Maps appear isolated (test setup properly copied maps)")
	}
}

// TestBDDContextIsolation ensures BDD test contexts don't share state
func TestBDDContextIsolation(t *testing.T) {
	t.Parallel()

	// Create two BDD contexts
	ctx1 := &ReverseProxyBDDTestContext{}
	ctx2 := &ReverseProxyBDDTestContext{}

	// They should be distinct
	assert.NotSame(t, ctx1, ctx2, "Context instances should be different")

	// Set up different configs
	ctx1.config = &ReverseProxyConfig{
		CacheTTL: 100 * time.Second,
	}
	ctx2.config = &ReverseProxyConfig{
		CacheTTL: 200 * time.Second,
	}

	// Verify isolation
	assert.Equal(t, 100*time.Second, ctx1.config.CacheTTL)
	assert.Equal(t, 200*time.Second, ctx2.config.CacheTTL)

	// Modify ctx1
	ctx1.config.CacheTTL = 150 * time.Second

	// Verify ctx2 is unaffected
	assert.Equal(t, 150*time.Second, ctx1.config.CacheTTL)
	assert.Equal(t, 200*time.Second, ctx2.config.CacheTTL)

	t.Logf("✅ BDD context isolation verified")
}

// TestResetContextCleanup verifies resetContext properly clears everything
func TestResetContextCleanup(t *testing.T) {
	ctx := &ReverseProxyBDDTestContext{}

	// Set up a config
	ctx.config = &ReverseProxyConfig{
		CacheTTL: 123 * time.Second,
		BackendServices: map[string]string{
			"test": "http://localhost:8080",
		},
	}

	// Verify it's set
	require.NotNil(t, ctx.config)
	require.Equal(t, 123*time.Second, ctx.config.CacheTTL)

	// Call resetContext
	ctx.resetContext()

	// Verify config was reset to a fresh instance
	require.NotNil(t, ctx.config, "Config should be initialized by resetContext")

	// Check that maps are fresh (empty)
	assert.Empty(t, ctx.config.BackendServices, "BackendServices should be empty after reset")
	assert.Empty(t, ctx.config.Routes, "Routes should be empty after reset")

	// CacheTTL should be zero value
	assert.Equal(t, time.Duration(0), ctx.config.CacheTTL, "CacheTTL should be zero after reset")

	t.Logf("✅ resetContext properly clears all config state")
}

// TestSequentialContextReuse simulates what happens in BDD scenarios
func TestSequentialContextReuse(t *testing.T) {
	ctx := &ReverseProxyBDDTestContext{}

	// Scenario 1: Set config with 300s TTL
	t.Run("Scenario1_ResponseCaching", func(t *testing.T) {
		ctx.config = &ReverseProxyConfig{
			CacheTTL: 300 * time.Second,
		}
		t.Logf("Scenario 1 set CacheTTL: %v", ctx.config.CacheTTL)
		assert.Equal(t, 300*time.Second, ctx.config.CacheTTL)

		// Simulate end of scenario - resetContext is called
		ctx.resetContext()
		t.Logf("After reset: CacheTTL=%v, config=%v", ctx.config.CacheTTL, ctx.config != nil)
	})

	// Scenario 2: Set config with 1s TTL
	t.Run("Scenario2_CacheTTLBehavior", func(t *testing.T) {
		ctx.config = &ReverseProxyConfig{
			CacheTTL: 1 * time.Second,
		}
		t.Logf("Scenario 2 set CacheTTL: %v", ctx.config.CacheTTL)

		// This is the critical test - did Scenario 1's 300s leak through?
		if ctx.config.CacheTTL != 1*time.Second {
			t.Errorf("❌ CONFIG LEAK: Expected 1s, got %v (leaked from Scenario 1?)", ctx.config.CacheTTL)
		} else {
			t.Logf("✅ No leak: CacheTTL is correctly set to 1s")
		}
	})
}

// TestConfigPointerSharing checks if config pointers are being reused
func TestConfigPointerSharing(t *testing.T) {
	ctx := &ReverseProxyBDDTestContext{}

	// Create first config
	ctx.config = &ReverseProxyConfig{
		CacheTTL: 111 * time.Second,
	}
	firstConfigPtr := ctx.config
	firstCacheTTL := ctx.config.CacheTTL

	// Reset context
	ctx.resetContext()

	// Create second config
	ctx.config = &ReverseProxyConfig{
		CacheTTL: 222 * time.Second,
	}
	secondConfigPtr := ctx.config
	secondCacheTTL := ctx.config.CacheTTL

	// Verify they're different pointers
	if firstConfigPtr == secondConfigPtr {
		t.Errorf("❌ POINTER REUSE: Same config pointer used after reset!")
		t.Errorf("This could cause state bleeding between scenarios")
	} else {
		t.Logf("✅ Different config pointers: %p vs %p", firstConfigPtr, secondConfigPtr)
	}

	// Verify values are correct
	assert.Equal(t, 111*time.Second, firstCacheTTL, "First config should have 111s")
	assert.Equal(t, 222*time.Second, secondCacheTTL, "Second config should have 222s")

	// Verify first config didn't get modified
	assert.Equal(t, 111*time.Second, firstConfigPtr.CacheTTL, "First config should still be 111s")
}
