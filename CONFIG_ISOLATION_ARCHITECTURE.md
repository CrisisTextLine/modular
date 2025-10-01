# Config Isolation Architecture Analysis

## Critical Issues Identified

### 1. StdConfigProvider Stores Configs by Reference

**Current Implementation:**
```go
type StdConfigProvider struct {
    cfg any
}

func (s *StdConfigProvider) GetConfig() any {
    return s.cfg  // Returns the SAME reference every time
}
```

**Problem:** All consumers of a config provider receive the **same pointer**. Modifications by any consumer affect all others.

### 2. No Framework-Level Config Protection

The framework has deep copy functionality (`deepCopyValue`, `createTempConfigDeep`) but:
- These functions are **package-private** (not exported)
- They're only used for temporary processing during validation
- **Not used to protect configs from mutation**

### 3. Module.Init() Modifies Configs In-Place

Example from `reverseproxy/module.go:482-487`:
```go
func (m *ReverseProxyModule) validateConfig() error {
    // ...
    if m.config.CacheEnabled && m.config.CacheTTL <= 0 {
        m.app.Logger().Warn("Cache is enabled but CacheTTL is not set, using default of 60s")
        m.config.CacheTTL = 60 * time.Second  // MODIFIES THE CONFIG IN-PLACE
    }
    // ...
}
```

**Impact:** This modification affects the config provider's stored reference, potentially affecting other modules or subsequent test runs.

## Why This is a Problem

### In Production
1. **Multi-tenant applications**: Tenant A's config modifications could bleed into Tenant B
2. **Module initialization order**: Later modules might see modified configs from earlier modules
3. **Config reloading**: Reloading configs might inherit stale modifications

### In Tests
1. **Test isolation failures**: Exactly what we observed - tests pollute each other's configs
2. **Flaky tests**: Tests pass/fail depending on execution order
3. **Hard to debug**: The source of pollution is far from where symptoms appear

## Architectural Solutions

### Option 1: Config Providers Return Deep Copies (Recommended)

**Make StdConfigProvider return a copy:**
```go
func (s *StdConfigProvider) GetConfig() any {
    // Return a deep copy instead of the original reference
    copied, _, err := createTempConfigDeep(s.cfg)
    if err != nil {
        // Fallback to original if copy fails
        return s.cfg
    }
    return copied
}
```

**Pros:**
- Automatic protection for all configs
- No code changes needed in modules
- Works for existing applications

**Cons:**
- Performance overhead (copying on every GetConfig call)
- Need to export createTempConfigDeep
- May break applications that rely on shared mutation (though that's a bug)

### Option 2: Immutable Config Pattern

**Make configs immutable after initialization:**
```go
type ImmutableConfigProvider struct {
    cfg any
    frozen bool
}

func (p *ImmutableConfigProvider) GetConfig() any {
    if p.frozen {
        // Return a deep copy since config is frozen
        return deepCopy(p.cfg)
    }
    return p.cfg
}

func (p *ImmutableConfigProvider) Freeze() {
    p.frozen = true
}
```

**Pros:**
- Clear semantics - configs can't change after freezing
- Performance - only copy when needed
- Explicit control

**Cons:**
- Requires application code changes
- Modules must be careful not to modify configs
- Backward compatibility issues

### Option 3: Copy-on-Write Config Provider

**Lazy copy when config would be modified:**
```go
type CowConfigProvider struct {
    original any
    copy     any
    dirty    bool
}

func (p *CowConfigProvider) GetConfig() any {
    if !p.dirty {
        return p.original
    }
    return p.copy
}

func (p *CowConfigProvider) GetMutableConfig() any {
    if !p.dirty {
        p.copy = deepCopy(p.original)
        p.dirty = true
    }
    return p.copy
}
```

**Pros:**
- Performance - only copy when needed
- Explicit intent - GetMutableConfig() vs GetConfig()
- Tracks modification state

**Cons:**
- Two different methods - API complexity
- Requires module authors to use GetMutableConfig() when modifying
- Still allows shared mutation if everyone uses GetMutableConfig()

### Option 4: Module Gets Deep Copy During Init

**Framework copies config before passing to module:**
```go
func (app *StdApplication) Init() error {
    for _, module := range app.modules {
        cfg, err := app.GetConfigSection(module.Name())
        if err != nil {
            return err
        }

        // Create a deep copy for this specific module instance
        cfgCopy, _, err := createTempConfigDeep(cfg.GetConfig())
        if err != nil {
            return err
        }

        // Pass the copy to the module
        module.SetConfig(cfgCopy)
    }
}
```

**Pros:**
- Each module gets its own config copy
- No config provider changes
- Isolated module configs

**Cons:**
- Breaks shared config scenarios (if intentional)
- Module interface changes (need SetConfig method)
- Doesn't solve test isolation (same provider used across tests)

## Recommended Approach

**Hybrid: Option 1 + Export Deep Copy Utilities**

1. **Export deep copy functions:**
   ```go
   // Export for use by modules and tests
   func DeepCopyConfig(cfg any) (any, error) {
       copied, _, err := createTempConfigDeep(cfg)
       return copied, err
   }
   ```

2. **Add option to StdConfigProvider:**
   ```go
   type StdConfigProvider struct {
       cfg      any
       copyMode ConfigCopyMode
   }

   type ConfigCopyMode int
   const (
       NoCopy ConfigCopyMode = iota  // Current behavior (reference)
       DeepCopy                       // Return deep copy
   )

   func NewStdConfigProvider(cfg any) *StdConfigProvider {
       return &StdConfigProvider{cfg: cfg, copyMode: NoCopy}
   }

   func NewIsolatedConfigProvider(cfg any) *StdConfigProvider {
       return &StdConfigProvider{cfg: cfg, copyMode: DeepCopy}
   }

   func (s *StdConfigProvider) GetConfig() any {
       if s.copyMode == DeepCopy {
           copied, _, _ := createTempConfigDeep(s.cfg)
           return copied
       }
       return s.cfg
   }
   ```

3. **Best Practices Documentation:**
   - Tests should use `NewIsolatedConfigProvider`
   - Production apps should consider isolation needs
   - Modules should NOT modify configs in-place

## Action Items

1. ✅ **Remove test-specific deepCopyConfig** - use framework's deep copy
2. **Export DeepCopyConfig function** from config_provider.go
3. **Add NewIsolatedConfigProvider** for test isolation
4. **Update BDD tests** to use NewIsolatedConfigProvider
5. **Add lint rule** to warn about config field mutations
6. **Document config isolation best practices**
7. **Consider making DeepCopy the default** in a future major version

## Migration Path

### Phase 1 (Current - Backward Compatible)
- Export deep copy utilities
- Add NewIsolatedConfigProvider
- Fix tests to use isolated providers
- Document the issue and best practices

### Phase 2 (Next Minor Version)
- Add deprecation warnings for direct config mutation
- Add metrics to track config provider usage patterns
- Provide migration guide

### Phase 3 (Next Major Version)
- Make DeepCopy the default behavior
- Remove NoCopy mode (breaking change)
- Full config immutability guarantee
