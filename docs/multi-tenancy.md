# Multi-Tenancy Support

The Modular framework provides comprehensive support for building multi-tenant applications with isolated configurations, tenant-aware modules, and tenant-specific services.

## Table of Contents

- [Overview](#overview)
- [Tenant Contexts](#tenant-contexts)
- [Tenant Service](#tenant-service)
- [Tenant-Aware Modules](#tenant-aware-modules)
- [Tenant-Aware Configuration](#tenant-aware-configuration)
- [Tenant Configuration Loading](#tenant-configuration-loading)
- [Best Practices](#best-practices)

## Overview

Multi-tenancy allows a single application instance to serve multiple tenants (customers, organizations, etc.) with:

- **Tenant Isolation**: Separate configurations and data per tenant
- **Tenant Context**: Tracking the current tenant throughout request processing
- **Tenant-Aware Modules**: Modules that respond to tenant lifecycle events
- **Tenant Configuration**: Per-tenant configuration overrides
- **Tenant Services**: Tenant-specific service instances

## Tenant Contexts

Tenant contexts allow operations to be performed in the context of a specific tenant.

### Creating Tenant Contexts

```go
// Create a tenant context
tenantID := modular.TenantID("tenant1")
ctx := modular.NewTenantContext(context.Background(), tenantID)

// Extract tenant ID from a context
if tid, ok := modular.GetTenantIDFromContext(ctx); ok {
    fmt.Println("Current tenant:", tid)
}
```

### Using Tenant Contexts with Application

```go
// Create a tenant-specific application context
tenantCtx, err := app.WithTenant(tenantID)
if err != nil {
    log.Fatal("Failed to create tenant context:", err)
}

// Use the tenant context in your handlers
func handleRequest(w http.ResponseWriter, r *http.Request) {
    // Extract tenant ID from request header
    tenantID := r.Header.Get("X-Tenant-ID")
    
    // Create tenant context
    ctx := modular.NewTenantContext(r.Context(), modular.TenantID(tenantID))
    
    // Pass context to services
    result := myService.ProcessRequest(ctx, data)
}
```

## Tenant Service

The TenantService interface defines operations for managing tenants:

```go
type TenantService interface {
    // Get tenant-specific configuration
    GetTenantConfig(tenantID TenantID, section string) (ConfigProvider, error)
    
    // Get all registered tenant IDs
    GetTenants() []TenantID
    
    // Register a new tenant with configurations
    RegisterTenant(tenantID TenantID, configs map[string]ConfigProvider) error
    
    // Register a module as tenant-aware
    RegisterTenantAwareModule(module TenantAwareModule) error
}
```

### Standard Tenant Service

```go
// Create a standard tenant service
tenantService := modular.NewStandardTenantService(logger)

// Register it as a service
app.RegisterService("tenantService", tenantService)
```

### Registering Tenants

```go
// Register a tenant with specific configurations
tenantService.RegisterTenant("tenant1", map[string]modular.ConfigProvider{
    "database": modular.NewStdConfigProvider(&database.Config{
        Host: "tenant1-db.example.com",
    }),
    "cache": modular.NewStdConfigProvider(&cache.Config{
        Prefix: "tenant1:",
    }),
})
```

## Tenant-Aware Modules

Modules can implement the `TenantAwareModule` interface to respond to tenant lifecycle events:

```go
type TenantAwareModule interface {
    Module
    OnTenantRegistered(tenantID TenantID)
    OnTenantRemoved(tenantID TenantID)
}
```

### Implementation Example

```go
type MultiTenantModule struct {
    tenantData map[modular.TenantID]*TenantData
    mutex      sync.RWMutex
}

func (m *MultiTenantModule) OnTenantRegistered(tenantID modular.TenantID) {
    m.mutex.Lock()
    defer m.mutex.Unlock()
    
    // Initialize resources for this tenant
    m.tenantData[tenantID] = &TenantData{
        Cache:       cache.New(),
        Connections: make(map[string]*Connection),
        Initialized: time.Now(),
    }
    
    m.logger.Info("Tenant registered", "tenant", tenantID)
}

func (m *MultiTenantModule) OnTenantRemoved(tenantID modular.TenantID) {
    m.mutex.Lock()
    defer m.mutex.Unlock()
    
    // Clean up tenant resources
    if resource, exists := m.tenantData[tenantID]; exists {
        resource.Cache.Close()
        for _, conn := range resource.Connections {
            conn.Close()
        }
        delete(m.tenantData, tenantID)
    }
    
    m.logger.Info("Tenant removed", "tenant", tenantID)
}
```

### Accessing Tenant Data

```go
func (m *MultiTenantModule) ProcessRequest(ctx context.Context, data interface{}) error {
    // Get tenant ID from context
    tenantID, ok := modular.GetTenantIDFromContext(ctx)
    if !ok {
        return errors.New("no tenant context")
    }
    
    // Get tenant-specific data
    m.mutex.RLock()
    tenantData, exists := m.tenantData[tenantID]
    m.mutex.RUnlock()
    
    if !exists {
        return fmt.Errorf("tenant %s not found", tenantID)
    }
    
    // Use tenant-specific resources
    return tenantData.Cache.Set(key, value)
}
```

## Tenant-Aware Configuration

Tenant-aware configuration allows different settings per tenant while maintaining a default configuration.

### Creating Tenant-Aware Config

```go
func (m *MultiTenantModule) RegisterConfig(app modular.Application) error {
    // Default configuration
    defaultConfig := &MyConfig{
        Setting:  "default",
        Timeout:  30,
        MaxRetry: 3,
    }
    
    // Get tenant service
    var tenantService modular.TenantService
    if err := app.GetService("tenantService", &tenantService); err != nil {
        return err
    }
    
    // Create tenant-aware config provider
    tenantAwareConfig := modular.NewTenantAwareConfig(
        modular.NewStdConfigProvider(defaultConfig),
        tenantService,
        "mymodule",
    )
    
    app.RegisterConfigSection("mymodule", tenantAwareConfig)
    return nil
}
```

### Using Tenant-Aware Configuration

```go
func (m *MultiTenantModule) ProcessRequestWithTenant(ctx context.Context) {
    // Get config specific to the tenant in the context
    config, ok := m.config.(*modular.TenantAwareConfig)
    if !ok {
        // Handle non-tenant-aware config
        return
    }
    
    // Get tenant-specific configuration
    myConfig := config.GetConfigWithContext(ctx).(*MyConfig)
    
    // Use tenant-specific settings
    fmt.Println("Tenant setting:", myConfig.Setting)
    fmt.Println("Tenant timeout:", myConfig.Timeout)
}
```

### Configuration Inheritance

Tenant configurations inherit from the default configuration and override specific values:

```go
// Default config
defaultConfig := &MyConfig{
    Setting:  "default",
    Timeout:  30,
    MaxRetry: 3,
    Features: map[string]bool{
        "feature1": true,
        "feature2": false,
    },
}

// Tenant1 config (overrides only what's different)
tenant1Config := &MyConfig{
    Setting: "tenant1-specific",
    Features: map[string]bool{
        "feature2": true, // Override
    },
}

// Tenant1 will get:
// - Setting: "tenant1-specific" (overridden)
// - Timeout: 30 (inherited)
// - MaxRetry: 3 (inherited)
// - Features: {feature1: true, feature2: true} (merged)
```

## Tenant Configuration Loading

Modular provides utilities for loading tenant configurations from files.

### File-Based Tenant Config Loader

```go
import "regexp"

// Set up file-based tenant config loader
configLoader := modular.NewFileBasedTenantConfigLoader(modular.TenantConfigParams{
    ConfigNameRegex: regexp.MustCompile(`^tenant-[\w-]+\.(json|yaml)$`),
    ConfigDir:       "./configs/tenants",
    ConfigFeeders:   []modular.Feeder{},
})

// Register the loader as a service
app.RegisterService("tenantConfigLoader", configLoader)
```

### Directory Structure

```
configs/
├── app-config.yaml          # Main application config
└── tenants/
    ├── tenant-acme.yaml     # Acme Corp tenant config
    ├── tenant-globex.yaml   # Globex Inc tenant config
    └── tenant-initech.yaml  # Initech tenant config
```

### Tenant Configuration Files

**Example: `configs/tenants/tenant-acme.yaml`**

```yaml
database:
  host: "acme-db.example.com"
  name: "acme_production"
  
cache:
  prefix: "acme:"
  ttl: 3600
  
features:
  premium: true
  beta: true
```

### Automatic Tenant Discovery

```go
// The loader automatically discovers and loads all tenant configs
// matching the regex pattern when the application starts

func (m *MyModule) Init(app modular.Application) error {
    var loader modular.TenantConfigLoader
    if err := app.GetService("tenantConfigLoader", &loader); err != nil {
        return err
    }
    
    // Load all tenant configurations
    tenants, err := loader.LoadTenantConfigurations()
    if err != nil {
        return err
    }
    
    // Tenants are now registered with the tenant service
    for _, tenantID := range tenants {
        m.logger.Info("Loaded tenant", "id", tenantID)
    }
    
    return nil
}
```

## Best Practices

### 1. Always Use Tenant Contexts

Pass tenant context through the entire request chain:

```go
func handleRequest(w http.ResponseWriter, r *http.Request) {
    // Extract tenant ID from request
    tenantID := extractTenantID(r)
    
    // Create tenant context
    ctx := modular.NewTenantContext(r.Context(), tenantID)
    
    // Pass to all downstream operations
    result, err := service.Process(ctx, data)
}
```

### 2. Implement Proper Tenant Isolation

Ensure tenant data never leaks between tenants:

```go
type TenantData struct {
    cache       *cache.Cache
    connections map[string]*Connection
    mu          sync.RWMutex
}

func (m *Module) getTenantData(tenantID modular.TenantID) (*TenantData, error) {
    m.mu.RLock()
    defer m.mu.RUnlock()
    
    data, exists := m.tenantData[tenantID]
    if !exists {
        return nil, fmt.Errorf("tenant not found: %s", tenantID)
    }
    
    return data, nil
}
```

### 3. Resource Cleanup

Always clean up tenant resources when tenants are removed:

```go
func (m *Module) OnTenantRemoved(tenantID modular.TenantID) {
    m.mu.Lock()
    defer m.mu.Unlock()
    
    if data, exists := m.tenantData[tenantID]; exists {
        // Close all connections
        for _, conn := range data.connections {
            conn.Close()
        }
        
        // Clear cache
        data.cache.Flush()
        
        // Remove from map
        delete(m.tenantData, tenantID)
    }
}
```

### 4. Tenant Identification

Use consistent tenant identification:

```go
// Extract from header
func extractTenantID(r *http.Request) modular.TenantID {
    return modular.TenantID(r.Header.Get("X-Tenant-ID"))
}

// Extract from subdomain
func extractTenantIDFromSubdomain(r *http.Request) modular.TenantID {
    host := r.Host
    parts := strings.Split(host, ".")
    if len(parts) > 0 {
        return modular.TenantID(parts[0])
    }
    return ""
}

// Extract from JWT token
func extractTenantIDFromToken(r *http.Request) modular.TenantID {
    token := extractJWT(r)
    claims := parseJWT(token)
    return modular.TenantID(claims["tenant_id"].(string))
}
```

### 5. Configuration Validation

Validate tenant configurations:

```go
type TenantConfig struct {
    TenantID string `yaml:"tenant_id" required:"true"`
    Database struct {
        Host string `yaml:"host" required:"true"`
        Name string `yaml:"name" required:"true"`
    } `yaml:"database"`
}

func (c *TenantConfig) Validate() error {
    if c.TenantID == "" {
        return errors.New("tenant_id is required")
    }
    if c.Database.Host == "" {
        return errors.New("database.host is required")
    }
    return nil
}
```

### 6. Monitoring and Logging

Include tenant information in logs:

```go
func (m *Module) Process(ctx context.Context, data interface{}) error {
    tenantID, _ := modular.GetTenantIDFromContext(ctx)
    
    m.logger.Info("Processing request",
        "tenant", tenantID,
        "operation", "process",
        "data_size", len(data),
    )
    
    // Process...
}
```

### 7. Performance Considerations

Cache tenant configurations to avoid repeated lookups:

```go
type Module struct {
    configCache map[modular.TenantID]*Config
    cacheMu     sync.RWMutex
}

func (m *Module) getConfig(ctx context.Context) *Config {
    tenantID, ok := modular.GetTenantIDFromContext(ctx)
    if !ok {
        return m.defaultConfig
    }
    
    // Check cache first
    m.cacheMu.RLock()
    if config, exists := m.configCache[tenantID]; exists {
        m.cacheMu.RUnlock()
        return config
    }
    m.cacheMu.RUnlock()
    
    // Load and cache
    config := m.loadTenantConfig(tenantID)
    
    m.cacheMu.Lock()
    m.configCache[tenantID] = config
    m.cacheMu.Unlock()
    
    return config
}
```

### 8. Testing Multi-Tenant Code

Test with multiple tenants:

```go
func TestMultiTenant(t *testing.T) {
    // Create tenant service
    tenantService := modular.NewStandardTenantService(logger)
    
    // Register multiple tenants
    tenantService.RegisterTenant("tenant1", configs1)
    tenantService.RegisterTenant("tenant2", configs2)
    
    // Test with tenant1 context
    ctx1 := modular.NewTenantContext(context.Background(), "tenant1")
    result1 := module.Process(ctx1, data)
    
    // Test with tenant2 context
    ctx2 := modular.NewTenantContext(context.Background(), "tenant2")
    result2 := module.Process(ctx2, data)
    
    // Verify isolation
    assert.NotEqual(t, result1, result2)
}
```

## Example: Complete Multi-Tenant Module

```go
package mymodule

import (
    "context"
    "sync"
    
    "github.com/CrisisTextLine/modular"
)

type Module struct {
    config      *modular.TenantAwareConfig
    tenantData  map[modular.TenantID]*TenantData
    mu          sync.RWMutex
    logger      *slog.Logger
}

type TenantData struct {
    cache       *Cache
    connections map[string]*Connection
}

type Config struct {
    Setting string `yaml:"setting" default:"default"`
    Timeout int    `yaml:"timeout" default:"30"`
}

func (m *Module) Name() string {
    return "mymodule"
}

func (m *Module) RegisterConfig(app modular.Application) error {
    defaultConfig := &Config{
        Setting: "default",
        Timeout: 30,
    }
    
    var tenantService modular.TenantService
    if err := app.GetService("tenantService", &tenantService); err != nil {
        return err
    }
    
    m.config = modular.NewTenantAwareConfig(
        modular.NewStdConfigProvider(defaultConfig),
        tenantService,
        m.Name(),
    )
    
    app.RegisterConfigSection(m.Name(), m.config)
    return nil
}

func (m *Module) Init(app modular.Application) error {
    m.tenantData = make(map[modular.TenantID]*TenantData)
    m.logger = app.Logger()
    return nil
}

func (m *Module) OnTenantRegistered(tenantID modular.TenantID) {
    m.mu.Lock()
    defer m.mu.Unlock()
    
    m.tenantData[tenantID] = &TenantData{
        cache:       NewCache(),
        connections: make(map[string]*Connection),
    }
    
    m.logger.Info("Tenant registered", "tenant", tenantID)
}

func (m *Module) OnTenantRemoved(tenantID modular.TenantID) {
    m.mu.Lock()
    defer m.mu.Unlock()
    
    if data, exists := m.tenantData[tenantID]; exists {
        data.cache.Close()
        delete(m.tenantData, tenantID)
    }
    
    m.logger.Info("Tenant removed", "tenant", tenantID)
}

func (m *Module) Process(ctx context.Context, input interface{}) error {
    // Get tenant configuration
    config := m.config.GetConfigWithContext(ctx).(*Config)
    
    // Get tenant ID
    tenantID, ok := modular.GetTenantIDFromContext(ctx)
    if !ok {
        return errors.New("no tenant context")
    }
    
    // Get tenant data
    m.mu.RLock()
    data, exists := m.tenantData[tenantID]
    m.mu.RUnlock()
    
    if !exists {
        return fmt.Errorf("tenant not found: %s", tenantID)
    }
    
    // Use tenant-specific config and data
    m.logger.Info("Processing",
        "tenant", tenantID,
        "setting", config.Setting,
        "timeout", config.Timeout,
    )
    
    // Process with tenant resources...
    return nil
}
```

## See Also

- [Configuration System](configuration.md) - Tenant-aware configuration
- [Module Lifecycle](module-lifecycle.md) - Tenant lifecycle events
- [Base Configuration](base-config.md) - Combining with base configs
