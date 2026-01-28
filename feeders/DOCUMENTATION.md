# Environment Variable Catalog and Feeder Integration

This document describes how the Modular framework's enhanced configuration system handles environment variables and manages feeder precedence.

## Overview

The Modular framework uses a unified **Environment Catalog** system that combines environment variables from multiple sources:
- Operating System environment variables
- `.env` file variables
- Dynamically set variables

This allows all environment-based feeders (EnvFeeder, AffixedEnvFeeder, InstanceAwareEnvFeeder, TenantAffixedEnvFeeder) to access variables from both OS environment and .env files with proper precedence handling.

## Environment Catalog System

### EnvCatalog Architecture

The `EnvCatalog` provides:
- **Unified Access**: Single interface for all environment variables
- **Source Tracking**: Tracks whether variables come from OS env or .env files
- **Precedence Management**: OS environment always takes precedence over .env files
- **Thread Safety**: Concurrent access safe with mutex protection

### Variable Precedence

1. **OS Environment Variables** (highest precedence)
2. **.env File Variables** (lower precedence)

When the same variable exists in both sources, the OS environment value is used.

### Global Catalog

- Single global catalog instance shared across all env-based feeders
- Initialized once and reused for performance
- Can be reset for testing scenarios

## Feeder Types and Integration

### File-Based Feeders
These feeders read from configuration files and populate structs directly:

1. **YamlFeeder**: Reads YAML files, supports nested structures
2. **JSONFeeder**: Reads JSON files, handles complex object hierarchies  
3. **TomlFeeder**: Reads TOML files, supports all TOML data types
4. **DotEnvFeeder**: Special hybrid - loads .env into catalog AND populates structs

### Environment-Based Feeders
These feeders read from the unified Environment Catalog:

1. **EnvFeeder**: Basic env var lookup using struct field `env` tags
2. **AffixedEnvFeeder**: Adds prefix/suffix to env variable names
3. **InstanceAwareEnvFeeder**: Handles instance-specific configurations
4. **TenantAffixedEnvFeeder**: Combines tenant-aware and affixed behavior

### DotEnvFeeder Behavior

The `DotEnvFeeder` has dual behavior:
1. **Catalog Population**: Loads .env variables into the global catalog for other env feeders
2. **Direct Population**: Populates config structs using catalog (respects OS env precedence)

This allows other env-based feeders to access .env variables while maintaining proper precedence.

## Field-Level Tracking

All feeders support comprehensive field-level tracking that records:

- **Field Path**: Complete field path (e.g., "Database.Connections.primary.DSN")
- **Field Type**: Data type of the field
- **Feeder Type**: Which feeder populated the field
- **Source Type**: Source category (env, yaml, json, toml, dotenv)
- **Source Key**: The actual key used (e.g., "DB_PRIMARY_DSN")
- **Value**: The value that was set
- **Search Keys**: All keys that were searched
- **Found Key**: The key that was actually found
- **Instance Info**: For instance-aware feeders

### Tracking Usage

```go
tracker := NewDefaultFieldTracker()
feeder.SetFieldTracker(tracker)

// After feeding
populations := tracker.GetFieldPopulations()
for _, pop := range populations {
    fmt.Printf("Field %s set to %v from %s\n", 
        pop.FieldPath, pop.Value, pop.SourceKey)
}
```

## Feeder Evaluation Order and Precedence

### Priority Control System (v1.12+)

All feeders now support explicit priority control via the `WithPriority(n)` method. This allows you to precisely control which configuration sources override others, solving common issues like test isolation.

**Key Concepts:**
- Higher priority values mean the feeder is applied later
- Later application means the feeder can override earlier feeders
- Default priority is 0 if not specified
- When priorities are equal, original order is preserved (stable sort)

**Use Cases:**
1. **Test Isolation**: Set YAML test configs to higher priority than environment variables
2. **Production Overrides**: Set environment variables to higher priority than config files
3. **Layered Configuration**: Use priority levels (e.g., 10, 50, 100) to create clear precedence layers

### Recommended Order

When using multiple feeders, the typical order is:

1. **File-based feeders** (YAML/JSON/TOML) - set base configuration
2. **DotEnvFeeder** - load .env variables into catalog  
3. **Environment-based feeders** - override with env-specific values

**With Priority Control:**
- Use lower priorities (e.g., 50) for base/default configuration
- Use higher priorities (e.g., 100) for overrides
- Explicit priority values make configuration behavior predictable

### Precedence Rules

**With Priority Control (Recommended)**: Use `.WithPriority(n)` to explicitly control precedence
- Higher priority values are applied later, allowing them to override lower priority feeders
- Default priority is 0 if not specified
- When priorities are equal, original order is preserved

**Without Priority (Legacy Behavior)**: Order of execution determines precedence
- Last feeder wins (overwrites previous values)

**For environment variables**: OS environment always beats .env files within the catalog system

### Example Multi-Feeder Setup

#### Production Configuration (Env Overrides Config Files)

```go
config := modular.NewConfig()

// Base configuration from YAML (lower priority)
config.AddFeeder(feeders.NewYamlFeeder("config.yaml").WithPriority(50))

// Load .env into catalog for other env feeders (medium priority)
config.AddFeeder(feeders.NewDotEnvFeeder(".env").WithPriority(75))

// Environment-based overrides (highest priority)
config.AddFeeder(feeders.NewEnvFeeder().WithPriority(100))
config.AddFeeder(feeders.NewAffixedEnvFeeder("APP_", "_PROD").WithPriority(100))

// Feed the configuration
err := config.Feed(&appConfig)
```

#### Test Configuration (Config Files Override Env)

```go
config := modular.NewConfig()

// Environment variables (lower priority - won't override test config)
config.AddFeeder(feeders.NewEnvFeeder().WithPriority(50))

// Test YAML configuration (higher priority - overrides environment)
config.AddFeeder(feeders.NewYamlFeeder("test-config.yaml").WithPriority(100))

// Feed the configuration
err := config.Feed(&appConfig)
```

### Precedence Flow

**With Priority Control:**
```
Lower Priority Values → Higher Priority Values
  (e.g., priority 50)  →  (e.g., priority 100)
```

**Without Priority (Legacy):**
```
YAML values → DotEnv values → OS Env values → Affixed Env values
   (base)    →  (if not in OS) →  (override)  →   (final override)
```

## Environment Variable Naming Patterns

### EnvFeeder
Uses env tags directly: `env:"DATABASE_URL"`

### AffixedEnvFeeder
Constructs: `PREFIX + ENVTAG + SUFFIX`
- Example with prefix `"PROD_"`, tag `"HOST"`, suffix `"_ENV"`: `PROD_HOST_ENV`
- **Users must include separators (like underscores) in their prefix/suffix**
- Framework no longer automatically adds underscores between components

### InstanceAwareEnvFeeder
Constructs: `MODULE_INSTANCE_FIELD`
- Example: `DB_PRIMARY_DSN`, `DB_SECONDARY_DSN`

### TenantAffixedEnvFeeder
Combines tenant ID with affixed pattern:
- Example with prefix function `tenantId + "_"` and tag `"CONFIG"`: `TENANT123_CONFIG`
- Prefix/suffix functions must include any desired separators
- Preserves pre-configured prefix/suffix when used with tenant config loader

## Error Handling

The system uses static error definitions to comply with linting rules:

```go
var (
    ErrDotEnvInvalidStructureType = errors.New("expected pointer to struct")
    ErrJSONCannotConvert         = errors.New("cannot convert value to field type")
    // ... more specific errors
)
```

Errors are wrapped with context using `fmt.Errorf("%w: %s", baseError, context)`.

## Verbose Debug Logging

All feeders support verbose debug logging for troubleshooting:

```go
feeder.SetVerboseDebug(true, logger)
```

Debug output includes:
- Environment variable lookups and results
- Field processing steps
- Type conversion attempts
- Source tracking information
- Error details with context

## Best Practices

### Configuration Setup
1. **Use explicit priorities** for predictable configuration behavior
2. Use file-based feeders for base configuration (lower priority: 50)
3. Use DotEnvFeeder to load .env files for local development (medium priority: 75)
4. Use env-based feeders for deployment-specific overrides (higher priority: 100)
5. Set up field tracking for debugging and audit trails

### Priority Guidelines
1. **Reserve priority ranges for different purposes:**
   - 0-50: Base/default configuration (files, defaults)
   - 51-99: Environment-specific configuration (.env files)
   - 100+: Runtime overrides (OS environment variables, command-line flags)
2. **For tests:** Use higher priority for test configs to override host environment
3. **For production:** Use higher priority for environment variables to override defaults
4. **Document your priority scheme** in application documentation

### Environment Variable Management
1. Use consistent naming patterns
2. Document env var precedence in your application
3. Test with both OS env and .env file scenarios
4. Use verbose debugging during development
5. **Use priority control** to make precedence explicit and testable

### Error Handling
1. Always check feeder errors during configuration loading
2. Use field tracking to identify configuration sources
3. Validate required fields after feeding
4. Provide clear error messages for missing configuration

## Testing Considerations

### Test Isolation

**Problem:** Environment variables from the host system can interfere with test configuration.

**Solution:** Use priority control to ensure test configs override environment variables.

```go
func TestWithIsolation(t *testing.T) {
    // Host environment may have SDK_KEY set
    t.Setenv("SDK_KEY", "host-value")

    // Create test YAML with explicit config
    yamlPath := createTestYAML(t, `sdkKey: "test-value"`)

    // Use higher priority for test config to override environment
    config := modular.NewConfig()
    config.AddFeeder(feeders.NewEnvFeeder().WithPriority(50))       // Lower priority
    config.AddFeeder(feeders.NewYamlFeeder(yamlPath).WithPriority(100)) // Higher priority
    config.AddStructKey("_main", &cfg)
    config.Feed()

    // Test gets explicit YAML value, not environment variable
    assert.Equal(t, "test-value", cfg.SDKKey)
}
```

### Catalog Management
```go
// Reset catalog between tests if needed
feeders.ResetGlobalEnvCatalog()

// Use t.Setenv for test environment variables
t.Setenv("TEST_VAR", "test_value")
```

### Multi-Feeder Testing
Test various combinations of feeders to ensure proper precedence handling:
1. Test with and without priority specified
2. Test priority ordering with multiple feeders
3. Test backward compatibility (no priority = original order)

### Field Tracking Validation
Verify that field tracking correctly reports source information for debugging and audit purposes.
