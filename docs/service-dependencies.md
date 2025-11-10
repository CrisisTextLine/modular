# Service Dependencies and Dependency Injection

> **Note:** This is a placeholder file. Full content extraction from DOCUMENTATION.md is in progress.

The Modular framework provides a powerful dependency injection system with both name-based and interface-based service matching.

## Topics Covered

This document will cover:

- Basic Service Dependencies
- Interface-Based Service Matching
- Dependency Resolution
- Service Injection Techniques (Constructor vs Init-Time)
- Best Practices

## Quick Reference

```go
// Name-based service dependency
func (m *MyModule) RequiresServices() []modular.ServiceDependency {
    return []modular.ServiceDependency{
        {
            Name:     "database",
            Required: true,
        },
    }
}

// Interface-based service matching
func (m *MyModule) RequiresServices() []modular.ServiceDependency {
    return []modular.ServiceDependency{
        {
            Name:               "router",
            Required:           true,
            MatchByInterface:   true,
            SatisfiesInterface: reflect.TypeOf((*Router)(nil)).Elem(),
        },
    }
}
```

**See:** The full content from the original `DOCUMENTATION.md` sections on:
- Service Dependencies
- Basic Service Dependencies
- Interface-Based Service Matching
- Dependency Resolution with Interface Matching
- Service Injection Techniques

> **TODO:** Extract full content from DOCUMENTATION.md lines 409-651
