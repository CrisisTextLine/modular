# Reverse Proxy Module

[![Go Reference](https://pkg.go.dev/badge/github.com/CrisisTextLine/modular/modules/reverseproxy.svg)](https://pkg.go.dev/github.com/CrisisTextLine/modular/modules/reverseproxy)

A module for the [Modular](https://github.com/CrisisTextLine/modular) framework that provides a flexible reverse proxy with advanced routing capabilities.

## Overview

The Reverse Proxy module functions as a versatile API gateway that can route requests to multiple backend services, combine responses, and support tenant-specific routing configurations. It's designed to be flexible, extensible, and easily configurable.

## Key Features

* **Multi-Backend Routing**: Route HTTP requests to any number of configurable backend services
* **Per-Backend Configuration**: Configure path rewriting and header rewriting for each backend service
* **Per-Endpoint Configuration**: Override backend configuration for specific endpoints within a backend
* **Feature Flag Support**: Control backend and route behavior using feature flags with optional alternatives
* **Hostname Handling**: Control how the Host header is handled (preserve original, use backend, or use custom)
* **Header Rewriting**: Add, modify, or remove headers before forwarding requests
* **Path Rewriting**: Transform request paths before forwarding to backends
* **Response Aggregation**: Combine responses from multiple backends using various strategies
* **Custom Response Transformers**: Create custom functions to transform and merge backend responses
* **Tenant Awareness**: Support for multi-tenant environments with tenant-specific routing
* **Pattern-Based Routing**: Direct requests to specific backends based on URL patterns
* **Custom Endpoint Mapping**: Define flexible mappings from frontend endpoints to backend services
* **Health Checking**: Continuous monitoring of backend service availability with DNS resolution and HTTP checks
* **Circuit Breaker**: Automatic failure detection and recovery with configurable thresholds
* **Response Caching**: Performance optimization with TTL-based caching
* **Metrics Collection**: Comprehensive metrics for monitoring and debugging

## Installation

```go
go get github.com/CrisisTextLine/modular/modules/reverseproxy@v1.0.0
```

## Usage

```go
package main

import (
	"github.com/CrisisTextLine/modular"
	"github.com/CrisisTextLine/modular/modules/chimux"
	"github.com/CrisisTextLine/modular/modules/reverseproxy"
	"log/slog"
	"os"
)

func main() {
	// Create a new application
	app := modular.NewStdApplication(
		modular.NewStdConfigProvider(&AppConfig{}),
		slog.New(slog.NewTextHandler(os.Stdout, nil)),
	)

	// Register required modules
	app.RegisterModule(chimux.NewChiMuxModule())
	
	// Register the reverseproxy module
	proxyModule, err := reverseproxy.NewModule()
	if err != nil {
		app.Logger().Error("Failed to create reverseproxy module", "error", err)
		os.Exit(1)
	}
	app.RegisterModule(proxyModule)

	// Run the application
	if err := app.Run(); err != nil {
		app.Logger().Error("Application error", "error", err)
		os.Exit(1)
	}
}
```

## Configuration

### Basic Configuration

```yaml
# config.yaml
reverseproxy:
  # Define your backend services
  backend_services:
    api: "http://api.example.com"
    auth: "http://auth.example.com"
    user: "http://user-service.example.com"
  
  # Set the default backend
  default_backend: "api"
  
  # Tenant-specific configuration
  tenant_id_header: "X-Tenant-ID"
  require_tenant_id: false
  
  # Health check configuration
  health_check:
    enabled: true
    interval: "30s"
    timeout: "5s"
    recent_request_threshold: "60s"
    expected_status_codes: [200, 204]
    health_endpoints:
      api: "/health"
      auth: "/api/health"
    backend_health_check_config:
      api:
        enabled: true
        interval: "15s"
        timeout: "3s"
        expected_status_codes: [200]
      auth:
        enabled: true
        endpoint: "/status"
        interval: "45s"
        timeout: "10s"
        expected_status_codes: [200, 201]

  # Per-backend configuration
  backend_configs:
    api:
      path_rewriting:
        strip_base_path: "/api/v1"
        base_path_rewrite: "/internal/api"
      header_rewriting:
        hostname_handling: "preserve_original"
        set_headers:
          X-API-Key: "secret-key"
          X-Service: "api"
      
      endpoints:
        users:
          pattern: "/users/*"
          path_rewriting:
            base_path_rewrite: "/internal/users"
          header_rewriting:
            hostname_handling: "use_custom"
            custom_hostname: "users.internal.com"
    
    auth:
      header_rewriting:
        hostname_handling: "use_backend"
        set_headers:
          X-Service: "auth"


  # Composite routes for response aggregation
  composite_routes:
    "/api/user/profile":
      pattern: "/api/user/profile"
      backends: ["user", "api"]
      strategy: "merge"
```

### Advanced Features

The module supports several advanced features:

1. **Custom Response Transformers**: Create custom functions to transform responses from multiple backends
2. **Custom Endpoint Mappings**: Define detailed mappings between frontend endpoints and backend services
3. **Tenant-Specific Routing**: Route requests to different backend URLs based on tenant ID
4. **Health Checking**: Continuous monitoring of backend service availability with configurable endpoints and intervals
5. **Circuit Breaker**: Automatic failure detection and recovery to prevent cascading failures
6. **Response Caching**: Performance optimization with TTL-based caching of responses
7. **Feature Flags**: Control backend and route behavior dynamically using feature flag evaluation

### Feature Flag Support

The reverse proxy module supports feature flags to control routing behavior dynamically. Feature flags can be used to:

- Enable/disable specific backends
- Route to alternative backends when features are disabled  
- Control composite route availability
- Support A/B testing and gradual rollouts
- Provide tenant-specific feature access

#### Feature Flag Configuration

```yaml
reverseproxy:
  # Backend configurations with feature flags
  backend_configs:
    api-v2:
      feature_flag_id: "api-v2-enabled"     # Feature flag to check
      alternative_backend: "api-v1"         # Fallback when disabled
    
    beta-features:
      feature_flag_id: "beta-features"
      alternative_backend: "stable-api"
  
  # Composite routes with feature flags
  composite_routes:
    "/api/enhanced":
      backends: ["api-v2", "analytics"]
      strategy: "merge"
      feature_flag_id: "enhanced-api"       # Feature flag for composite route
      alternative_backend: "api-v1"         # Single backend fallback
```

#### Feature Flag Evaluator Service

To use feature flags, register a `FeatureFlagEvaluator` service with your application:

```go
// Create feature flag evaluator (file-based example)
evaluator := reverseproxy.NewFileBasedFeatureFlagEvaluator()
evaluator.SetFlag("api-v2-enabled", true)
evaluator.SetTenantFlag("beta-tenant", "beta-features", true)

// Register as service
app.RegisterService("featureFlagEvaluator", evaluator)
```

The evaluator interface allows integration with external feature flag services like LaunchDarkly, Split.io, or custom implementations.

### Health Check Configuration

The reverseproxy module provides comprehensive health checking capabilities:

```yaml
health_check:
  enabled: true                    # Enable health checking
  interval: "30s"                  # Global check interval
  timeout: "5s"                    # Global check timeout
  recent_request_threshold: "60s"  # Skip checks if recent request within threshold
  expected_status_codes: [200, 204] # Global expected status codes
  
  # Custom health endpoints per backend
  health_endpoints:
    api: "/health"
    auth: "/api/health"
  
  # Per-backend health check configuration
  backend_health_check_config:
    api:
      enabled: true
      interval: "15s"              # Override global interval
      timeout: "3s"                # Override global timeout
      expected_status_codes: [200] # Override global status codes
    auth:
      enabled: true
      endpoint: "/status"          # Custom health endpoint
      interval: "45s"
      timeout: "10s"
      expected_status_codes: [200, 201]
```

**Health Check Features:**
- **DNS Resolution**: Verifies that backend hostnames resolve to IP addresses
- **HTTP Connectivity**: Tests HTTP connectivity to backends with configurable timeouts
- **Custom Endpoints**: Supports custom health check endpoints per backend
- **Smart Scheduling**: Skips health checks if recent requests have occurred
- **Per-Backend Configuration**: Allows fine-grained control over health check behavior
- **Status Monitoring**: Tracks health status, response times, and error details
- **Metrics Integration**: Exposes health status through metrics endpoints

1. **Per-Backend Configuration**: Configure path rewriting and header rewriting for each backend service
2. **Per-Endpoint Configuration**: Override backend configuration for specific endpoints
3. **Hostname Handling**: Control how the Host header is handled for each backend
4. **Header Rewriting**: Add, modify, or remove headers before forwarding requests
5. **Path Rewriting**: Transform request paths before forwarding to backends
6. **Custom Response Transformers**: Create custom functions to transform responses from multiple backends
7. **Custom Endpoint Mappings**: Define detailed mappings between frontend endpoints and backend services
8. **Tenant-Specific Routing**: Route requests to different backend URLs based on tenant ID

For detailed documentation and examples, see:
- [PATH_REWRITING_GUIDE.md](PATH_REWRITING_GUIDE.md) - Complete guide to path rewriting and header rewriting
- [PER_BACKEND_CONFIGURATION_GUIDE.md](PER_BACKEND_CONFIGURATION_GUIDE.md) - Per-backend and per-endpoint configuration
- [DOCUMENTATION.md](DOCUMENTATION.md) - General module documentation

## License

[MIT License](LICENSE)
