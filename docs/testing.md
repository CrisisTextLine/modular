# Testing Modules

The Modular framework provides testing utilities including a mock application and comprehensive parallelization strategies for writing effective tests.

## Table of Contents

- [Mock Application](#mock-application)
- [Testing Services](#testing-services)
- [Test Parallelization Strategy](#test-parallelization-strategy)
- [Best Practices](#best-practices)

## Mock Application

The mock application is a lightweight, in-memory implementation of the `Application` interface. It is useful for testing modules in isolation without starting the entire application.

### Creating a Mock Application

```go
// For testing individual modules, use the module's generated mock
// The modcli tool generates a NewMockApplication() function for each module
mockApp := NewMockApplication()

// Register modules for testing
mockApp.RegisterModule(NewDatabaseModule())
```

The mock application can be used to register modules and services for testing.

### Registering Modules

```go
// Register a module with the mock application
mockApp.RegisterModule(NewDatabaseModule())
```

### Registering Services

```go
// Register a service with the mock application
mockApp.RegisterService("database", &sql.DB{})
```

## Testing Services

Service testing focuses on verifying the behavior of individual services in isolation. This typically involves:

- Mocking dependencies
- Asserting method calls
- Verifying state changes

### Mocking Dependencies

Use the mock application to provide mock implementations of dependencies:

```go
// Mock a database connection
dbMock := &sql.DB{}

// Register the mock service
mockApp.RegisterService("database", dbMock)
```

### Asserting Method Calls

You can use testify's mock assertions to verify that methods are called with the expected arguments:

```go
// Assert that the Query method was called with the correct SQL
mockDB.AssertCalled(t, "Query", "SELECT * FROM users WHERE id = ?", 1)
```

### Verifying State Changes

Check that the state is modified as expected:

```go
// Assert the user was added to the database
var user User
mockDB.Find(&user, 1)
assert.Equal(t, "John Doe", user.Name)
```

## Test Parallelization Strategy

A pragmatic, rule-based approach is used to parallelize tests safely while maintaining determinism and clarity.

### Goals

- Reduce wall-clock CI time by leveraging `t.Parallel()` where side effects are eliminated
- Prevent data races or flakiness from shared mutable global state
- Encourage per-application configuration feeder usage over global mutation

### Key Rules (Go 1.25+)

1. **Environment Mutation Restriction**: A test (or subtest) that invokes `t.Setenv` or `t.Chdir` must not call `t.Parallel()` on the same `*testing.T` (runtime will panic: `test using t.Setenv or t.Chdir can not use t.Parallel`)

2. **Per-App Feeders**: Prefer `app.SetConfigFeeders(...)` (per-app feeders) instead of mutating the package-level `modular.ConfigFeeders` slice

3. **Shared Environment Setup**: Hoist shared environment setup to the parent test. Child subtests that do not mutate env / working directory can safely call `t.Parallel()`

4. **Avoid Shared Writable Globals**: Avoid shared writable globals (maps, slices, singletons). If unavoidable, keep the test serial and document the reason with a short comment

5. **Filesystem Isolation**: Use `t.TempDir()` for any filesystem interaction; never reuse paths across tests

6. **Network Isolation**: Allocate dynamic ports (port 0) or isolate networked resources; otherwise keep such tests serial

### Recommended Patterns

#### Serial parent + parallel children

```go
func TestWidgetModes(t *testing.T) {
    t.Setenv("WIDGET_FEATURE", "on") // parent is serial
    modes := []string{"fast","safe","debug"}
    for _, m := range modes {
        m := m
        t.Run(m, func(t *testing.T) {
            t.Parallel() // safe: no env mutation here
            // assertions using m
        })
    }
}
```

#### Fully serial when each case mutates env

```go
func TestModeMatrix(t *testing.T) {
    cases := []struct{Name, Value string}{{"Fast","fast"},{"Safe","safe"}}
    for _, c := range cases {
        t.Run(c.Name, func(t *testing.T) { // not parallel
            t.Setenv("MODE", c.Value)
            // assertions
        })
    }
}
```

### Documentation Comments

Add a brief note when a test stays serial:

```go
// NOTE: cannot parallelize: uses t.Setenv per subtest
```

### Field & Instance Tracking

Tests such as `TestInstanceAwareFieldTracking` remain serial by design because their correctness depends on sequential environment mutation establishing instance key prefixes.

**Rationale:** Clarity outweighs minor gains from forcing partial parallelism when setup complexity rises.

### Metrics & Auditing

- Count parallelized tests: `grep -R "t.Parallel()" -n . | wc -l`
- Identify env-mutating tests: `grep -R "t.Setenv(" -n .`

### Future Opportunities

- Snapshot helper(s) for any future global mutable state
- Containerized or ephemeral service fixtures for broader parallel integration testing

**When unsure, keep the test serial and explain why.**

## Best Practices

### 1. Use Mock Application for Unit Tests

```go
func TestMyModule(t *testing.T) {
    mockApp := modular.NewMockApplication(
        modular.WithLogger(testLogger),
        modular.WithConfigProvider(testConfig),
    )
    
    module := NewMyModule()
    mockApp.RegisterModule(module)
    
    // Test initialization
    err := mockApp.Init()
    assert.NoError(t, err)
}
```

### 2. Test Module Interfaces

Verify that modules implement expected interfaces:

```go
func TestModuleImplementsStartable(t *testing.T) {
    module := NewMyModule()
    
    // Compile-time check
    var _ modular.Startable = module
    
    // Runtime check
    _, ok := interface{}(module).(modular.Startable)
    assert.True(t, ok, "module should implement Startable")
}
```

### 3. Test Service Dependencies

```go
func TestModuleServiceDependencies(t *testing.T) {
    mockApp := modular.NewMockApplication()
    
    // Provide required services
    mockApp.RegisterService("database", &mockDB{})
    
    module := NewMyModule()
    mockApp.RegisterModule(module)
    
    err := mockApp.Init()
    assert.NoError(t, err)
}
```

### 4. Test Configuration Validation

```go
func TestConfigValidation(t *testing.T) {
    tests := []struct {
        name    string
        config  MyConfig
        wantErr bool
    }{
        {
            name:    "valid config",
            config:  MyConfig{Port: 8080},
            wantErr: false,
        },
        {
            name:    "invalid port",
            config:  MyConfig{Port: -1},
            wantErr: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := tt.config.Validate()
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

### 5. Use Table-Driven Tests

```go
func TestModuleBehavior(t *testing.T) {
    tests := []struct {
        name     string
        config   MyConfig
        expected string
    }{
        {"default", MyConfig{}, "default-value"},
        {"custom", MyConfig{Value: "custom"}, "custom"},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel() // Safe if no shared state
            
            module := NewMyModule()
            module.config = &tt.config
            
            result := module.GetValue()
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

### 6. Test Module Lifecycle

```go
func TestModuleLifecycle(t *testing.T) {
    mockApp := modular.NewMockApplication()
    module := NewMyModule()
    
    // Register
    mockApp.RegisterModule(module)
    
    // Configure
    err := module.RegisterConfig(mockApp)
    assert.NoError(t, err)
    
    // Initialize
    err = module.Init(mockApp)
    assert.NoError(t, err)
    
    // Start
    ctx := context.Background()
    err = module.Start(ctx)
    assert.NoError(t, err)
    
    // Stop
    err = module.Stop(ctx)
    assert.NoError(t, err)
}
```

### 7. Use Per-Application Configuration Feeders

```go
func TestConfigurationLoading(t *testing.T) {
    // Don't mutate global ConfigFeeders
    // Use per-app feeders instead
    
    app := modular.NewStdApplication(configProvider, logger)
    app.SetConfigFeeders([]modular.Feeder{
        feeders.NewYAMLFeeder("test-config.yaml"),
        feeders.NewEnvFeeder("TEST_"),
    })
    
    err := app.Init()
    assert.NoError(t, err)
}
```

### 8. Clean Up Resources

```go
func TestModuleWithCleanup(t *testing.T) {
    tmpDir := t.TempDir() // Automatically cleaned up
    
    module := NewMyModule()
    module.SetDataDir(tmpDir)
    
    err := module.Init(mockApp)
    assert.NoError(t, err)
    
    // Cleanup happens automatically
}
```

## Example: Complete Module Test

```go
package mymodule_test

import (
    "context"
    "testing"
    
    "github.com/CrisisTextLine/modular"
    "github.com/stretchr/testify/assert"
)

func TestMyModule(t *testing.T) {
    t.Run("implements required interfaces", func(t *testing.T) {
        module := NewMyModule()
        
        var _ modular.Module = module
        var _ modular.Startable = module
        var _ modular.Stoppable = module
    })
    
    t.Run("initializes correctly", func(t *testing.T) {
        mockApp := modular.NewMockApplication()
        module := NewMyModule()
        
        err := module.Init(mockApp)
        assert.NoError(t, err)
    })
    
    t.Run("handles service dependencies", func(t *testing.T) {
        mockApp := modular.NewMockApplication()
        mockApp.RegisterService("database", &mockDB{})
        
        module := NewMyModule()
        mockApp.RegisterModule(module)
        
        err := mockApp.Init()
        assert.NoError(t, err)
    })
    
    t.Run("starts and stops gracefully", func(t *testing.T) {
        mockApp := modular.NewMockApplication()
        module := NewMyModule()
        
        err := module.Init(mockApp)
        assert.NoError(t, err)
        
        ctx := context.Background()
        
        err = module.Start(ctx)
        assert.NoError(t, err)
        
        err = module.Stop(ctx)
        assert.NoError(t, err)
    })
}
```

## See Also

- [Debugging](debugging.md) - Debugging modules and troubleshooting
- [Module Lifecycle](module-lifecycle.md) - Module initialization
- [Service Dependencies](service-dependencies.md) - Service injection
- [Configuration](configuration.md) - Configuration testing
