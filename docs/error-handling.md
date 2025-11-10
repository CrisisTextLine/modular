# Error Handling

The Modular framework defines common error types and follows Go's error wrapping conventions to provide context and enable error inspection.

## Table of Contents

- [Common Error Types](#common-error-types)
- [Error Wrapping](#error-wrapping)
- [Error Inspection](#error-inspection)
- [Best Practices](#best-practices)

## Common Error Types

Modular defines common error types to help with error handling:

### Service Errors

```go
// ErrServiceAlreadyRegistered - Service with this name already exists
modular.ErrServiceAlreadyRegistered

// ErrServiceNotFound - Requested service not found in registry
modular.ErrServiceNotFound

// ErrServiceIncompatible - Service type doesn't match expected type
modular.ErrServiceIncompatible
```

**Example:**

```go
err := app.RegisterService("database", dbConnection)
if errors.Is(err, modular.ErrServiceAlreadyRegistered) {
    // Handle duplicate service registration
    log.Warn("Service already registered, skipping")
}
```

### Config Errors

```go
// ErrConfigSectionNotFound - Configuration section not found
modular.ErrConfigSectionNotFound

// ErrConfigValidationFailed - Configuration validation failed
modular.ErrConfigValidationFailed
```

**Example:**

```go
provider := app.GetConfigProvider("mymodule")
if provider == nil {
    return modular.ErrConfigSectionNotFound
}

if err := config.Validate(); err != nil {
    return fmt.Errorf("%w: %v", modular.ErrConfigValidationFailed, err)
}
```

### Dependency Errors

```go
// ErrCircularDependency - Circular dependency detected in module dependencies
modular.ErrCircularDependency

// ErrModuleDependencyMissing - Required module dependency not found
modular.ErrModuleDependencyMissing
```

**Example:**

```go
err := app.Init()
if errors.Is(err, modular.ErrCircularDependency) {
    log.Error("Circular dependency detected in modules")
    // Handle circular dependency
}
```

### Tenant Errors

```go
// ErrTenantNotFound - Tenant ID not found
modular.ErrTenantNotFound

// ErrTenantConfigNotFound - Tenant configuration not found
modular.ErrTenantConfigNotFound
```

**Example:**

```go
config, err := tenantService.GetTenantConfig(tenantID, "mymodule")
if errors.Is(err, modular.ErrTenantNotFound) {
    // Handle missing tenant
    log.Warn("Tenant not found", "tenant", tenantID)
}
```

## Error Wrapping

Modular follows Go's error wrapping conventions to provide context:

### Basic Wrapping

```go
if err := doSomething(); err != nil {
    return fmt.Errorf("module '%s' failed: %w", m.Name(), err)
}
```

### Multi-Level Wrapping

```go
func (m *MyModule) Init(app modular.Application) error {
    if err := m.initDatabase(); err != nil {
        return fmt.Errorf("failed to initialize database: %w", err)
    }
    return nil
}

func (m *MyModule) initDatabase() error {
    db, err := sql.Open("postgres", m.config.DSN)
    if err != nil {
        return fmt.Errorf("failed to open database connection: %w", err)
    }
    m.db = db
    return nil
}
```

This creates an error chain:
```
failed to initialize database: failed to open database connection: connection refused
```

## Error Inspection

### Using errors.Is

Check if an error is or wraps a specific error value:

```go
if errors.Is(err, modular.ErrServiceNotFound) {
    // Handle missing service
    log.Warn("Service not found")
}

if errors.Is(err, sql.ErrNoRows) {
    // Handle database record not found
    return nil // Not an error in this context
}
```

### Using errors.As

Extract a specific error type from an error chain:

```go
var configErr *modular.ConfigValidationError
if errors.As(err, &configErr) {
    // Handle configuration validation error
    log.Error("Config validation failed", 
        "field", configErr.Field,
        "value", configErr.Value)
}

var netErr *net.OpError
if errors.As(err, &netErr) {
    // Handle network operation error
    log.Error("Network error", 
        "op", netErr.Op,
        "addr", netErr.Addr)
}
```

## Best Practices

### 1. Always Wrap Errors with Context

Add context when wrapping errors to make debugging easier:

```go
// ❌ Bad: No context
if err := doSomething(); err != nil {
    return err
}

// ✅ Good: Add context
if err := doSomething(); err != nil {
    return fmt.Errorf("module %s initialization failed: %w", m.Name(), err)
}
```

### 2. Use Sentinel Errors for Well-Known Cases

Define package-level sentinel errors for expected error conditions:

```go
package mymodule

var (
    ErrNotInitialized = errors.New("module not initialized")
    ErrAlreadyStarted = errors.New("module already started")
    ErrInvalidConfig  = errors.New("invalid configuration")
)

func (m *MyModule) Start(ctx context.Context) error {
    if m.started {
        return ErrAlreadyStarted
    }
    if m.config == nil {
        return ErrNotInitialized
    }
    // ...
}
```

### 3. Don't Wrap Errors Unnecessarily

If an error already has sufficient context, don't wrap it:

```go
// ❌ Bad: Over-wrapping
if err := os.ReadFile("config.yaml"); err != nil {
    return fmt.Errorf("error reading file: %w", err)
    // os.ReadFile already includes the filename
}

// ✅ Good: Only wrap when adding value
if err := validateConfig(config); err != nil {
    return fmt.Errorf("config validation failed for %s: %w", m.Name(), err)
}
```

### 4. Check Errors at Multiple Levels

Use `errors.Is` and `errors.As` at different levels:

```go
func handleRequest(w http.ResponseWriter, r *http.Request) {
    if err := processRequest(r); err != nil {
        // Check for specific errors
        if errors.Is(err, modular.ErrTenantNotFound) {
            http.Error(w, "Tenant not found", http.StatusNotFound)
            return
        }
        
        // Check for validation errors
        var validationErr *ValidationError
        if errors.As(err, &validationErr) {
            http.Error(w, validationErr.Error(), http.StatusBadRequest)
            return
        }
        
        // Generic error
        http.Error(w, "Internal server error", http.StatusInternalServerError)
    }
}
```

### 5. Log Errors Appropriately

Include relevant context when logging errors:

```go
if err := m.Start(ctx); err != nil {
    m.logger.Error("Failed to start module",
        "module", m.Name(),
        "error", err,
        "config", m.config, // Be careful with sensitive data
    )
    return err
}
```

### 6. Return Errors, Don't Panic

Prefer returning errors over panicking:

```go
// ❌ Bad: Panic on error
func (m *MyModule) Init(app modular.Application) error {
    if m.config == nil {
        panic("config is nil")
    }
    // ...
}

// ✅ Good: Return error
func (m *MyModule) Init(app modular.Application) error {
    if m.config == nil {
        return fmt.Errorf("config is nil for module %s", m.Name())
    }
    // ...
}
```

### 7. Use Custom Error Types for Rich Context

Define custom error types when you need to provide additional context:

```go
type ValidationError struct {
    Field   string
    Value   interface{}
    Message string
}

func (e *ValidationError) Error() string {
    return fmt.Sprintf("validation failed for field %s: %s", e.Field, e.Message)
}

func validatePort(port int) error {
    if port < 1024 || port > 65535 {
        return &ValidationError{
            Field:   "port",
            Value:   port,
            Message: "must be between 1024 and 65535",
        }
    }
    return nil
}
```

### 8. Document Error Conditions

Document what errors a function can return:

```go
// Init initializes the module.
//
// Returns:
//   - ErrConfigValidationFailed if configuration is invalid
//   - ErrServiceNotFound if required service is not available
//   - Any error from underlying database initialization
func (m *MyModule) Init(app modular.Application) error {
    // ...
}
```

## Complete Example

```go
package mymodule

import (
    "context"
    "errors"
    "fmt"
    
    "github.com/CrisisTextLine/modular"
)

var (
    ErrNotInitialized = errors.New("module not initialized")
    ErrAlreadyStarted = errors.New("module already started")
)

type Module struct {
    config  *Config
    started bool
}

func (m *Module) Init(app modular.Application) error {
    // Get configuration
    var config *Config
    provider := app.GetConfigProvider(m.Name())
    if provider == nil {
        return fmt.Errorf("%w: section %s", modular.ErrConfigSectionNotFound, m.Name())
    }
    
    config = provider.GetConfig().(*Config)
    
    // Validate configuration
    if err := config.Validate(); err != nil {
        return fmt.Errorf("%w: %v", modular.ErrConfigValidationFailed, err)
    }
    
    m.config = config
    return nil
}

func (m *Module) Start(ctx context.Context) error {
    if m.config == nil {
        return ErrNotInitialized
    }
    
    if m.started {
        return ErrAlreadyStarted
    }
    
    // Start the module
    if err := m.startServer(ctx); err != nil {
        return fmt.Errorf("failed to start server: %w", err)
    }
    
    m.started = true
    return nil
}

func (m *Module) startServer(ctx context.Context) error {
    // Implementation...
    return nil
}
```

## See Also

- [Debugging](debugging.md) - Debugging and troubleshooting
- [Module Lifecycle](module-lifecycle.md) - Module initialization and lifecycle
- [Configuration](configuration.md) - Configuration validation
- [Testing](testing.md) - Testing error conditions
