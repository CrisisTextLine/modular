# Modular Framework Documentation

Welcome to the comprehensive documentation for the Modular Go framework. This documentation is organized into focused topic areas for easier navigation and maintenance.

## Quick Start

For a quick introduction to the framework, see the [README](../README.md) which includes:
- Installation instructions
- Basic usage examples
- Available modules overview
- Quick start guides

## Core Framework Documentation

The core framework documentation is organized into the following topic areas:

### Application Architecture

- **[Application Builder API](application-builder.md)** - Using the builder pattern to construct applications with decorators and functional options
- **[Module Lifecycle](module-lifecycle.md)** - Module registration, initialization, startup, and shutdown
- **[Service Dependencies](service-dependencies.md)** - Service registry, dependency injection, and interface-based matching

### Configuration

- **[Configuration System](configuration.md)** - Configuration providers, validation, feeders, and sample generation
- **[Base Configuration](base-config.md)** - Multi-environment configuration with base config and overrides
- **[Multi-Tenancy](multi-tenancy.md)** - Building multi-tenant applications with tenant-aware modules and configuration

### Development

- **[Debugging and Troubleshooting](debugging.md)** - Diagnostic tools, common issues, and debugging workflows
- **[Testing](testing.md)** - Testing modules, mock application, and parallelization strategies
- **[Error Handling](error-handling.md)** - Common error types, error wrapping, and best practices

### Advanced Topics

- **[Observer Pattern](../OBSERVER_PATTERN.md)** - Event-driven communication and observer pattern implementation
- **[CloudEvents Integration](../CLOUDEVENTS.md)** - CloudEvents specification support and event handling
- **[Concurrency Guidelines](../CONCURRENCY_GUIDELINES.md)** - Synchronization patterns, race avoidance, and thread safety
- **[Priority System](../PRIORITY_SYSTEM_GUIDE.md)** - Configuration priority and feeder ordering
- **[API Contract Management](../API_CONTRACT_MANAGEMENT.md)** - Managing API contracts and versioning

## Module Documentation

Each module has its own comprehensive documentation:

### Available Modules

| Module | Description | Documentation |
|--------|-------------|---------------|
| [auth](../modules/auth/) | Authentication and authorization | [README](../modules/auth/README.md) |
| [cache](../modules/cache/) | Multi-backend caching | [README](../modules/cache/README.md) |
| [chimux](../modules/chimux/) | Chi router integration | [README](../modules/chimux/README.md) |
| [database](../modules/database/) | Database connectivity | [README](../modules/database/README.md) |
| [eventbus](../modules/eventbus/) | Event pub/sub messaging | [README](../modules/eventbus/README.md) |
| [eventlogger](../modules/eventlogger/) | Structured event logging | [README](../modules/eventlogger/README.md) |
| [httpclient](../modules/httpclient/) | HTTP client | [README](../modules/httpclient/README.md) |
| [httpserver](../modules/httpserver/) | HTTP/HTTPS server | [README](../modules/httpserver/README.md) |
| [jsonschema](../modules/jsonschema/) | JSON Schema validation | [README](../modules/jsonschema/README.md) |
| [letsencrypt](../modules/letsencrypt/) | SSL/TLS automation | [README](../modules/letsencrypt/README.md) |
| [reverseproxy](../modules/reverseproxy/) | Reverse proxy | [README](../modules/reverseproxy/README.md), [Configuration Guide](../modules/reverseproxy/CONFIGURATION.md) |
| [scheduler](../modules/scheduler/) | Job scheduling | [README](../modules/scheduler/README.md) |

For a complete overview of modules, see the [Modules README](../modules/README.md).

## Examples

Working example applications demonstrating various features:

- **[basic-app](../examples/basic-app/)** - Simple modular application with HTTP server and routing
- **[reverse-proxy](../examples/reverse-proxy/)** - HTTP reverse proxy with load balancing
- **[http-client](../examples/http-client/)** - HTTP client integration patterns
- **[advanced-logging](../examples/advanced-logging/)** - Advanced logging and debugging
- **[observer-pattern](../examples/observer-pattern/)** - Event-driven architecture with CloudEvents
- **[base-config-example](../examples/base-config-example/)** - Multi-environment configuration management
- **[multi-tenant-app](../examples/multi-tenant-app/)** - Multi-tenant application patterns

See the [Examples README](../examples/README.md) for more details.

## Additional Resources

- **[Migration Guide](../MIGRATION_GUIDE.md)** - Upgrading from older versions
- **[Go Module Versioning](../GO_MODULE_VERSIONING.md)** - Understanding semantic versioning and releases
- **[Recommended Modules](../RECOMMENDED_MODULES.md)** - Curated list of recommended modules for common use cases
- **[Service Registration Analysis](../SERVICE_REGISTRATION_ANALYSIS.md)** - Deep dive into service registration

## API Reference

Complete API documentation is available on [pkg.go.dev](https://pkg.go.dev/github.com/CrisisTextLine/modular).

## Contributing

See the repository's contribution guidelines for information on how to contribute to the framework and its modules.

## Support

- GitHub Issues: Report bugs or request features
- Discussions: Ask questions and share ideas
- Examples: Learn from working examples in the examples/ directory

## License

The Modular framework is licensed under the [MIT License](../LICENSE).
