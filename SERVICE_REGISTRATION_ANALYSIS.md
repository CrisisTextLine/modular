# Service Registration Timing Analysis

## Issue Summary

The reported issue claimed that services declared via `ProvidesServices()` were not available to dependent modules during their `Init()` phase, causing "service not found" errors.

## Investigation Results

After comprehensive testing and code analysis, I've determined that **the service registration timing is working correctly** in the current codebase. The issue appears to be a documentation problem rather than a code bug.

## How Service Registration Actually Works

### Current Implementation (Verified)

The framework processes modules in dependency order and for each module:

1. **Inject Required Services** (line 544 in application.go)
   - Services declared via `RequiresServices()` are resolved
   - Module constructor is called with injected services (if applicable)

2. **Initialize Module** (line 557)
   - `module.Init(app)` is called
   - Module can access services via `app.GetService()`

3. **Register Provided Services** (line 564)
   - Services declared in `ProvidesServices()` are registered
   - They become available for subsequent modules

4. **Next Module** (loop continues)
   - Process repeats for next module in dependency order

### Key Insight: Implicit Dependency Resolution

The framework already implements implicit dependency resolution! When a module declares:

```go
func (m *MyModule) RequiresServices() []ServiceDependency {
    return []ServiceDependency{
        {
            Name:     "scheduler.provider",
            Required: true,
        },
    }
}
```

The framework automatically:
- Identifies which module provides "scheduler.provider"
- Adds an implicit dependency edge in the module graph
- Ensures the provider module initializes first

This is implemented in `application.go`:
- `addImplicitDependencies()` (line 1054)
- `addNameBasedDependencies()` (line 1209)
- `addNameBasedDependency()` (line 1440)

## The Real Problem: Incomplete Documentation

The issue occurs when developers:
1. Call `app.GetService()` during `Init()`
2. But DON'T declare the dependency via `RequiresServices()` or `Dependencies()`

Without explicit dependency declaration, modules initialize in alphabetical order, which may not match service dependency requirements.

## Solutions Implemented

### 1. Updated Scheduler Module README

Added clear documentation showing three approaches:

**Option 1: RequiresServices() (Recommended)**
```go
func (m *MyModule) RequiresServices() []ServiceDependency {
    return []ServiceDependency{
        {Name: "scheduler.provider", Required: true},
    }
}

func (m *MyModule) Init(app Application) error {
    var scheduler *SchedulerModule
    err := app.GetService("scheduler.provider", &scheduler)
    // ...
}
```

**Option 2: Module Dependencies()**
```go
func (m *MyModule) Dependencies() []string {
    return []string{"scheduler"}
}
```

**Option 3: Interface-Based Matching**
```go
func (m *MyModule) RequiresServices() []ServiceDependency {
    return []ServiceDependency{
        {
            Name:               "scheduler",
            Required:           true,
            MatchByInterface:   true,
            SatisfiesInterface: reflect.TypeOf((*SchedulerModule)(nil)).Elem(),
        },
    }
}
```

### 2. Added Prominent Warning

> ⚠️ **Important**: If you access the scheduler service during `Init()` using `app.GetService()` without declaring the dependency via `RequiresServices()` or `Dependencies()`, your module may be initialized before the scheduler module, causing a "service not found" error. Always declare service dependencies explicitly to ensure proper initialization order.

### 3. Comprehensive Test Coverage

Created tests in `service_registration_timing_test.go` that validate:

- ✅ Services available during Init() with explicit Dependencies()
- ✅ Services available via RequiresServices() + Constructor pattern
- ✅ Implicit dependency ordering with RequiresServices() + Required:true
- ✅ Failure scenario when dependencies not declared
- ✅ Real-world scheduler + jobs module pattern

Created example in `scheduler_dependency_example_test.go` demonstrating:
- Correct usage of RequiresServices() with scheduler
- Automatic initialization order resolution
- Module registration order independence

## Test Results

All tests pass successfully:

```
=== RUN   TestServiceRegistrationTiming
--- PASS: TestServiceRegistrationTiming (0.00s)
=== RUN   TestServiceRegistrationTimingWithConstructor
--- PASS: TestServiceRegistrationTimingWithConstructor (0.00s)
=== RUN   TestServiceRegistrationTimingWithoutDependencies
--- PASS: TestServiceRegistrationTimingWithoutDependencies (0.00s)
=== RUN   TestServiceRegistrationTimingWithRequiresServices
--- PASS: TestServiceRegistrationTimingWithRequiresServices (0.00s)
=== RUN   TestSchedulerServiceRegistrationTiming
--- PASS: TestSchedulerServiceRegistrationTiming (0.00s)
=== RUN   TestSchedulerDependencyPattern
--- PASS: TestSchedulerDependencyPattern (0.00s)
```

## Conclusion

The service registration timing issue described in the original report **does not exist** in the current codebase. The framework already:

1. Registers services immediately after each module's Init() completes
2. Creates implicit dependencies based on RequiresServices() declarations
3. Initializes modules in correct dependency order

The actual problem was **insufficient documentation** about the requirement to declare service dependencies. This has been resolved by:

1. Updating module documentation with clear examples
2. Adding warnings about proper dependency declaration
3. Creating comprehensive tests demonstrating correct usage
4. Providing working examples of the dependency pattern

## Recommendations

For developers using the framework:

1. **Always declare service dependencies** using `RequiresServices()` or `Dependencies()`
2. **Prefer RequiresServices()** for service-level dependencies (more granular)
3. **Use Dependencies()** for module-level dependencies (simpler but less flexible)
4. **Test module initialization order** to ensure dependencies are resolved correctly

For framework maintainers:

1. Consider adding runtime warnings when `GetService()` is called during Init() for undeclared services
2. Add dependency graph visualization tools for debugging
3. Improve error messages to suggest using RequiresServices() when service not found during Init()
