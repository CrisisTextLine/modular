package reverseproxy

import (
	"testing"
)

func TestDebugHealthCheck(t *testing.T) {
	ctx := &ReverseProxyBDDTestContext{}

	// Test basic setup
	t.Run("Basic Setup", func(t *testing.T) {
		err := ctx.iHaveAModularApplicationWithReverseProxyModuleConfigured()
		if err != nil {
			t.Logf("Setup failed: %v", err)

			// Debug the context state
			t.Logf("ctx.app: %v", ctx.app)
			t.Logf("ctx.config: %v", ctx.config)

			t.Fatalf("Basic setup failed: %v", err)
		}

		t.Logf("Setup successful!")
		t.Logf("ctx.app: %v", ctx.app != nil)
		t.Logf("ctx.config: %v", ctx.config != nil)

		// Try one health check step
		err = ctx.iHaveAReverseProxyWithHealthChecksConfiguredForDNSResolution()
		if err != nil {
			t.Fatalf("DNS resolution setup failed: %v", err)
		}

		t.Logf("DNS resolution setup successful!")
	})
}
