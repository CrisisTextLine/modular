# Application Builder API

The Modular framework v1.7 introduces a powerful builder pattern for constructing applications. This provides a clean, composable way to configure applications with various decorators and options.

## Table of Contents

- [Concurrency & Race Guidelines](#concurrency--race-guidelines)
- [Builder Pattern](#builder-pattern)
- [Functional Options](#functional-options)
- [Decorator Pattern](#decorator-pattern)

## Concurrency & Race Guidelines

For official guidance on synchronization patterns, avoiding data races, safe observer usage, defensive config copying, and request body handling for parallel fan-out, see the dedicated document: [Concurrency & Race Guidelines](../CONCURRENCY_GUIDELINES.md). All new modules must adhere to these standards and pass `go test -race`.

## Builder Pattern

The Modular framework provides a builder pattern for constructing applications with a clean, composable API.

### Basic Usage

```go
app, err := modular.NewApplication(
    modular.WithLogger(logger),
    modular.WithConfigProvider(configProvider),
    modular.WithModules(
        &DatabaseModule{},
        &APIModule{},
    ),
)
if err != nil {
    return err
}
```

## Functional Options

The builder uses functional options to provide flexibility and extensibility:

### Core Options

- **`WithLogger(logger)`**: Sets the application logger (required)
- **`WithConfigProvider(provider)`**: Sets the main configuration provider
- **`WithBaseApplication(app)`**: Wraps an existing application with decorators
- **`WithModules(modules...)`**: Registers multiple modules at construction time

### Configuration Options

- **`WithConfigDecorators(decorators...)`**: Applies configuration decorators for enhanced config processing
- **`InstanceAwareConfig()`**: Enables instance-aware configuration decoration
- **`TenantAwareConfigDecorator(loader)`**: Enables tenant-specific configuration overrides

### Enhanced Functionality Options

- **`WithTenantAware(loader)`**: Adds multi-tenant capabilities with automatic tenant resolution
- **`WithObserver(observers...)`**: Adds event observers for application lifecycle and custom events

## Decorator Pattern

The framework uses the decorator pattern to add cross-cutting concerns without modifying core application logic:

### TenantAwareDecorator

Wraps applications to add multi-tenant functionality:

```go
app, err := modular.NewApplication(
    modular.WithLogger(logger),
    modular.WithConfigProvider(configProvider),
    modular.WithTenantAware(&MyTenantLoader{}),
    modular.WithModules(modules...),
)
```

**Features:**
- Automatic tenant resolution during startup
- Tenant-scoped configuration and services
- Integration with tenant-aware modules

### ObservableDecorator

Adds observer pattern capabilities with CloudEvents integration:

```go
eventObserver := func(ctx context.Context, event cloudevents.Event) error {
    log.Printf("Event: %s from %s", event.Type(), event.Source())
    return nil
}

app, err := modular.NewApplication(
    modular.WithLogger(logger),
    modular.WithConfigProvider(configProvider),
    modular.WithObserver(eventObserver),
    modular.WithModules(modules...),
)
```

**Features:**
- Automatic emission of application lifecycle events
- CloudEvents specification compliance
- Multiple observer support with error isolation

### Benefits of Decorator Pattern

1. **Separation of Concerns**: Cross-cutting functionality is isolated in decorators
2. **Composability**: Multiple decorators can be combined as needed
3. **Flexibility**: Applications can be enhanced without changing core logic
4. **Testability**: Decorators can be tested independently

## See Also

- [Observer Pattern](../OBSERVER_PATTERN.md) - Event-driven communication
- [CloudEvents Integration](../CLOUDEVENTS.md) - CloudEvents support
- [Multi-Tenancy](multi-tenancy.md) - Tenant-aware applications
- [Module Lifecycle](module-lifecycle.md) - Module initialization
