# Feature Flag Migration Guide: Aggregator Pattern

This guide helps you migrate your feature flag implementation to use the new aggregator pattern introduced in the reverse proxy module. The new system allows multiple feature flag evaluators to work together with priority-based ordering.

## Overview of Changes

The feature flag system has been enhanced with:

1. **Aggregator Pattern**: Multiple feature flag evaluators can now be registered and coordinated
2. **Weight-Based Priority**: Evaluators are called in order based on their priority weights
3. **Enhanced Error Handling**: Special sentinel errors control evaluation flow
4. **Service Naming Convention**: New naming pattern for registering multiple evaluators

## What's New

### 1. Multiple Evaluators Support

**Before**: Only one feature flag evaluator could be active at a time.

**After**: Multiple evaluators can be registered and work together, with the aggregator coordinating their evaluation.

### 2. Weight-Based Priority System

Evaluators now support priority weights:
- **Lower weight = Higher priority** (evaluated first)
- **Default weight**: 100 for evaluators that don't specify a weight
- **Built-in file evaluator**: Weight 1000 (lowest priority, fallback)

### 3. New Service Naming Convention

**Before**: External evaluators registered as `"featureFlagEvaluator"`

**After**: Use the pattern `"featureFlagEvaluator.<name>"` for external evaluators:
- `"featureFlagEvaluator.file"` - Built-in file-based evaluator (automatically registered)
- `"featureFlagEvaluator.remote"` - Remote feature flag service
- `"featureFlagEvaluator.rules"` - Rules-based evaluator
- `"featureFlagEvaluator"` - The aggregator that coordinates all evaluators

## Migration Steps

### Step 1: Update External Evaluator Registration

If you were registering a custom evaluator as `"featureFlagEvaluator"`, you need to change the service name:

**Before**:
```go
// This will conflict with the new aggregator
app.RegisterService("featureFlagEvaluator", myCustomEvaluator)
```

**After**:
```go
// Use a descriptive name with the prefix pattern
app.RegisterService("featureFlagEvaluator.remote", myCustomEvaluator)
```

### Step 2: Implement WeightedEvaluator (Optional)

If you want to control the priority of your evaluator, implement the `WeightedEvaluator` interface:

```go
type MyCustomEvaluator struct {
    // Your existing implementation
}

// Add the Weight method to control priority
func (e *MyCustomEvaluator) Weight() int {
    return 10 // High priority (lower number = higher priority)
}

// Your existing FeatureFlagEvaluator methods
func (e *MyCustomEvaluator) EvaluateFlag(ctx context.Context, flagID string, tenantID modular.TenantID, req *http.Request) (bool, error) {
    // Your implementation
    if shouldDeferToNext {
        return false, reverseproxy.ErrNoDecision // Continue to next evaluator
    }
    return true, nil // Return decision and stop evaluation chain
}
```

### Step 3: Use Enhanced Error Handling (Optional)

The new system supports special sentinel errors for better control:

```go
import "github.com/CrisisTextLine/modular/modules/reverseproxy"

func (e *MyCustomEvaluator) EvaluateFlag(ctx context.Context, flagID string, tenantID modular.TenantID, req *http.Request) (bool, error) {
    // Check if you can make a decision
    if !e.canEvaluate(flagID) {
        return false, reverseproxy.ErrNoDecision // Let next evaluator try
    }
    
    // If there's a fatal error that should stop all evaluation
    if criticalError := e.checkCriticalError(); criticalError != nil {
        return false, reverseproxy.ErrEvaluatorFatal // Stop evaluation chain
    }
    
    // Make your decision
    decision := e.makeDecision(flagID, tenantID, req)
    return decision, nil // Return decision and stop evaluation
}
```

## Examples

### Example 1: Simple External Evaluator

```go
type RemoteEvaluator struct {
    client *http.Client
    baseURL string
}

// Implement WeightedEvaluator for high priority
func (r *RemoteEvaluator) Weight() int {
    return 50 // Higher priority than default (100) but lower than critical evaluators
}

func (r *RemoteEvaluator) EvaluateFlag(ctx context.Context, flagID string, tenantID modular.TenantID, req *http.Request) (bool, error) {
    // Try to get flag from remote service
    enabled, err := r.checkRemoteFlag(ctx, flagID, string(tenantID))
    if err != nil {
        // Let other evaluators try if remote service is unavailable
        return false, reverseproxy.ErrNoDecision
    }
    return enabled, nil // Return decision
}

func (r *RemoteEvaluator) EvaluateFlagWithDefault(ctx context.Context, flagID string, tenantID modular.TenantID, req *http.Request, defaultValue bool) bool {
    enabled, err := r.EvaluateFlag(ctx, flagID, tenantID, req)
    if err != nil {
        return defaultValue
    }
    return enabled
}

// Register the evaluator
app.RegisterService("featureFlagEvaluator.remote", &RemoteEvaluator{
    client: &http.Client{Timeout: 5 * time.Second},
    baseURL: "https://flags.example.com",
})
```

### Example 2: Tenant-Specific Rules Evaluator

```go
type TenantRulesEvaluator struct {
    rules map[string]map[string]bool // tenant -> flag -> enabled
}

func (t *TenantRulesEvaluator) Weight() int {
    return 25 // Very high priority for tenant-specific rules
}

func (t *TenantRulesEvaluator) EvaluateFlag(ctx context.Context, flagID string, tenantID modular.TenantID, req *http.Request) (bool, error) {
    tenantRules, exists := t.rules[string(tenantID)]
    if !exists {
        return false, reverseproxy.ErrNoDecision // No rules for this tenant
    }
    
    if enabled, exists := tenantRules[flagID]; exists {
        return enabled, nil // Return tenant-specific decision
    }
    
    return false, reverseproxy.ErrNoDecision // No rule for this flag
}

func (t *TenantRulesEvaluator) EvaluateFlagWithDefault(ctx context.Context, flagID string, tenantID modular.TenantID, req *http.Request, defaultValue bool) bool {
    enabled, err := t.EvaluateFlag(ctx, flagID, tenantID, req)
    if err != nil {
        return defaultValue
    }
    return enabled
}

// Register with high priority
app.RegisterService("featureFlagEvaluator.rules", &TenantRulesEvaluator{
    rules: map[string]map[string]bool{
        "tenant1": {"beta-feature": true},
        "tenant2": {"beta-feature": false},
    },
})
```

## Evaluation Flow

With multiple evaluators registered, the flow works as follows:

1. **Discovery**: Aggregator finds all services matching `"featureFlagEvaluator.*"`
2. **Ordering**: Evaluators are sorted by weight (ascending - lower weight = higher priority)
3. **Evaluation**: Each evaluator is called in order until one returns a decision:
   - `(decision, nil)` → Return the decision and stop
   - `(_, ErrNoDecision)` → Continue to the next evaluator
   - `(_, ErrEvaluatorFatal)` → Stop evaluation and return error
   - `(_, other error)` → Log warning and continue to next evaluator

### Example Evaluation Order

With these evaluators registered:
- `featureFlagEvaluator.rules` (weight: 25)
- `featureFlagEvaluator.remote` (weight: 50) 
- `featureFlagEvaluator.cache` (weight: 75)
- `featureFlagEvaluator.file` (weight: 1000, built-in)

**Evaluation order**: rules → remote → cache → file

## Backwards Compatibility

The new system maintains backwards compatibility:

1. **Existing Code**: If you don't register any external evaluators, the built-in file evaluator works as before
2. **Service Dependencies**: Modules depending on `"featureFlagEvaluator"` continue to work (they get the aggregator)
3. **Configuration**: All existing feature flag configuration continues to work unchanged

## Troubleshooting

### Issue: "Multiple evaluators conflict"

**Cause**: You registered an evaluator as `"featureFlagEvaluator"` which conflicts with the aggregator.

**Solution**: Rename your service to `"featureFlagEvaluator.<name>"`:
```go
// Change this:
app.RegisterService("featureFlagEvaluator", evaluator)

// To this:
app.RegisterService("featureFlagEvaluator.myservice", evaluator)
```

### Issue: "Evaluator not being called"

**Cause**: Your evaluator has a high weight (low priority) and earlier evaluators are returning decisions.

**Solution**: 
1. Lower your evaluator's weight for higher priority
2. Make other evaluators return `ErrNoDecision` when appropriate
3. Check evaluation logs to see the order

### Issue: "Evaluation stops unexpectedly"

**Cause**: An evaluator returned `ErrEvaluatorFatal` or a decision instead of `ErrNoDecision`.

**Solution**: Review your error handling:
- Use `ErrNoDecision` when you can't make a decision
- Use `ErrEvaluatorFatal` only for critical errors that should stop evaluation
- Return `(decision, nil)` only when you have a definitive answer

## Need Help?

If you encounter issues during migration:

1. Check the logs for evaluation order and errors
2. Verify your service naming follows the `"featureFlagEvaluator.<name>"` pattern  
3. Review your `Weight()` implementation if using `WeightedEvaluator`
4. Test with a simple evaluator first, then add complexity

The file-based evaluator (weight: 1000) always acts as the final fallback, so your system will continue to work even if external evaluators have issues.