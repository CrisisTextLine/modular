# Error Handling

> **Note:** This is a placeholder file. Full content extraction from DOCUMENTATION.md is in progress.

The Modular framework defines common error types and follows Go's error wrapping conventions.

## Topics Covered

This document will cover:

- Common Error Types
- Error Wrapping
- Best Practices

## Quick Reference

### Common Error Types

```go
// Service errors
modular.ErrServiceAlreadyRegistered
modular.ErrServiceNotFound
modular.ErrServiceIncompatible

// Config errors
modular.ErrConfigSectionNotFound
modular.ErrConfigValidationFailed

// Dependency errors
modular.ErrCircularDependency
modular.ErrModuleDependencyMissing

// Tenant errors
modular.ErrTenantNotFound
modular.ErrTenantConfigNotFound
```

### Error Wrapping

```go
if err := doSomething(); err != nil {
    return fmt.Errorf("module '%s' failed: %w", m.Name(), err)
}

// Check with errors.Is
if errors.Is(err, modular.ErrServiceNotFound) {
    // Handle missing service
}

// Unwrap with errors.As
var configErr *modular.ConfigValidationError
if errors.As(err, &configErr) {
    // Handle validation error
}
```

**See:** The full content from the original `DOCUMENTATION.md` sections on:
- Error Handling
- Common Error Types
- Error Wrapping

> **TODO:** Extract full content from DOCUMENTATION.md lines 1378-1413
