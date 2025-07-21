# Feature Flag Proxy Example

This example demonstrates how to use feature flags to control routing behavior in the reverse proxy module.

## Overview

The example sets up:
- A reverse proxy with feature flag-controlled backends
- Multiple backend servers to demonstrate different routing scenarios
- Tenant-aware feature flags
- Composite routes with feature flag controls

## Feature Flags Configured

1. **`beta-feature`** (globally disabled, enabled for "beta-tenant"):
   - Controls access to the default backend
   - Falls back to alternative backend when disabled

2. **`new-backend`** (globally enabled):
   - Controls access to the new-feature backend  
   - Falls back to default backend when disabled

3. **`composite-route`** (globally enabled):
   - Controls access to the composite route that combines multiple backends
   - Falls back to default backend when disabled

## Backend Services

- **Default Backend** (port 9001): Main backend service
- **Alternative Backend** (port 9002): Fallback when feature flags are disabled
- **New Feature Backend** (port 9003): New service controlled by feature flag
- **API Backend** (port 9004): Used in composite routes
- **Beta Backend** (port 9005): Special backend for beta tenant

## Running the Example

1. Start the application:
   ```bash
   go run main.go
   ```

2. The application will start on port 8080 with backends on ports 9001-9005

## Testing Feature Flags

### Test beta-feature flag (globally disabled)

```bash
# Normal user - should get alternative backend (feature disabled)
curl http://localhost:8080/api/beta

# Beta tenant - should get default backend (feature enabled for this tenant)
curl -H "X-Tenant-ID: beta-tenant" http://localhost:8080/api/beta
```

### Test new-backend flag (globally enabled)

```bash
# Should get new-feature backend (feature enabled)
curl http://localhost:8080/api/new
```

### Test composite route flag

```bash
# Should get composite response from multiple backends (feature enabled)
curl http://localhost:8080/api/composite
```

### Test tenant-specific routing

```bash
# Beta tenant gets routed to their specific backend
curl -H "X-Tenant-ID: beta-tenant" http://localhost:8080/
```

## Configuration

The feature flags are configured in code in this example, but in a real application they would typically be:
- Loaded from a configuration file
- Retrieved from a feature flag service (LaunchDarkly, Split.io, etc.)
- Stored in a database

## Expected Responses

Each backend returns JSON with information about which backend served the request, making it easy to verify feature flag behavior:

```json
{
  "backend": "alternative",
  "path": "/api/beta", 
  "method": "GET",
  "feature": "fallback"
}
```

## Architecture

The feature flag system works by:
1. Registering a `FeatureFlagEvaluator` service with the application
2. Configuring feature flag IDs in backend and route configurations
3. The reverse proxy evaluates feature flags on each request
4. Routes are dynamically switched based on feature flag values
5. Tenant-specific overrides are supported for multi-tenant scenarios

This allows for:
- A/B testing new backends
- Gradual rollouts of new features
- Tenant-specific feature access
- Fallback behavior when features are disabled