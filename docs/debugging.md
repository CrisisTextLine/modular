# Debugging and Troubleshooting

The Modular framework provides several debugging utilities to help diagnose common issues with module lifecycle, interface implementation, and service injection.

## Table of Contents

- [Module Interface Debugging](#module-interface-debugging)
- [Common Issues](#common-issues)
- [Diagnostic Tools](#diagnostic-tools)
- [Debugging Workflows](#debugging-workflows)
- [Best Practices](#best-practices-for-debugging)

## Module Interface Debugging

### DebugModuleInterfaces

Use `DebugModuleInterfaces` to check which interfaces a specific module implements:

```go
import "github.com/CrisisTextLine/modular"

// Debug a specific module
modular.DebugModuleInterfaces(app, "your-module-name")
```

**Output example:**
```
🔍 Debugging module 'web-server' (type: *webserver.Module)
   Memory address: 0x14000026840
   ✅ Module
   ✅ Configurable
   ❌ DependencyAware
   ✅ ServiceAware
   ✅ Startable
   ✅ Stoppable
   ❌ Constructable
   📦 Provides 1 services, Requires 0 services
```

### DebugAllModuleInterfaces

Debug all registered modules at once:

```go
// Debug all modules in the application
modular.DebugAllModuleInterfaces(app)
```

### CompareModuleInstances

Compare module instances before and after initialization to detect if modules are being replaced:

```go
// Store reference before initialization
originalModule := app.moduleRegistry["module-name"]

// After initialization
currentModule := app.moduleRegistry["module-name"]

modular.CompareModuleInstances(originalModule, currentModule, "module-name")
```

## Common Issues

### 1. "Module does not implement Startable, skipping"

**Symptoms:** Module has a `Start` method but is reported as not implementing `Startable`.

**Common Causes:**

#### Incorrect method signature

Most common issue - missing context parameter:

```go
// ❌ WRONG - missing context parameter
func (m *MyModule) Start() error { 
    return nil 
}

// ✅ CORRECT
func (m *MyModule) Start(ctx context.Context) error { 
    return nil 
}
```

#### Wrong context import

```go
// ❌ WRONG - old context package
import "golang.org/x/net/context"

// ✅ CORRECT - standard library
import "context"
```

#### Constructor returns module without Startable interface

```go
// ❌ PROBLEMATIC - returns different type
func (m *MyModule) Constructor() ModuleConstructor {
    return func(app Application, services map[string]any) (Module, error) {
        return &DifferentModuleType{}, nil // Lost Startable!
    }
}

// ✅ CORRECT - preserves all interfaces
func (m *MyModule) Constructor() ModuleConstructor {
    return func(app Application, services map[string]any) (Module, error) {
        return m, nil // Or create new instance with all interfaces
    }
}
```

### 2. Service Injection Failures

**Symptoms:** `"failed to inject services for module"` errors.

**Debugging steps:**

1. Verify service names match exactly
2. Check that required services are provided by other modules
3. Ensure dependency order is correct
4. Use interface-based matching for flexibility

**Example:**

```go
// Check service is registered
var svc MyService
if err := app.GetService("myservice", &svc); err != nil {
    log.Printf("Service not found: %v", err)
}

// Use DebugModuleInterfaces to see what services are provided
modular.DebugModuleInterfaces(app, "provider-module")
```

### 3. Module Replacement Issues

**Symptoms:** Module works before `Init()` but fails after.

**Cause:** Constructor-based injection replaces the original module instance.

**Solution:** Ensure your Constructor returns a module that implements all the same interfaces.

```go
// Store original interfaces
type MyModule struct {
    // ... fields
}

// Ensure constructor preserves interfaces
func (m *MyModule) Constructor() ModuleConstructor {
    return func(app Application, services map[string]any) (Module, error) {
        // Create new instance with same type
        newModule := &MyModule{
            // ... initialize with services
        }
        return newModule, nil
    }
}
```

## Diagnostic Tools

### CheckModuleStartableImplementation

For detailed analysis of why a module doesn't implement Startable:

```go
import "github.com/CrisisTextLine/modular"

// Check specific module
modular.CheckModuleStartableImplementation(yourModule)
```

**Output includes:**
- Method signature analysis
- Expected vs actual parameter types
- Interface compatibility check

### Example Debugging Workflow

When troubleshooting module issues:

```go
func debugModuleIssues(app *modular.StdApplication) {
    // 1. Check all modules before initialization
    fmt.Println("=== BEFORE INIT ===")
    modular.DebugAllModuleInterfaces(app)
    
    // 2. Store references to original modules
    originalModules := make(map[string]modular.Module)
    for name, module := range app.GetAllModules() {
        originalModules[name] = module
    }
    
    // 3. Initialize
    err := app.Init()
    if err != nil {
        fmt.Printf("Init error: %v\n", err)
    }
    
    // 4. Check modules after initialization
    fmt.Println("=== AFTER INIT ===")
    modular.DebugAllModuleInterfaces(app)
    
    // 5. Compare instances
    for name, original := range originalModules {
        if current, exists := app.GetModule(name); exists {
            modular.CompareModuleInstances(original, current, name)
        }
    }
    
    // 6. Check specific problematic modules
    if problematicModule, exists := app.GetModule("problematic-module"); exists {
        modular.CheckModuleStartableImplementation(problematicModule)
    }
}
```

## Debugging Workflows

### Debugging Service Dependencies

```go
// 1. List all registered services
for name, service := range app.Services() {
    fmt.Printf("Service: %s (type: %T)\n", name, service)
}

// 2. Check module service requirements
module := app.GetModule("mymodule")
if serviceAware, ok := module.(modular.ServiceAware); ok {
    deps := serviceAware.RequiresServices()
    for _, dep := range deps {
        fmt.Printf("Requires: %s (required: %v)\n", dep.Name, dep.Required)
    }
}

// 3. Verify service availability
var svc MyService
if err := app.GetService("myservice", &svc); err != nil {
    fmt.Printf("Service not available: %v\n", err)
} else {
    fmt.Printf("Service found: %T\n", svc)
}
```

### Debugging Configuration

```go
// 1. Check registered config sections
sections := app.GetConfigSections()
for name := range sections {
    fmt.Printf("Config section: %s\n", name)
}

// 2. Inspect module configuration
provider := app.GetConfigProvider("mymodule")
if provider != nil {
    config := provider.GetConfig()
    fmt.Printf("Module config: %+v\n", config)
}

// 3. Validate configuration
if validator, ok := config.(modular.ConfigValidator); ok {
    if err := validator.Validate(); err != nil {
        fmt.Printf("Config validation failed: %v\n", err)
    }
}
```

### Debugging Module Initialization Order

```go
// Enable verbose logging
app.Logger().Info("Module initialization order:")

for i, moduleName := range app.GetInitializationOrder() {
    fmt.Printf("%d. %s\n", i+1, moduleName)
}
```

## Best Practices for Debugging

### 1. Add debugging early

Use debugging utilities during development, not just when issues occur:

```go
func main() {
    app := createApp()
    
    // Debug during development
    if os.Getenv("DEBUG") == "true" {
        modular.DebugAllModuleInterfaces(app)
    }
    
    if err := app.Run(); err != nil {
        log.Fatal(err)
    }
}
```

### 2. Check before and after Init()

Many issues occur during the initialization phase when modules are replaced via constructors:

```go
// Before Init
modular.DebugModuleInterfaces(app, "mymodule")

err := app.Init()

// After Init
modular.DebugModuleInterfaces(app, "mymodule")
```

### 3. Verify method signatures

Double-check that your Start/Stop methods match the expected interface signatures exactly:

```go
// Compile-time interface check
var _ modular.Startable = (*MyModule)(nil)
var _ modular.Stoppable = (*MyModule)(nil)
```

### 4. Use specific error messages

The debugging tools provide detailed information about why interfaces aren't implemented.

### 5. Test interface implementations

Add compile-time checks to ensure your modules implement expected interfaces:

```go
// This will fail to compile if MyModule doesn't implement Startable
var _ modular.Startable = (*MyModule)(nil)
var _ modular.Stoppable = (*MyModule)(nil)
var _ modular.Configurable = (*MyModule)(nil)
```

### 6. Check memory addresses

If memory addresses differ before and after Init(), your module was replaced by a constructor:

```go
modular.CompareModuleInstances(beforeModule, afterModule, "mymodule")
```

**Output:**
```
⚠️  Module 'mymodule' instance changed during initialization
   Before: 0x14000026840
   After:  0x14000026900
   This usually happens when using Constructor-based injection
```

### 7. Enable verbose logging

Use structured logging to track module lifecycle:

```go
logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelDebug,
}))

app := modular.NewStdApplication(configProvider, logger)
```

### 8. Use debug endpoints

If you're building a web service, add debug endpoints:

```go
http.HandleFunc("/debug/modules", func(w http.ResponseWriter, r *http.Request) {
    modules := app.GetAllModules()
    for name, module := range modules {
        fmt.Fprintf(w, "Module: %s (type: %T)\n", name, module)
    }
})
```

## Common Patterns

### Pattern: Verify Module Implementation

```go
func verifyModuleImplementation(module modular.Module) {
    fmt.Printf("Checking module: %s\n", module.Name())
    
    if _, ok := module.(modular.Startable); ok {
        fmt.Println("  ✅ Implements Startable")
    }
    
    if _, ok := module.(modular.Stoppable); ok {
        fmt.Println("  ✅ Implements Stoppable")
    }
    
    if _, ok := module.(modular.Configurable); ok {
        fmt.Println("  ✅ Implements Configurable")
    }
    
    if sa, ok := module.(modular.ServiceAware); ok {
        fmt.Println("  ✅ Implements ServiceAware")
        fmt.Printf("     Provides: %d services\n", len(sa.ProvidesServices()))
        fmt.Printf("     Requires: %d services\n", len(sa.RequiresServices()))
    }
}
```

### Pattern: Debug Service Resolution

```go
func debugServiceResolution(app modular.Application, serviceName string) {
    var svc interface{}
    
    // Try to get service
    err := app.GetService(serviceName, &svc)
    
    if err != nil {
        fmt.Printf("❌ Service '%s' not found: %v\n", serviceName, err)
        
        // List available services
        fmt.Println("Available services:")
        for name := range app.Services() {
            fmt.Printf("  - %s\n", name)
        }
    } else {
        fmt.Printf("✅ Service '%s' found (type: %T)\n", serviceName, svc)
    }
}
```

## See Also

- [Module Lifecycle](module-lifecycle.md) - Understanding module lifecycle
- [Service Dependencies](service-dependencies.md) - Service injection
- [Testing](testing.md) - Testing modules
- [Application Builder](application-builder.md) - Building applications
