package modular

import (
	"context"
	"regexp"
	"testing"
)

// Test_TenantFunctionality tests tenant-related methods
func Test_TenantFunctionality(t *testing.T) {
	// Setup tenant service and configs
	tenantSvc := &mockTenantService{
		tenantConfigs: map[TenantID]map[string]ConfigProvider{
			"tenant1": {
				"app": NewStdConfigProvider(testCfg{Str: "tenant1-config"}),
			},
		},
	}

	app := &StdApplication{
		cfgProvider:    NewStdConfigProvider(testCfg{Str: "test"}),
		cfgSections:    make(map[string]ConfigProvider),
		svcRegistry:    make(ServiceRegistry),
		moduleRegistry: make(ModuleRegistry),
		logger:         &logger{t},
		ctx:            context.Background(),
	}

	// Register tenant service
	if err := app.RegisterService("tenantService", tenantSvc); err != nil {
		t.Fatalf("Failed to register tenant service: %v", err)
	}
	if err := app.RegisterService("tenantConfigLoader", NewFileBasedTenantConfigLoader(TenantConfigParams{
		ConfigNameRegex: regexp.MustCompile(`.*\.json$`),
		ConfigDir:       "",
		ConfigFeeders:   nil,
	})); err != nil {
		t.Fatalf("Failed to register tenant config loader: %v", err)
	}

	// Test GetTenantService
	t.Run("GetTenantService", func(t *testing.T) {
		ts, err := app.GetTenantService()
		if err != nil {
			t.Errorf("GetTenantService() error = %v, expected no error", err)
			return
		}
		if ts == nil {
			t.Error("GetTenantService() returned nil service")
		}
	})

	// Test WithTenant
	t.Run("WithTenant", func(t *testing.T) {
		tctx, err := app.WithTenant("tenant1")
		if err != nil {
			t.Errorf("WithTenant() error = %v, expected no error", err)
			return
		}
		if tctx == nil {
			t.Error("WithTenant() returned nil context")
			return
		}
		if tctx.GetTenantID() != "tenant1" {
			t.Errorf("WithTenant() tenantID = %v, expected tenant1", tctx.GetTenantID())
		}
	})

	// Test WithTenant with nil context
	t.Run("WithTenant with nil context", func(t *testing.T) {
		appWithNoCtx := &StdApplication{
			cfgProvider:    NewStdConfigProvider(testCfg{Str: "test"}),
			cfgSections:    make(map[string]ConfigProvider),
			svcRegistry:    make(ServiceRegistry),
			moduleRegistry: make(ModuleRegistry),
			logger:         &logger{t},
			ctx:            nil, // No context initialized
		}

		_, err := appWithNoCtx.WithTenant("tenant1")
		if err == nil {
			t.Error("WithTenant() expected error for nil app context, got nil")
		}
	})

	// Test GetTenantConfig
	t.Run("GetTenantConfig", func(t *testing.T) {
		cfg, err := app.GetTenantConfig("tenant1", "app")
		if err != nil {
			t.Errorf("GetTenantConfig() error = %v, expected no error", err)
			return
		}
		if cfg == nil {
			t.Error("GetTenantConfig() returned nil config")
			return
		}

		// Verify the config content
		tcfg, ok := cfg.GetConfig().(testCfg)
		if !ok {
			t.Errorf("Failed to get structured config: %v", err)
			return
		}
		if tcfg.Str != "tenant1-config" {
			t.Errorf("Expected config value 'tenant1-config', got '%s'", tcfg.Str)
		}
	})

	// Test GetTenantConfig for non-existent tenant
	t.Run("GetTenantConfig for non-existent tenant", func(t *testing.T) {
		_, err := app.GetTenantConfig("non-existent", "app")
		if err == nil {
			t.Error("GetTenantConfig() expected error for non-existent tenant, got nil")
		}
	})
}
