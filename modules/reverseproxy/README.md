# Reverse Proxy Module

[![Go Reference](https://pkg.go.dev/badge/github.com/CrisisTextLine/modular/modules/reverseproxy.svg)](https://pkg.go.dev/github.com/CrisisTextLine/modular/modules/reverseproxy)

A module for the [Modular](https://github.com/CrisisTextLine/modular) framework that provides a flexible reverse proxy with advanced routing capabilities.

## Overview

The Reverse Proxy module functions as a versatile API gateway that can route requests to multiple backend services, combine responses, and support tenant-specific routing configurations. It's designed to be flexible, extensible, and easily configurable.

## Key Features

* **Multi-Backend Routing**: Route HTTP requests to any number of configurable backend services
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

For detailed documentation and examples, see the [DOCUMENTATION.md](DOCUMENTATION.md) file.

## License

[MIT License](LICENSE)
