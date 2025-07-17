# ReverseProxy Module - Hostname Forwarding and Path Rewriting

## Overview

The reverseproxy module has been enhanced with two key features:

1. **Hostname Forwarding Fix**: The module no longer passes the hostname forward to backend services, preserving the original request's Host header.
2. **Path Rewriting Support**: Comprehensive path rewriting capabilities including base path rewriting and per-endpoint path rewriting.

## Hostname Forwarding

### Before the Fix
```go
// Original request: Host: client.example.com
// Backend receives: Host: backend.internal.com (hostname forwarded)
```

### After the Fix
```go
// Original request: Host: client.example.com
// Backend receives: Host: client.example.com (hostname preserved)
```

This change ensures that backend services receive the original client's Host header, which is important for:
- Virtual hosting scenarios
- SSL certificate validation
- Application logic that depends on the original hostname

## Path Rewriting Configuration

The new `PathRewritingConfig` provides flexible path transformation options:

```yaml
reverseproxy:
  backend_services:
    api: "http://internal-api.example.com"
    legacy: "http://legacy-api.example.com"
  
  path_rewriting:
    # Strip base path from all requests
    strip_base_path: "/api/v1"
    
    # Rewrite base path for all requests
    base_path_rewrite: "/internal/api"
    
    # Per-endpoint rewriting rules
    endpoint_rewrites:
      users-v1:
        pattern: "/users/*"
        replacement: "/internal/users"
        backend: "api"  # optional: apply only to specific backend
      
      legacy-users:
        pattern: "/legacy/users/*"
        replacement: "/users"
        backend: "legacy"
```

### Base Path Operations

#### Strip Base Path
Removes a specified base path from all incoming requests:

```yaml
path_rewriting:
  strip_base_path: "/api/v1"
```

- Request: `/api/v1/users/123` → Backend: `/users/123`
- Request: `/api/v1/orders/456` → Backend: `/orders/456`

#### Base Path Rewrite
Prepends a new base path to all requests:

```yaml
path_rewriting:
  base_path_rewrite: "/internal/api"
```

- Request: `/users/123` → Backend: `/internal/api/users/123`
- Request: `/orders/456` → Backend: `/internal/api/orders/456`

#### Combined Strip and Rewrite
Both operations can be used together:

```yaml
path_rewriting:
  strip_base_path: "/api/v1"
  base_path_rewrite: "/internal/api"
```

- Request: `/api/v1/users/123` → Backend: `/internal/api/users/123`

### Per-Endpoint Rewriting

Define specific rewriting rules for different endpoint patterns:

```yaml
path_rewriting:
  endpoint_rewrites:
    users-endpoint:
      pattern: "/api/users/*"
      replacement: "/internal/users"
    
    orders-endpoint:
      pattern: "/api/orders/*"
      replacement: "/internal/orders"
```

#### Pattern Matching

- **Exact Match**: `/api/users` matches only `/api/users`
- **Wildcard Match**: `/api/users/*` matches `/api/users/123`, `/api/users/123/profile`, etc.
- **Star Match**: `/api/users*` matches `/api/users123`, `/api/users/123`, etc.

#### Rule Priority

When multiple rules could match a path, the first matching rule is applied.

### Tenant-Specific Path Rewriting

Path rewriting rules can be defined per tenant:

```yaml
# Global configuration
reverseproxy:
  path_rewriting:
    strip_base_path: "/api/v1"
    endpoint_rewrites:
      users:
        pattern: "/users/*"
        replacement: "/global/users"

# Tenant-specific configuration
tenants:
  tenant-123:
    reverseproxy:
      path_rewriting:
        strip_base_path: "/api/v2"  # Override global setting
        endpoint_rewrites:
          users:
            pattern: "/users/*"
            replacement: "/tenant/users"  # Override global rule
```

## Configuration Examples

### Basic Configuration
```yaml
reverseproxy:
  backend_services:
    api: "http://api.internal.com"
  default_backend: "api"
  
  path_rewriting:
    strip_base_path: "/api/v1"
```

### Advanced Configuration
```yaml
reverseproxy:
  backend_services:
    api: "http://api.internal.com"
    legacy: "http://legacy.internal.com"
  
  routes:
    "/api/*": "api"
    "/legacy/*": "legacy"
  
  path_rewriting:
    strip_base_path: "/public"
    base_path_rewrite: "/internal"
    
    endpoint_rewrites:
      api-users:
        pattern: "/api/users/*"
        replacement: "/v2/users"
        backend: "api"
      
      legacy-users:
        pattern: "/legacy/users/*"
        replacement: "/users"
        backend: "legacy"
      
      catch-all:
        pattern: "/*"
        replacement: "/default"
```

### Multi-Tenant Configuration
```yaml
reverseproxy:
  backend_services:
    api: "http://api.internal.com"
  
  tenant_id_header: "X-Tenant-ID"
  
  path_rewriting:
    strip_base_path: "/api/v1"
    endpoint_rewrites:
      users:
        pattern: "/users/*"
        replacement: "/global/users"

# Tenant overrides
tenants:
  premium-tenant:
    reverseproxy:
      backend_services:
        api: "http://premium-api.internal.com"
      
      path_rewriting:
        strip_base_path: "/api/v2"
        endpoint_rewrites:
          users:
            pattern: "/users/*"
            replacement: "/premium/users"
```

## Usage Examples

### Go Configuration
```go
config := &reverseproxy.ReverseProxyConfig{
    BackendServices: map[string]string{
        "api": "http://api.internal.com",
    },
    DefaultBackend: "api",
    
    PathRewriting: reverseproxy.PathRewritingConfig{
        StripBasePath: "/api/v1",
        BasePathRewrite: "/internal/api",
        
        EndpointRewrites: map[string]reverseproxy.EndpointRewriteRule{
            "users": {
                Pattern: "/users/*",
                Replacement: "/internal/users",
            },
        },
    },
}
```

### Testing the Configuration

The module includes comprehensive test coverage for both hostname forwarding and path rewriting. Key test scenarios include:

1. **Hostname Forwarding Tests**: Verify that the original Host header is preserved
2. **Base Path Rewriting Tests**: Test stripping and rewriting of base paths
3. **Endpoint Rewriting Tests**: Test pattern matching and replacement logic
4. **Tenant-Specific Tests**: Verify tenant-specific configurations work correctly
5. **Edge Cases**: Handle nil configurations, empty paths, multiple slashes, etc.

## Migration Guide

### From Previous Versions

If you were relying on the hostname forwarding behavior, you may need to update your backend services to handle the original Host header instead of the backend's host.

### New Configuration Options

The new path rewriting configuration is optional and backward compatible. Existing configurations will continue to work unchanged.

## Benefits

1. **Better Security**: Preserving original Host headers improves security for virtual hosting scenarios
2. **Flexible Path Management**: Support for complex path transformations without changing backend APIs
3. **Multi-Tenant Support**: Tenant-specific path rewriting rules enable customized routing per tenant
4. **Backward Compatibility**: All existing configurations continue to work unchanged