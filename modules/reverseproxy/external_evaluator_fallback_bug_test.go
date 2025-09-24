package reverseproxy

import (
	"context"
	"net/http"
	"testing"

	"github.com/CrisisTextLine/modular"
	"github.com/stretchr/testify/mock"
)

// mockExternalEvaluatorReturnsErrNoDecision simulates an external evaluator (like LaunchDarkly)
// that is configured but returns ErrNoDecision due to initialization failures
type mockExternalEvaluatorReturnsErrNoDecision struct{}

func (m *mockExternalEvaluatorReturnsErrNoDecision) EvaluateFlag(ctx context.Context, flagID string, tenantID modular.TenantID, req *http.Request) (bool, error) {
	// Simulate external evaluator that's configured but not working (e.g., invalid SDK key)
	return false, ErrNoDecision
}

func (m *mockExternalEvaluatorReturnsErrNoDecision) EvaluateFlagWithDefault(ctx context.Context, flagID string, tenantID modular.TenantID, req *http.Request, defaultValue bool) bool {
	result, err := m.EvaluateFlag(ctx, flagID, tenantID, req)
	if err != nil {
		return defaultValue
	}
	return result
}

func (m *mockExternalEvaluatorReturnsErrNoDecision) Weight() int {
	return 50 // Higher priority than file evaluator (weight 1000)
}

// TestExternalEvaluatorFallbackBug reproduces the bug described in the issue
func TestExternalEvaluatorFallbackBug(t *testing.T) {
	// Create a mock application
	moduleApp := NewMockTenantApplication()

	// Configure reverseproxy with feature flags enabled and a flag set to true
	config := &ReverseProxyConfig{
		BackendServices: map[string]string{
			"primary":     "http://127.0.0.1:18080",
			"alternative": "http://127.0.0.1:18081",
		},
		FeatureFlags: FeatureFlagsConfig{
			Enabled: true,
			Flags: map[string]bool{
				"my-api": false, // This should route to alternative backend when external evaluator abstains
			},
		},
	}

	// Register the configuration
	moduleApp.RegisterConfigSection("reverseproxy", modular.NewStdConfigProvider(config))

	// Create the module
	module := NewModule()

	// Create a mock router and set up expected calls
	mockRouter := &MockRouter{}
	// Allow any HandleFunc calls
	mockRouter.On("HandleFunc", mock.Anything, mock.Anything).Return()

	// Create an external evaluator that returns ErrNoDecision (simulating misconfigured LaunchDarkly)
	externalEvaluator := &mockExternalEvaluatorReturnsErrNoDecision{}

	// Provide services via constructor - this is where the bug occurs
	services := map[string]any{
		"router":               mockRouter,
		"featureFlagEvaluator": externalEvaluator, // This bypasses the aggregator
	}

	constructedModule, err := module.Constructor()(moduleApp, services)
	if err != nil {
		t.Fatalf("Failed to construct module: %v", err)
	}
	module = constructedModule.(*ReverseProxyModule)

	// Set up the configuration first
	err = module.RegisterConfig(moduleApp)
	if err != nil {
		t.Fatalf("Failed to register config: %v", err)
	}

	// Initialize the module (this sets up the feature flag evaluator)
	if err := module.Init(moduleApp); err != nil {
		t.Fatalf("Failed to initialize module: %v", err)
	}

	// Start the module (this calls setupFeatureFlagEvaluation)
	if err := module.Start(context.Background()); err != nil {
		t.Fatalf("Failed to start module: %v", err)
	}

	// Test what the behavior SHOULD be (this will fail until we fix the bug)
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/test", nil)
	
	// We need to test the flag with the YAML value false to show the problem
	result := module.evaluateFeatureFlag("my-api", req)
	
	// First verify the setup: external evaluator returns ErrNoDecision
	if module.featureFlagEvaluator != nil {
		directResult, err := module.featureFlagEvaluator.EvaluateFlag(context.Background(), "my-api", "", req)
		t.Logf("Direct evaluator result: %v, error: %v", directResult, err)
		if err == nil || err != ErrNoDecision {
			t.Errorf("Expected external evaluator to return ErrNoDecision, got: %v", err)
		}
	}

	// The problem: evaluateFeatureFlag uses EvaluateFlagWithDefault(true) 
	// which returns true when external evaluator returns ErrNoDecision
	// It should instead fall back to file evaluator which has the YAML config
	t.Logf("evaluateFeatureFlag result: %v", result)
	t.Logf("featureFlagEvaluatorProvided: %v", module.featureFlagEvaluatorProvided)
	t.Logf("featureFlagEvaluator type: %T", module.featureFlagEvaluator)

	// This test will pass with the current buggy behavior but shows the problem
	if result {
		t.Logf("Current behavior: returns true (hard-coded default) - this is the bug")
	}

	// What SHOULD happen: fallback to file evaluator should return the YAML value
	// Let's manually test what the file evaluator would return
	// We need to access the app's registered service "featureFlagEvaluator.file"
	var fileEvaluator FeatureFlagEvaluator
	if err := module.app.GetService("featureFlagEvaluator.file", &fileEvaluator); err == nil {
		fileResult, fileErr := fileEvaluator.EvaluateFlag(context.Background(), "my-api", "", req)
		t.Logf("File evaluator would return: %v, error: %v", fileResult, fileErr)
		if fileErr == nil && fileResult != result {
			t.Errorf("BUG DETECTED: External evaluator fallback should return file evaluator result (%v), but returned hard-coded default (%v)", fileResult, result)
		}
	} else {
		t.Logf("File evaluator not registered: %v", err)
	}
}