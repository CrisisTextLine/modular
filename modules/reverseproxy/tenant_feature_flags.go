package reverseproxy

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/CrisisTextLine/modular"
)

// TenantConfigFeatureFlagEvaluator implements a feature flag evaluator that loads tenant-specific
// configuration from files and provides graceful fallback.
type TenantConfigFeatureFlagEvaluator struct {
	tenantConfigs map[modular.TenantID]map[string]bool
	globalConfig  map[string]bool
	logger        *slog.Logger
}

// NewTenantConfigFeatureFlagEvaluator creates a new tenant config-based feature flag evaluator.
func NewTenantConfigFeatureFlagEvaluator(logger *slog.Logger) *TenantConfigFeatureFlagEvaluator {
	return &TenantConfigFeatureFlagEvaluator{
		tenantConfigs: make(map[modular.TenantID]map[string]bool),
		globalConfig:  make(map[string]bool),
		logger:        logger,
	}
}

// LoadTenantConfig loads feature flag configuration for a specific tenant.
func (t *TenantConfigFeatureFlagEvaluator) LoadTenantConfig(tenantID modular.TenantID, config map[string]bool) {
	t.tenantConfigs[tenantID] = config
	t.logger.DebugContext(context.Background(), "Loaded tenant feature flag config", "tenant", tenantID, "flags", len(config))
}

// LoadGlobalConfig loads global feature flag configuration.
func (t *TenantConfigFeatureFlagEvaluator) LoadGlobalConfig(config map[string]bool) {
	t.globalConfig = config
	t.logger.DebugContext(context.Background(), "Loaded global feature flag config", "flags", len(config))
}

// EvaluateFlag evaluates a feature flag using tenant configuration.
func (t *TenantConfigFeatureFlagEvaluator) EvaluateFlag(ctx context.Context, flagID string, tenantID modular.TenantID, req *http.Request) (bool, error) {
	// Check tenant-specific configuration first
	if tenantID != "" {
		if tenantConfig, exists := t.tenantConfigs[tenantID]; exists {
			if value, found := tenantConfig[flagID]; found {
				t.logger.DebugContext(ctx, "Feature flag found in tenant config",
					"flag", flagID,
					"tenant", tenantID,
					"value", value)
				return value, nil
			}
		}
	}

	// Fall back to global configuration
	if value, found := t.globalConfig[flagID]; found {
		t.logger.DebugContext(ctx, "Feature flag found in global config",
			"flag", flagID,
			"value", value)
		return value, nil
	}

	t.logger.DebugContext(ctx, "Feature flag not found in config, defaulting to false",
		"flag", flagID,
		"tenant", tenantID)
	return false, fmt.Errorf("feature flag %s not found: %w", flagID, ErrFeatureFlagNotFound)
}

// EvaluateFlagWithDefault evaluates a feature flag with a default value.
func (t *TenantConfigFeatureFlagEvaluator) EvaluateFlagWithDefault(ctx context.Context, flagID string, tenantID modular.TenantID, req *http.Request, defaultValue bool) bool {
	result, err := t.EvaluateFlag(ctx, flagID, tenantID, req)
	if err != nil {
		return defaultValue
	}
	return result
}
