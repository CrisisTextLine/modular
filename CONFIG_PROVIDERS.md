# Configuration Provider Best Practices

## Overview

The modular framework now provides four types of configuration providers, each optimized for different use cases. This document explains when and how to use each provider type.

## Provider Types

### 1. StdConfigProvider (Shared Reference)

**What it does:** Returns the SAME reference on every `GetConfig()` call.

**Use cases:**
- Simple single-instance applications
- When you explicitly need shared mutable config
- Legacy compatibility

**Thread safety:** ❌ NOT thread-safe for modifications

**Performance:** ⚡️ Excellent (no overhead)

**Example:**
```go
cfg := &MyConfig{Host: "localhost", Port: 8080}
provider := modular.NewStdConfigProvider(cfg)

// Both get the same reference
cfg1 := provider.GetConfig().(*MyConfig)
cfg2 := provider.GetConfig().(*MyConfig)
// cfg1 == cfg2 (same pointer)
```

**⚠️ WARNING:** Modifications by any consumer affect ALL other consumers!

---

### 2. IsolatedConfigProvider (Complete Isolation)

**What it does:** Returns a deep copy on EVERY `GetConfig()` call.

**Use cases:**
- ✅ Test isolation (RECOMMENDED for tests)
- Multi-tenant applications requiring per-tenant isolation
- Defensive programming where modules might mutate configs

**Thread safety:** ✅ Thread-safe (each call gets independent copy)

**Performance:** 🐌 Slower (deep copy on every access)

**Example:**
```go
cfg := &MyConfig{Host: "localhost", Port: 8080}
provider := modular.NewIsolatedConfigProvider(cfg)

// Each call returns a completely independent copy
copy1 := provider.GetConfig().(*MyConfig)
copy2 := provider.GetConfig().(*MyConfig)
// copy1 != copy2 (different pointers)

copy1.Port = 9090  // Does NOT affect copy2
```

**✅ BEST FOR:** Test scenarios to prevent config pollution between test runs.

---

### 3. ImmutableConfigProvider (Atomic Operations)

**What it does:** Stores config in `atomic.Value` for lock-free concurrent reads.

**Use cases:**
- ✅ Production applications (RECOMMENDED for production)
- High-performance read-heavy workloads
- Configuration hot-reloading with atomic swaps
- Multi-tenant applications with shared config

**Thread safety:** ✅ Fully thread-safe with atomic operations

**Performance:** ⚡️ Excellent (lock-free reads, no copies)

**Example:**
```go
cfg := &MyConfig{Host: "localhost", Port: 8080}
provider := modular.NewImmutableConfigProvider(cfg)

// Thread-safe reads from multiple goroutines
config := provider.GetConfig().(*MyConfig)

// Atomic update (useful for config hot-reload)
newCfg := &MyConfig{Host: "example.com", Port: 443}
provider.UpdateConfig(newCfg)
```

**✅ BEST FOR:** Production concurrent scenarios with high read throughput.

---

### 4. CopyOnWriteConfigProvider (Explicit Copy Control)

**What it does:** Returns original for reads, provides explicit method for getting mutable copies.

**Use cases:**
- Modules that need to apply defensive modifications
- When you want explicit control over when copies are made
- Scenarios requiring both read-only and mutable access

**Thread safety:** ✅ Thread-safe with RWMutex

**Performance:** 🚀 Good (only copies when explicitly requested)

**Example:**
```go
cfg := &MyConfig{Host: "localhost", Port: 8080}
provider := modular.NewCopyOnWriteConfigProvider(cfg)

// Read-only access (no copy, fast)
readCfg := provider.GetConfig().(*MyConfig)

// Need to modify? Get a mutable copy
mutableCfg, err := provider.GetMutableConfig()
if err == nil {
    cfg := mutableCfg.(*MyConfig)
    cfg.Port = 9090  // Safe to modify, won't affect others
}

// Update the original (e.g., for hot-reload)
newCfg := &MyConfig{Host: "example.com", Port: 443}
provider.UpdateOriginal(newCfg)
```

**✅ BEST FOR:** Modules that occasionally need to make defensive config modifications.

---

## Decision Matrix

| Scenario | Recommended Provider | Rationale |
|----------|---------------------|-----------|
| **Unit Tests** | `IsolatedConfigProvider` | Prevents test pollution |
| **Integration Tests** | `IsolatedConfigProvider` | Ensures test isolation |
| **Production Single-Threaded** | `StdConfigProvider` | Simplest, no overhead |
| **Production Multi-Threaded** | `ImmutableConfigProvider` | Lock-free, thread-safe |
| **Config Hot-Reload** | `ImmutableConfigProvider` | Atomic updates |
| **Multi-Tenant Apps** | `IsolatedConfigProvider` or `ImmutableConfigProvider` | Depends on isolation needs |
| **Defensive Modules** | `CopyOnWriteConfigProvider` | Explicit copy control |

---

## Best Practices

### 1. ❌ Don't Modify Configs In-Place

**Bad:**
```go
func (m *MyModule) Init(app modular.Application) error {
    cfg := app.GetConfig().(*MyConfig)
    if cfg.Port == 0 {
        cfg.Port = 8080  // ❌ Modifies shared config!
    }
    return nil
}
```

**Good:**
```go
func (m *MyModule) Init(app modular.Application) error {
    cfg := app.GetConfig().(*MyConfig)
    if cfg.Port == 0 {
        // Use defaults without mutation
        m.port = 8080
    } else {
        m.port = cfg.Port
    }
    return nil
}
```

### 2. ✅ Use Isolated Providers in Tests

**Good:**
```go
func TestMyModule(t *testing.T) {
    cfg := &MyConfig{Host: "localhost", Port: 8080}

    // Each test gets isolated config
    provider := modular.NewIsolatedConfigProvider(cfg)

    app := modular.NewApplication(
        modular.NewStdConfigProvider(provider),
        logger,
    )

    // Test won't pollute other tests
}
```

### 3. ✅ Use Immutable Providers in Production

**Good:**
```go
func main() {
    cfg := loadConfigFromFile()

    // Thread-safe for concurrent access
    provider := modular.NewImmutableConfigProvider(cfg)

    app := modular.NewApplication(
        modular.NewStdConfigProvider(provider),
        logger,
    )

    // Can hot-reload config atomically
    go watchConfigChanges(func(newCfg *Config) {
        provider.UpdateConfig(newCfg)
    })
}
```

### 4. ✅ Use Copy-On-Write for Defensive Modifications

**Good:**
```go
func (m *MyModule) Init(app modular.Application) error {
    cowProvider := app.GetConfig().(*modular.CopyOnWriteConfigProvider)

    // Get mutable copy for safe modifications
    mutableCfg, err := cowProvider.GetMutableConfig()
    if err != nil {
        return err
    }

    cfg := mutableCfg.(*MyConfig)
    // Safe to modify - won't affect other modules
    cfg.Port = normalizePort(cfg.Port)

    m.config = cfg
    return nil
}
```

---

## Performance Comparison

Based on benchmarks (see `config_provider_test.go`):

```
BenchmarkConfigProviders/StdConfigProvider-10                  ⚡️ ~1-2 ns/op
BenchmarkConfigProviders/ImmutableConfigProvider-10            ⚡️ ~3-5 ns/op
BenchmarkConfigProviders/CopyOnWriteConfigProvider_Read-10     🚀 ~10-20 ns/op
BenchmarkConfigProviders/IsolatedConfigProvider-10             🐌 ~500-2000 ns/op
BenchmarkConfigProviders/CopyOnWriteConfigProvider_Mutable-10  🐌 ~500-2000 ns/op
```

**Key Takeaways:**
- `StdConfigProvider` is fastest but unsafe
- `ImmutableConfigProvider` has minimal overhead with full safety
- `IsolatedConfigProvider` is slower but provides complete isolation
- `CopyOnWriteConfigProvider` is fast for reads, slower for mutable copies

---

## Migration Guide

### From StdConfigProvider to IsolatedConfigProvider (Tests)

**Before:**
```go
cfg := &MyConfig{}
provider := modular.NewStdConfigProvider(cfg)
```

**After:**
```go
cfg := &MyConfig{}
provider := modular.NewIsolatedConfigProvider(cfg)
```

### From StdConfigProvider to ImmutableConfigProvider (Production)

**Before:**
```go
cfg := loadConfig()
provider := modular.NewStdConfigProvider(cfg)
app := modular.NewApplication(provider, logger)
```

**After:**
```go
cfg := loadConfig()
provider := modular.NewImmutableConfigProvider(cfg)
app := modular.NewApplication(provider, logger)

// Optional: hot-reload support
provider.UpdateConfig(newCfg)
```

---

## Deep Copy Utility

The framework also exports a utility function for manually creating deep copies:

```go
originalCfg := &MyConfig{
    Host: "localhost",
    Tags: []string{"a", "b"},
    Metadata: map[string]string{"key": "value"},
}

// Create a deep copy
copiedCfg, err := modular.DeepCopyConfig(originalCfg)
if err != nil {
    // Handle error
}

// Modifications to copy don't affect original
copy := copiedCfg.(*MyConfig)
copy.Tags[0] = "modified"  // Original remains unchanged
```

This is useful when you need manual control over config copying outside of providers.

---

## Multi-Tenant Configuration

The framework provides specialized support for multi-tenant configurations with built-in isolation:

### TenantConfigProvider with Isolation

**For complete tenant isolation:**
```go
defaultCfg := &MyConfig{Host: "localhost", Port: 8080}

// Each tenant gets isolated copies of configs
tcp := modular.NewTenantConfigProviderWithIsolation(defaultCfg)

// Set isolated config for tenant1
tenant1Cfg := &DatabaseConfig{Host: "tenant1-db.example.com"}
tcp.SetTenantConfigIsolated("tenant1", "database", tenant1Cfg)

// Set isolated config for tenant2
tenant2Cfg := &DatabaseConfig{Host: "tenant2-db.example.com"}
tcp.SetTenantConfigIsolated("tenant2", "database", tenant2Cfg)

// Each tenant gets completely isolated copies
provider, _ := tcp.GetTenantConfig("tenant1", "database")
cfg := provider.GetConfig().(*DatabaseConfig)
// Modifications to cfg won't affect tenant2 or the original
```

**✅ BEST FOR:** Multi-tenant SaaS applications requiring strict tenant isolation.

### TenantConfigProvider with Immutability

**For shared thread-safe configs:**
```go
// All tenants share immutable config (thread-safe)
tcp := modular.NewTenantConfigProviderImmutable(sharedCfg)

// Set immutable config for specific tenant
tcp.SetTenantConfigImmutable("tenant1", "cache", &CacheConfig{
    TTL: 60 * time.Second,
})
```

**✅ BEST FOR:** Multi-tenant apps where tenants share common config with thread-safe access.

### Mixed Provider Types

You can mix different provider types for different tenants:

```go
tcp := modular.NewTenantConfigProvider(defaultProvider)

// Tenant1 needs isolation
tcp.SetTenantConfigIsolated("tenant1", "app", cfg1)

// Tenant2 needs thread-safe shared config
tcp.SetTenantConfigImmutable("tenant2", "app", cfg2)

// Tenant3 uses standard provider
tcp.SetTenantConfig("tenant3", "app", modular.NewStdConfigProvider(cfg3))
```

### Tenant Configuration Best Practices

1. **Use Isolation for Sensitive Data:**
   ```go
   // Customer-specific database configs should be isolated
   tcp.SetTenantConfigIsolated(tenantID, "database", dbConfig)
   ```

2. **Use Immutable for Shared Resources:**
   ```go
   // Shared cache settings can be immutable
   tcp.SetTenantConfigImmutable(tenantID, "cache", cacheConfig)
   ```

3. **Prevent Cross-Tenant Pollution:**
   ```go
   // ❌ Bad: Shared mutable config can leak between tenants
   tcp.SetTenantConfig(tenantID, "app", modular.NewStdConfigProvider(cfg))

   // ✅ Good: Isolated configs prevent cross-tenant pollution
   tcp.SetTenantConfigIsolated(tenantID, "app", cfg)
   ```

---

## Related Documentation

- [CONFIG_ISOLATION_ARCHITECTURE.md](CONFIG_ISOLATION_ARCHITECTURE.md) - Problem analysis
- [CLAUDE.md](CLAUDE.md) - Development guidelines
- [AGENTS.md](AGENTS.md) - Architecture overview

---

## Summary

Choose your configuration provider based on your needs:

| Priority | Choose This |
|----------|-------------|
| **Test Isolation** | `IsolatedConfigProvider` |
| **Production Performance** | `ImmutableConfigProvider` |
| **Defensive Modules** | `CopyOnWriteConfigProvider` |
| **Simple/Legacy** | `StdConfigProvider` (with caution) |

**Default Recommendation:** Use `IsolatedConfigProvider` for tests and `ImmutableConfigProvider` for production.
