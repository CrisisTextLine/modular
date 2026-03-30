# Module Lifecycle

Modules in the Modular framework go through a well-defined lifecycle with multiple phases: Registration, Configuration, Initialization, Startup, and Shutdown.

## Table of Contents

- [Registration](#registration)
- [Configuration](#configuration)
- [Initialization](#initialization)
- [Startup](#startup)
- [Shutdown](#shutdown)
- [Module Interfaces](#module-interfaces)

## Registration

Modules are registered with the Application, which adds them to an internal registry:

```go
app.RegisterModule(NewDatabaseModule())
app.RegisterModule(NewAPIModule())
```

The registration order doesn't matter - the framework automatically determines the correct initialization order based on dependencies.

## Configuration

During the application's `Init` phase, each module that implements the `Configurable` interface will have its `RegisterConfig` method called:

```go
// Implement the Configurable interface
func (m *MyModule) RegisterConfig(app modular.Application) error {
    m.config = &MyConfig{
        // Default values
        Port: 8080,
    }
    app.RegisterConfigSection(m.Name(), modular.NewStdConfigProvider(m.config))
    return nil // Note: This method returns error
}
```

**Key Points:**
- Configuration registration happens before initialization
- Each module can register its own configuration section
- Default values can be set in the config struct
- The framework will apply configuration feeders to populate values

## Initialization

After configuration, modules are initialized in dependency order:

```go
func (m *MyModule) Init(app modular.Application) error {
    // Initialize the module with the configuration
    if m.config.Debug {
        app.Logger().Debug("Initializing module in debug mode", "module", m.Name())
    }
    
    // Set up resources
    return nil
}
```

**Initialization Order:**
1. Modules with no dependencies are initialized first
2. Modules with dependencies are initialized after their dependencies
3. Circular dependencies are detected and cause an error
4. Interface-based service matching creates implicit dependencies

## Startup

When the application starts, each module that implements the `Startable` interface will have its `Start` method called:

```go
// Implement the Startable interface
func (m *MyModule) Start(ctx context.Context) error {
    // Start services
    m.server = &http.Server{
        Addr:    fmt.Sprintf(":%d", m.config.Port),
        Handler: m.router,
    }
    
    go func() {
        if err := m.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            m.logger.Error("Server error", "error", err)
        }
    }()
    
    return nil
}
```

**Important:**
- The `Start` method should be non-blocking for long-running services
- Use goroutines for services that run continuously
- Return errors only if the startup fails
- The context can be used for cancellation signals

## Shutdown

When the application stops, each module that implements the `Stoppable` interface will have its `Stop` method called in reverse initialization order:

```go
// Implement the Stoppable interface
func (m *MyModule) Stop(ctx context.Context) error {
    // Graceful shutdown
    return m.server.Shutdown(ctx)
}
```

**Shutdown Order:**
- Modules are stopped in reverse of initialization order
- This ensures dependencies are available during shutdown
- The context may have a timeout for graceful shutdown
- Return errors if shutdown fails

## Module Interfaces

The framework uses Go's interface composition to provide optional functionality:

### Core Interface (Required)

```go
type Module interface {
    Name() string
    Init(app Application) error
}
```

### Optional Interfaces

**Configurable** - For modules with configuration:
```go
type Configurable interface {
    RegisterConfig(app Application) error
}
```

**DependencyAware** - For modules with dependencies:
```go
type DependencyAware interface {
    Dependencies() []string
}
```

**ServiceAware** - For modules providing or requiring services:
```go
type ServiceAware interface {
    ProvidesServices() []ServiceProvider
    RequiresServices() []ServiceDependency
}
```

**Startable** - For modules that can be started:
```go
type Startable interface {
    Start(ctx context.Context) error
}
```

**Stoppable** - For modules that can be stopped:
```go
type Stoppable interface {
    Stop(ctx context.Context) error
}
```

**Constructable** - For modules with custom constructors:
```go
type Constructable interface {
    Constructor() ModuleConstructor
}
```

**TenantAwareModule** - For multi-tenant modules:
```go
type TenantAwareModule interface {
    Module
    OnTenantRegistered(tenantID TenantID)
    OnTenantRemoved(tenantID TenantID)
}
```

## Complete Lifecycle Example

```go
type MyModule struct {
    config *MyConfig
    db     *sql.DB
    server *http.Server
}

func (m *MyModule) Name() string {
    return "mymodule"
}

func (m *MyModule) RegisterConfig(app modular.Application) error {
    m.config = &MyConfig{Port: 8080}
    app.RegisterConfigSection(m.Name(), modular.NewStdConfigProvider(m.config))
    return nil
}

func (m *MyModule) Dependencies() []string {
    return []string{"database"}
}

func (m *MyModule) Init(app modular.Application) error {
    // Get database service
    if err := app.GetService("database", &m.db); err != nil {
        return err
    }
    return nil
}

func (m *MyModule) Start(ctx context.Context) error {
    m.server = &http.Server{
        Addr:    fmt.Sprintf(":%d", m.config.Port),
        Handler: m.router,
    }
    
    go func() {
        if err := m.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            // Handle error
        }
    }()
    
    return nil
}

func (m *MyModule) Stop(ctx context.Context) error {
    return m.server.Shutdown(ctx)
}
```

## See Also

- [Service Dependencies](service-dependencies.md) - Service injection and dependency resolution
- [Configuration](configuration.md) - Configuration system
- [Application Builder](application-builder.md) - Building applications
- [Testing](testing.md) - Testing modules
