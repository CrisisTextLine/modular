# Service Dependencies and Dependency Injection

The Modular framework provides a powerful dependency injection system with both name-based and interface-based service matching. This enables loose coupling between modules while ensuring proper initialization order.

## Table of Contents

- [Basic Service Dependencies](#basic-service-dependencies)
- [Interface-Based Service Matching](#interface-based-service-matching)
- [Dependency Resolution](#dependency-resolution-with-interface-matching)
- [Service Injection Techniques](#service-injection-techniques)
- [Best Practices](#best-practices-for-service-dependencies)

## Basic Service Dependencies

At the core of Modular's dependency injection system is the `ServiceDependency` struct, which allows modules to declare what services they require:

```go
type ServiceDependency struct {
    Name               string       // Service name to lookup
    Required           bool         // If true, application fails to start if service is missing
    Type               reflect.Type // Concrete type (if known)
    SatisfiesInterface reflect.Type // Interface type (if known)
    MatchByInterface   bool         // If true, find first service that satisfies interface type
}
```

The simplest form of dependency is a name-based lookup:

```go
func (m *MyModule) RequiresServices() []modular.ServiceDependency {
    return []modular.ServiceDependency{
        {
            Name:     "database",
            Required: true,
        },
    }
}
```

With this approach, the framework will look for a service registered with the exact name "database" and inject it into your module.

## Interface-Based Service Matching

A more flexible approach is to specify that your module requires a service that implements a particular interface, regardless of what name it was registered under. This is achieved using the `MatchByInterface` and `SatisfiesInterface` fields:

```go
func (m *MyModule) RequiresServices() []modular.ServiceDependency {
    return []modular.ServiceDependency{
        {
            Name:               "router", // The name used to access this service in your code
            Required:           true,
            MatchByInterface:   true,
            SatisfiesInterface: reflect.TypeOf((*Router)(nil)).Elem(), // The interface the service should implement
        },
    }
}
```

With this configuration, the framework will:

1. Search through all registered services (regardless of their names)
2. Find any service that implements the `Router` interface
3. Inject that service into your module under the name "router"

This allows for greater flexibility in how services are provided and consumed:

- Service providers can name their services however they want (e.g., "chi.router", "gin.router")
- Service consumers can rely on interface compatibility rather than specific implementations
- Implementations can be swapped without changing consumer code

### Example: Router Service

Consider a scenario where you have a module that needs a router service:

```go
// Define the router interface
type Router interface {
    HandleFunc(pattern string, handler func(http.ResponseWriter, http.Request))
}

// Module that requires any router service
func (m *APIModule) RequiresServices() []modular.ServiceDependency {
    return []modular.ServiceDependency{
        {
            Name:               "router",
            Required:           true,
            MatchByInterface:   true,
            SatisfiesInterface: reflect.TypeOf((*Router)(nil)).Elem(),
        },
    }
}

// Constructor that uses the router
func (m *APIModule) Constructor() modular.ModuleConstructor {
    return func(app modular.Application, services map[string]any) (modular.Module, error) {
        router := services["router"].(Router) // Cast to the interface type
        
        // Register API routes
        router.HandleFunc("/api/users", m.handleUsers)
        
        return m, nil
    }
}
```

Now you can use different router implementations without changing your API module:

```go
// Chi router module
app.RegisterModule(chimux.NewModule())

// OR a custom router
app.RegisterService("custom.router", &MyCustomRouter{})
```

Either way, the `APIModule` will receive a service that implements the `Router` interface, regardless of the actual implementation type or registered name.

### Multiple Interface Implementations

If multiple services in the application implement the same interface, the framework will use the first matching service it finds. This behavior is deterministic but may not always select the service you expect.

For more control in this scenario, you should:

1. Use more specific interfaces for different use cases
2. Use name-based lookup when you need a specific implementation
3. Consider using a selector pattern where a coordinator service decides which implementation to use

### Example: Multiple Logger Implementations

```go
// If multiple services implement the Logger interface,
// you might want to be more specific:
func (m *MyModule) RequiresServices() []modular.ServiceDependency {
    return []modular.ServiceDependency{
        {
            // When you need any logger:
            Name:               "logger",
            Required:           true,
            MatchByInterface:   true,
            SatisfiesInterface: reflect.TypeOf((*Logger)(nil)).Elem(),
        },
        {
            // When you need a specific logger:
            Name:     "json.logger", // Specific service name
            Required: true,
        },
    }
}
```

## Dependency Resolution with Interface Matching

The Modular framework automatically creates implicit dependencies between modules based on interface matching. This ensures that modules providing services are initialized before modules that require those services.

For example, if:
- Module A requires a service implementing interface X
- Module B provides a service implementing interface X

Then Module B will be initialized before Module A, even if there is no explicit dependency declared between them.

This automatic resolution ensures that services are available when needed, regardless of the order in which modules are registered with the application.

## Service Injection Techniques

### Constructor Injection

Constructor injection is the recommended approach for most scenarios:

```go
// Implement the ServiceAware interface
func (m *MyModule) RequiresServices() []modular.ServiceDependency {
    return []modular.ServiceDependency{
        {
            Name:               "db",
            Required:           true,
            MatchByInterface:   true,
            SatisfiesInterface: reflect.TypeOf((*Database)(nil)).Elem(),
        },
    }
}

func (m *MyModule) ProvidesServices() []modular.ServiceProvider {
    return nil // This module doesn't provide any services
}

// Implement the Constructable interface
func (m *MyModule) Constructor() modular.ModuleConstructor {
    return func(app modular.Application, services map[string]any) (modular.Module, error) {
        db, ok := services["db"].(Database)
        if !ok {
            return nil, errors.New("invalid database service")
        }
        
        // Create a new instance with the service
        return &MyModule{
            db: db,
            // Initialize other fields
        }, nil
    }
}
```

**Benefits of constructor injection:**
- Clear separation of concerns
- Immutable module state after construction
- Easy to test with mock services

### Init-Time Injection

For simpler modules, you can use init-time injection:

```go
// Implement the ServiceAware interface
type SimpleModule struct {
    db Database
}

func (m *SimpleModule) RequiresServices() []modular.ServiceDependency {
    return []modular.ServiceDependency{
        {
            Name:     "database",
            Required: true,
        },
    }
}

func (m *SimpleModule) ProvidesServices() []modular.ServiceProvider {
    return nil // This module doesn't provide any services
}

func (m *SimpleModule) Init(app modular.Application) error {
    // Get the service during initialization
    if err := app.GetService("database", &m.db); err != nil {
        return fmt.Errorf("failed to get database service: %w", err)
    }
    
    return nil
}
```

## Best Practices for Service Dependencies

When using interface-based service matching:

### 1. Design Focused Interfaces

Use the interface segregation principle - define small, focused interfaces rather than large, general ones.

```go
// Good: Focused interface
type UserRepository interface {
    GetUser(id string) (*User, error)
    SaveUser(user *User) error
}

// Avoid: Large, general interface
type Database interface {
    Query(sql string) ([]Row, error)
    Execute(sql string) error
    GetUser(id string) (*User, error)
    SaveUser(user *User) error
    // ... many more methods
}
```

### 2. Document Required Interfaces

Clearly document what interfaces your module expects services to implement:

```go
// Package mymodule requires a Router service that implements the
// following interface:
//
//   type Router interface {
//       HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request))
//   }
//
// The router service can be registered under any name; the module will
// find it automatically through interface matching.
package mymodule
```

### 3. Export Interfaces

Make interfaces public in their own package so they can be imported by both service providers and consumers:

```go
// In package interfaces
package interfaces

type Router interface {
    HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request))
}

// In your module
import "myapp/interfaces"

func (m *MyModule) RequiresServices() []modular.ServiceDependency {
    return []modular.ServiceDependency{
        {
            Name:               "router",
            Required:           true,
            MatchByInterface:   true,
            SatisfiesInterface: reflect.TypeOf((*interfaces.Router)(nil)).Elem(),
        },
    }
}
```

### 4. Use Interface-Based Matching Judiciously

Use interface-based matching for:
- Optional dependencies where you want flexibility
- Well-defined, stable interfaces
- When you want to swap implementations easily

Use name-based matching for:
- Required dependencies where you need a specific implementation
- Internal modules where you control both provider and consumer
- When you want explicit, clear dependencies

### 5. Consider Name Conventions

Even with interface matching, consider using consistent naming conventions for common service types:

```go
// Good: Consistent naming
"database.connection"
"cache.redis"
"logger.json"

// Avoid: Inconsistent naming
"db"
"redisCache"
"myLogger"
```

## Complete Example

Here's a complete example showing both service provision and consumption:

```go
// Service provider module
type DatabaseModule struct {
    db *sql.DB
}

func (m *DatabaseModule) Name() string {
    return "database"
}

func (m *DatabaseModule) Init(app modular.Application) error {
    // Initialize database connection
    db, err := sql.Open("postgres", m.config.DSN)
    if err != nil {
        return err
    }
    m.db = db
    return nil
}

func (m *DatabaseModule) ProvidesServices() []modular.ServiceProvider {
    return []modular.ServiceProvider{
        {
            Name:        "database",
            Description: "PostgreSQL database connection",
            Instance:    m.db,
        },
    }
}

// Service consumer module
type APIModule struct {
    db Database
}

func (m *APIModule) Name() string {
    return "api"
}

func (m *APIModule) RequiresServices() []modular.ServiceDependency {
    return []modular.ServiceDependency{
        {
            Name:               "db",
            Required:           true,
            MatchByInterface:   true,
            SatisfiesInterface: reflect.TypeOf((*Database)(nil)).Elem(),
        },
    }
}

func (m *APIModule) Constructor() modular.ModuleConstructor {
    return func(app modular.Application, services map[string]any) (modular.Module, error) {
        db := services["db"].(Database)
        
        return &APIModule{
            db: db,
        }, nil
    }
}

func (m *APIModule) Init(app modular.Application) error {
    // Module is already constructed with dependencies
    return nil
}
```

## See Also

- [Module Lifecycle](module-lifecycle.md) - Module initialization and ordering
- [Application Builder](application-builder.md) - Building applications with modules
- [Testing](testing.md) - Testing modules with mock services
- [Configuration](configuration.md) - Configuration system
