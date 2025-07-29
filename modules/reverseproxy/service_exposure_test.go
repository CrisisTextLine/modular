package reverseproxy

import (
	"context"
	"net/http"
	"reflect"
	"testing"

	"github.com/CrisisTextLine/modular"
)

// TestFeatureFlagEvaluatorServiceExposure tests that the module exposes the feature flag evaluator as a service
func TestFeatureFlagEvaluatorServiceExposure(t *testing.T) {
	tests := []struct {
		name              string
		config            *ReverseProxyConfig
		expectService     bool
		expectGlobalFlags int
		expectTenantFlags int
	}{
		{
			name: "FeatureFlagsDisabled",
			config: &ReverseProxyConfig{
				BackendServices: map[string]string{
					"test": "http://test:8080",
				},
				FeatureFlags: FeatureFlagsConfig{
					Enabled: false,
				},
			},
			expectService: false,
		},
		{
			name: "FeatureFlagsEnabledNoDefaults",
			config: &ReverseProxyConfig{
				BackendServices: map[string]string{
					"test": "http://test:8080",
				},
				FeatureFlags: FeatureFlagsConfig{
					Enabled: true,
				},
			},
			expectService:     true,
			expectGlobalFlags: 0,
			expectTenantFlags: 0,
		},
		{
			name: "FeatureFlagsEnabledWithDefaults",
			config: &ReverseProxyConfig{
				BackendServices: map[string]string{
					"test": "http://test:8080",
				},
				FeatureFlags: FeatureFlagsConfig{
					Enabled: true,
					GlobalFlags: map[string]bool{
						"global-flag-1": true,
						"global-flag-2": false,
					},
					TenantFlags: map[string]map[string]bool{
						"tenant1": {
							"tenant-flag-1": true,
						},
						"tenant2": {
							"tenant-flag-2": false,
						},
					},
				},
			},
			expectService:     true,
			expectGlobalFlags: 2,
			expectTenantFlags: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock router
			mockRouter := &testRouter{routes: make(map[string]http.HandlerFunc)}

			// Create mock application
			app := NewMockTenantApplication()

			// Create module
			module := NewModule()

			// Set the configuration
			module.config = tt.config

			// Set router via constructor
			services := map[string]any{
				"router": mockRouter,
			}
			constructedModule, err := module.Constructor()(app, services)
			if err != nil {
				t.Fatalf("Failed to construct module: %v", err)
			}
			module = constructedModule.(*ReverseProxyModule)

			// Set the app reference
			module.app = app

			// Start the module to trigger feature flag evaluator creation
			if err := module.Start(context.Background()); err != nil {
				t.Fatalf("Failed to start module: %v", err)
			}

			// Test service exposure
			providedServices := module.ProvidesServices()

			if tt.expectService {
				// Should provide exactly one service (featureFlagEvaluator)
				if len(providedServices) != 1 {
					t.Errorf("Expected 1 provided service, got %d", len(providedServices))
					return
				}

				service := providedServices[0]
				if service.Name != "featureFlagEvaluator" {
					t.Errorf("Expected service name 'featureFlagEvaluator', got '%s'", service.Name)
				}

				// Verify the service implements FeatureFlagEvaluator
				if _, ok := service.Instance.(FeatureFlagEvaluator); !ok {
					t.Errorf("Expected service to implement FeatureFlagEvaluator, got %T", service.Instance)
				}

				// Test that it's the FileBasedFeatureFlagEvaluator specifically
				evaluator, ok := service.Instance.(*FileBasedFeatureFlagEvaluator)
				if !ok {
					t.Errorf("Expected service to be *FileBasedFeatureFlagEvaluator, got %T", service.Instance)
					return
				}

				// Test configuration was applied correctly
				req, _ := http.NewRequestWithContext(context.Background(), "GET", "/test", nil)

				// Test global flags
				if tt.expectGlobalFlags > 0 {
					for flagID, expectedValue := range tt.config.FeatureFlags.GlobalFlags {
						actualValue, err := evaluator.EvaluateFlag(context.Background(), flagID, "", req)
						if err != nil {
							t.Errorf("Error evaluating flag %s: %v", flagID, err)
						}
						if actualValue != expectedValue {
							t.Errorf("Global flag %s: expected %v, got %v", flagID, expectedValue, actualValue)
						}
					}
				}

				// Test tenant flags
				if tt.expectTenantFlags > 0 {
					for tenantIDStr, tenantFlags := range tt.config.FeatureFlags.TenantFlags {
						tenantID := modular.TenantID(tenantIDStr)
						for flagID, expectedValue := range tenantFlags {
							actualValue, err := evaluator.EvaluateFlag(context.Background(), flagID, tenantID, req)
							if err != nil {
								t.Errorf("Error evaluating tenant flag %s for tenant %s: %v", flagID, tenantID, err)
							}
							if actualValue != expectedValue {
								t.Errorf("Tenant flag %s for tenant %s: expected %v, got %v", flagID, tenantID, expectedValue, actualValue)
							}
						}
					}
				}

			} else {
				// Should not provide any services
				if len(providedServices) != 0 {
					t.Errorf("Expected 0 provided services, got %d", len(providedServices))
				}
			}
		})
	}
}

// TestFeatureFlagEvaluatorServiceDependencyResolution tests that external services take precedence
func TestFeatureFlagEvaluatorServiceDependencyResolution(t *testing.T) {
	// Create mock router
	mockRouter := &testRouter{routes: make(map[string]http.HandlerFunc)}

	// Create external feature flag evaluator
	externalEvaluator := NewFileBasedFeatureFlagEvaluator()
	externalEvaluator.SetFlag("external-flag", true)

	// Create mock application
	app := NewMockTenantApplication()

	// Create module
	module := NewModule()

	// Set configuration with feature flags enabled
	module.config = &ReverseProxyConfig{
		BackendServices: map[string]string{
			"test": "http://test:8080",
		},
		FeatureFlags: FeatureFlagsConfig{
			Enabled: true,
			GlobalFlags: map[string]bool{
				"internal-flag": true,
			},
		},
	}

	// Set router and external evaluator via constructor
	services := map[string]any{
		"router":               mockRouter,
		"featureFlagEvaluator": externalEvaluator,
	}
	constructedModule, err := module.Constructor()(app, services)
	if err != nil {
		t.Fatalf("Failed to construct module: %v", err)
	}
	module = constructedModule.(*ReverseProxyModule)

	// Set the app reference
	module.app = app

	// Start the module
	if err := module.Start(context.Background()); err != nil {
		t.Fatalf("Failed to start module: %v", err)
	}

	// Test that the external evaluator is used, not the internal one
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/test", nil)

	// The external flag should exist
	externalValue, err := module.featureFlagEvaluator.EvaluateFlag(context.Background(), "external-flag", "", req)
	if err != nil {
		t.Errorf("Error evaluating external flag: %v", err)
	}
	if !externalValue {
		t.Error("Expected external flag to be true")
	}

	// The internal flag should not exist (because we're using external evaluator)
	_, err = module.featureFlagEvaluator.EvaluateFlag(context.Background(), "internal-flag", "", req)
	if err == nil {
		t.Error("Expected internal flag to not exist when using external evaluator")
	}

	// The module should still provide the service (it's the external one)
	providedServices := module.ProvidesServices()
	if len(providedServices) != 1 {
		t.Errorf("Expected 1 provided service, got %d", len(providedServices))
		return
	}

	// Verify it's the same instance as the external evaluator
	if providedServices[0].Instance != externalEvaluator {
		t.Error("Expected provided service to be the same instance as external evaluator")
	}
}

// TestFeatureFlagEvaluatorConfigValidation tests configuration validation
func TestFeatureFlagEvaluatorConfigValidation(t *testing.T) {
	// Create mock router
	mockRouter := &testRouter{routes: make(map[string]http.HandlerFunc)}

	// Create mock application
	app := NewMockTenantApplication()

	// Create module
	module := NewModule()

	// Test with nil config (should not crash)
	module.config = nil

	// Set router via constructor
	services := map[string]any{
		"router": mockRouter,
	}
	constructedModule, err := module.Constructor()(app, services)
	if err != nil {
		t.Fatalf("Failed to construct module: %v", err)
	}
	module = constructedModule.(*ReverseProxyModule)

	// Set the app reference
	module.app = app

	// This should not crash even with nil config
	providedServices := module.ProvidesServices()
	if len(providedServices) != 0 {
		t.Errorf("Expected 0 provided services with nil config, got %d", len(providedServices))
	}
}

// TestServiceProviderInterface tests that the service properly implements the expected interface
func TestServiceProviderInterface(t *testing.T) {
	// Create the evaluator
	evaluator := NewFileBasedFeatureFlagEvaluator()

	// Test that it implements FeatureFlagEvaluator
	var _ FeatureFlagEvaluator = evaluator

	// Test using reflection (as the framework would)
	evaluatorType := reflect.TypeOf(evaluator)
	featureFlagInterface := reflect.TypeOf((*FeatureFlagEvaluator)(nil)).Elem()

	if !evaluatorType.Implements(featureFlagInterface) {
		t.Error("FileBasedFeatureFlagEvaluator does not implement FeatureFlagEvaluator interface")
	}
}
