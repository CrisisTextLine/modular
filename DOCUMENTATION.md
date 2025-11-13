# Modular Framework - Complete Documentation Index

> **Note:** This documentation has been reorganized into focused topic areas for easier navigation. For specific topics, please refer to the links below.

## Documentation Organization

The Modular framework documentation is now organized into focused topic files in the `docs/` directory. This makes it easier to find specific information and maintain up-to-date documentation.

## Quick Navigation

### Getting Started
- **[README](README.md)** - Quick start, installation, and basic usage
- **[Examples](examples/)** - Working example applications

### Core Framework

| Topic | Documentation |
|-------|---------------|
| **Application Builder** | [Application Builder API](docs/application-builder.md) - Builder pattern, decorators, functional options |
| **Configuration** | [Configuration System](docs/configuration.md) - Config providers, validation, feeders, instance-aware config |
| **Base Configuration** | [Base Config Guide](docs/base-config.md) - Multi-environment config management |
| **Module Lifecycle** | [Module Lifecycle](docs/module-lifecycle.md) - Registration, initialization, startup, shutdown |
| **Service Dependencies** | [Service Dependencies](docs/service-dependencies.md) - Service registry, dependency injection, interface matching |
| **Multi-Tenancy** | [Multi-Tenancy Guide](docs/multi-tenancy.md) - Tenant-aware modules, configuration, and services |
| **Testing** | [Testing Guide](docs/testing.md) - Testing modules, mocking, parallelization |
| **Debugging** | [Debugging Guide](docs/debugging.md) - Diagnostic tools, troubleshooting, common issues |
| **Error Handling** | [Error Handling](docs/error-handling.md) - Error types, wrapping, best practices |

### Advanced Topics

| Topic | Documentation |
|-------|---------------|
| **Observer Pattern** | [Observer Pattern](OBSERVER_PATTERN.md) - Event-driven communication |
| **CloudEvents** | [CloudEvents Integration](CLOUDEVENTS.md) - CloudEvents support |
| **Concurrency** | [Concurrency Guidelines](CONCURRENCY_GUIDELINES.md) - Thread safety, race avoidance |
| **Priority System** | [Priority System Guide](PRIORITY_SYSTEM_GUIDE.md) - Configuration priority |
| **API Contracts** | [API Contract Management](API_CONTRACT_MANAGEMENT.md) - API versioning |

### Modules

Each module has comprehensive documentation in its directory:

- **[auth](modules/auth/README.md)** - Authentication and authorization
- **[cache](modules/cache/README.md)** - Multi-backend caching
- **[chimux](modules/chimux/README.md)** - Chi router integration
- **[database](modules/database/README.md)** - Database connectivity
- **[eventbus](modules/eventbus/README.md)** - Event pub/sub messaging
- **[eventlogger](modules/eventlogger/README.md)** - Structured event logging
- **[httpclient](modules/httpclient/README.md)** - HTTP client
- **[httpserver](modules/httpserver/README.md)** - HTTP/HTTPS server
- **[jsonschema](modules/jsonschema/README.md)** - JSON Schema validation
- **[letsencrypt](modules/letsencrypt/README.md)** - SSL/TLS automation
- **[reverseproxy](modules/reverseproxy/README.md)** - Reverse proxy ([Configuration Guide](modules/reverseproxy/CONFIGURATION.md))
- **[scheduler](modules/scheduler/README.md)** - Job scheduling

For a complete overview, see [Modules README](modules/README.md).

## Core Concepts - Quick Reference

### Application

The Application is the central container that holds all modules, services, and configurations.

```go
// Create a new application using the builder pattern
app, err := modular.NewApplication(
    modular.WithLogger(logger),
    modular.WithConfigProvider(configProvider),
    modular.WithModules(
        &DatabaseModule{},
        &APIModule{},
    ),
)
```

For details, see [Application Builder API](docs/application-builder.md).

### Modules

Modules are the building blocks of a Modular application. Each module encapsulates specific functionality.

**Core Module Interface:**
```go
type Module interface {
    Name() string
    Init(app Application) error
}
```

**Optional Interfaces:**
- `Configurable` - For modules with configuration
- `DependencyAware` - For modules with dependencies
- `ServiceAware` - For modules providing/requiring services
- `Startable` - For modules that can be started
- `Stoppable` - For modules that can be stopped
- `Constructable` - For modules with custom constructors

For details, see [Module Lifecycle](docs/module-lifecycle.md).

### Service Registry

The Service Registry enables loose coupling through dependency injection.

```go
// Register a service
app.RegisterService("database", dbConnection)

// Get a service
var db *sql.DB
app.GetService("database", &db)
```

For details, see [Service Dependencies](docs/service-dependencies.md).

### Configuration

Flexible configuration system with validation and multiple sources.

```go
type AppConfig struct {
    Name    string `yaml:"name" default:"MyApp" desc:"Application name"`
    Version string `yaml:"version" required:"true" desc:"Application version"`
    Debug   bool   `yaml:"debug" default:"false" desc:"Enable debug mode"`
}
```

For details, see:
- [Configuration System](docs/configuration.md)
- [Base Configuration](docs/base-config.md)

### Multi-Tenancy

Built-in support for multi-tenant applications.

```go
// Create tenant context
tenantID := modular.TenantID("tenant1")
ctx := modular.NewTenantContext(context.Background(), tenantID)

// Create tenant-aware config
tenantAwareConfig := modular.NewTenantAwareConfig(
    modular.NewStdConfigProvider(&defaultConfig{}),
    tenantService,
    "mymodule",
)
```

For details, see [Multi-Tenancy Guide](docs/multi-tenancy.md).

## Migration from Old Documentation Structure

The previous monolithic `DOCUMENTATION.md` file has been reorganized into focused topic files for better maintainability and navigation. If you're looking for specific content:

| Old Section | New Location |
|-------------|--------------|
| Application Builder API | [docs/application-builder.md](docs/application-builder.md) |
| Configuration System | [docs/configuration.md](docs/configuration.md) |
| Base Configuration | [docs/base-config.md](docs/base-config.md) |
| Module Lifecycle | [docs/module-lifecycle.md](docs/module-lifecycle.md) |
| Service Dependencies | [docs/service-dependencies.md](docs/service-dependencies.md) |
| Multi-tenancy Support | [docs/multi-tenancy.md](docs/multi-tenancy.md) |
| Reverse Proxy Module | [modules/reverseproxy/CONFIGURATION.md](modules/reverseproxy/CONFIGURATION.md) |
| Testing Modules | [docs/testing.md](docs/testing.md) |
| Debugging and Troubleshooting | [docs/debugging.md](docs/debugging.md) |
| Error Handling | [docs/error-handling.md](docs/error-handling.md) |

## Additional Resources

- **[Migration Guide](MIGRATION_GUIDE.md)** - Upgrading between versions
- **[Go Module Versioning](GO_MODULE_VERSIONING.md)** - Semantic versioning guide
- **[Recommended Modules](RECOMMENDED_MODULES.md)** - Curated module list
- **[Examples](examples/)** - Working example applications
- **[API Reference](https://pkg.go.dev/github.com/CrisisTextLine/modular)** - Complete API documentation

## Contributing

For information on contributing to the framework documentation:

1. Follow the documentation structure in `docs/`
2. Keep focused documentation files under 500 lines when possible
3. Use cross-references to connect related topics
4. Include working code examples
5. Update the index when adding new documentation

## License

The Modular framework is licensed under the [MIT License](LICENSE).
