# Core Framework Documentation

This directory contains the core framework documentation organized by topic. For a complete overview, see [index.md](index.md).

## Documentation Files

### Configuration
- **[configuration.md](configuration.md)** - Configuration system, providers, validation, and feeders
- **[base-config.md](base-config.md)** - Multi-environment configuration with base config and overrides

### Application Architecture
- **[application-builder.md](application-builder.md)** - Application builder pattern and decorators
- **[module-lifecycle.md](module-lifecycle.md)** - Module registration, initialization, and lifecycle
- **[service-dependencies.md](service-dependencies.md)** - Service registry and dependency injection

### Multi-Tenancy
- **[multi-tenancy.md](multi-tenancy.md)** - Building multi-tenant applications

### Development
- **[debugging.md](debugging.md)** - Debugging tools and troubleshooting
- **[testing.md](testing.md)** - Testing modules and parallelization
- **[error-handling.md](error-handling.md)** - Error types and best practices

## Organization Principles

1. **Focused Topics**: Each file covers a specific topic area
2. **Self-Contained**: Each file can be read independently
3. **Cross-Referenced**: Related topics link to each other
4. **Practical Examples**: Include working code examples
5. **Maintained**: Keep up-to-date with framework changes

## Related Documentation

- **Root Level Docs**: Specialized topics remain at repository root
  - [OBSERVER_PATTERN.md](../OBSERVER_PATTERN.md)
  - [CLOUDEVENTS.md](../CLOUDEVENTS.md)
  - [CONCURRENCY_GUIDELINES.md](../CONCURRENCY_GUIDELINES.md)
  - [And others...]

- **Module Docs**: Each module has its own README
  - [modules/auth/README.md](../modules/auth/README.md)
  - [modules/cache/README.md](../modules/cache/README.md)
  - [modules/reverseproxy/README.md](../modules/reverseproxy/README.md)
  - [And others...]

- **Examples**: Working example applications
  - [examples/basic-app/](../examples/basic-app/)
  - [examples/base-config-example/](../examples/base-config-example/)
  - [And others...]

## Contributing

When adding new documentation:

1. Keep files focused on a single topic
2. Aim for 200-500 lines per file
3. Include a table of contents for longer files
4. Add working code examples
5. Update [index.md](index.md) with new topics
6. Cross-reference related documentation
7. Test all code examples

## Navigation

- Start with [index.md](index.md) for a complete overview
- Use the main [README.md](../README.md) for quick start
- Check [DOCUMENTATION.md](../DOCUMENTATION.md) for topic index
