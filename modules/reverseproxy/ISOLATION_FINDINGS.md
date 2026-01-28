# Config Isolation Investigation Findings

## Summary
The test hanging issue is caused by config state pollution that **only occurs when running the full test suite** (100+ non-BDD tests + BDD tests together). When BDD tests run in isolation, config isolation works correctly.

## Evidence

### 1. BDD Tests in Isolation: ✅ WORKS
When running `go test -v -run "TestReverseProxyModuleBDD"`:
```
🔍 DEBUG: iHaveAReverseProxyWithSpecificCacheTTLConfigured() starting (Cache TTL behavior scenario)
🔍 DEBUG: Set CacheTTL=1s (Cache TTL behavior signature)
...
🔍 DEBUG: cachedResponsesAgeBeyondTTL reading CacheTTL=1s, will sleep for 1.5s
```
**Result**: Correctly uses 1s CacheTTL, sleeps for 1.5s, test completes quickly.

### 2. Full Test Suite: ❌ FAILS
When running `go test -v .` (all 175+ tests):
```
🔍 DEBUG: iHaveAReverseProxyWithSpecificCacheTTLConfigured() starting (Cache TTL behavior scenario)
🔍 DEBUG: Set CacheTTL=1s (Cache TTL behavior signature)
...
🔍 DEBUG: After setupApplicationWithConfig, ctx.config.CacheTTL=5m0s
...
🔍 DEBUG: cachedResponsesAgeBeyondTTL reading CacheTTL=5m0s, will sleep for 5m0.5s
```
**Result**: CacheTTL changes from 1s to 5m0s (300s), causing 5-minute sleep and test timeout.

### 3. Isolation Tests: ✅ PASS
All isolation tests pass, confirming:
- Module instances are isolated ✅
- Config struct pointers are isolated ✅
- BDD context instances are isolated ✅
- `resetContext()` properly clears state ✅

## Root Cause

The config is being modified **inside `setupApplicationWithConfig()`** between setting the config and reading it back. Specifically:

1. Scenario sets `ctx.config.CacheTTL = 1 * time.Second`
2. Calls `setupApplicationWithConfig()`
3. After the call, `ctx.config.CacheTTL = 300 * time.Second`

The config pointer (`ctx.config`) stays the same, but the **object it points to is being modified**.

## Hypothesis

When all tests run together, one of the 100+ non-BDD tests:
1. Creates a global or static config with CacheTTL=300s (or 120s)
2. This config is cached or stored somewhere in the modular framework
3. When `setupApplicationWithConfig()` calls `ctx.app.Init()`, the framework's config loading mechanism somehow merges or overwrites the BDD test's config with the cached config

## Evidence of Config Modification Point

Debug logs show the modification happens during `setupApplicationWithConfig()`:
```
🔍 DEBUG: Set CacheTTL=1s (Cache TTL behavior signature)
🔍 DEBUG: ctx.config pointer = 0x14000a002c8
... (setupApplicationWithConfig() runs)
🔍 DEBUG: After setupApplicationWithConfig, ctx.config.CacheTTL=5m0s
```

## Tests with 300s CacheTTL

Files that set CacheTTL to 300 seconds:
- `bdd_caching_tenant_test.go:42` - `iHaveAReverseProxyWithCachingEnabled()` - **Response caching** scenario (now 337s)
- `bdd_tenant_caching_override_test.go:161` - Another test

Files that set CacheTTL to 120 seconds:
- `config_merge_test.go:34`
- `module_test.go:812`

## Next Steps

1. **Investigate the modular framework's config loading mechanism** to understand how it handles config sections during `Init()`
2. **Check if there's a config cache** or global registry that's polluting state
3. **Add deep copy logic** to ensure config structs are never shared between test runs
4. **Fix `setupApplicationWithConfig()`** to ensure the config passed in is the config that gets used

## Temporary Workaround

Use shorter CacheTTL values in all tests (< 5 seconds) to prevent timeouts even if config bleeding occurs.
