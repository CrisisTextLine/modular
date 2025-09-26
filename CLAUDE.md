# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

@AGENTS.md

## Claude-Specific Instructions

### Testing Commands
- **Run core tests**: `go test ./... -v`
- **Run core tests with race detection**: `go test -race -v ./...`
- **Run module tests**: `for module in modules/*/; do [ -f "$module/go.mod" ] && (cd "$module" && go test ./... -v); done`
- **Run module tests with race detection**: `for module in modules/*/; do [ -f "$module/go.mod" ] && (cd "$module" && go test -race -v ./...); done`
- **Run example tests**: `for example in examples/*/; do [ -f "$example/go.mod" ] && (cd "$example" && go test ./... -v); done`
- **Run CLI tests**: `cd cmd/modcli && go test ./... -v`
- **Format code**: `go fmt ./...`
- **Lint code**: `golangci-lint run`

#### Enhanced Testing for test-runner Agent
When using the test-runner agent or running comprehensive test verification, always:

1. **Use race detection**: Add `-race` flag to catch race conditions
2. **Capture full output**: Don't limit output with `head`/`tail` - analyze complete results
3. **Look for panic indicators**:
   - "panic:" strings in output
   - "runtime error:" messages
   - Stack traces with "runtime.gopanic"
   - "WARNING:" messages indicating recovered panics
4. **Check for systemic failures**: Multiple tests failing with same error pattern indicates structural issues
5. **Verify BDD scenarios**: Ensure BDD tests execute logic rather than fail fast with initialization errors
6. **Calculate pass rates**: Provide clear metrics on test health (e.g., "366 passing, 33 failing = 92% pass rate")
7. **Distinguish infrastructure vs business logic failures**: Infrastructure panics/races are critical; business logic test failures are normal development work
8. **Report improvement trends**: Compare current results against previous runs to show progress
9. **CRITICAL - Zero tolerance for failures**: ANY test failure must trigger immediate agent assignment for fixing
10. **Mandatory escalation**: If any failures detected, immediately categorize and assign to specialized agents:
    - Race conditions → multi-tenant-specialist or go-module-expert
    - Router/service issues → dependency-resolver
    - Configuration problems → config-validator
    - BDD scenario failures → go-module-expert
11. **Continuous verification**: After any fix, re-run tests to verify resolution and catch any new issues
12. **Quality gate enforcement**: Do not consider work complete until 100% test success is achieved

### Architecture Notes

#### Multi-Module Structure
This is a Go workspace with multiple go.mod files:
- Root: Core framework (application.go, module.go, service registry)
- modules/*/: Each module has its own go.mod (auth, cache, database, etc.)
- examples/*/: Each example has its own go.mod (basic-app, reverse-proxy, etc.)
- cmd/modcli/: CLI tool with its own go.mod

When working in modules or examples, be aware you're in a separate Go module.

#### Service Registry Pattern
The framework uses dependency injection through a service registry:
- Services are registered via `ProvidesServices()` method
- Services are consumed via `RequiresServices()` and `Constructor()` pattern
- Both name-based and interface-based service matching supported

#### Configuration System
- Struct tags drive validation: `required:"true"`, `default:"value"`, `desc:"description"`
- Custom validation via `ConfigValidator` interface
- Multi-format support: YAML, JSON, TOML
- Per-application config feeders via `app.SetConfigFeeders()` (preferred over global)

#### Multi-Tenancy Support
- Context-based tenant propagation via `modular.TenantContext`
- Tenant-aware modules implement `TenantAwareModule` interface
- Per-tenant configuration isolation

### Development Guidelines

#### Module Development
When creating or modifying modules:
1. Implement the core `Module` interface
2. Use dependency injection pattern with service registry
3. Follow configuration validation patterns with struct tags
4. Write comprehensive tests (unit, integration, BDD where applicable)
5. Each module directory has its own go.mod file

#### Testing Best Practices
- Use `app.SetConfigFeeders()` for test isolation instead of mutating global `modular.ConfigFeeders`
- Tests can run in parallel when properly isolated
- Each module/example tests independently due to separate go.mod files

#### Before Committing
Always run this sequence:
```bash
go fmt ./...
golangci-lint run
go test -race -v ./...
# Test modules with race detection
for module in modules/*/; do [ -f "$module/go.mod" ] && (cd "$module" && go test -race -v ./...); done
# Test examples
for example in examples/*/; do [ -f "$example/go.mod" ] && (cd "$example" && go test -race -v ./...); done
# Test CLI
cd cmd/modcli && go test -race -v ./...
```

#### Debugging Test Failures
When tests fail with panics or race conditions:

1. **Nil map panics**: Check for uninitialized maps in test contexts - add `make(map[...])` initialization
2. **Nil pointer dereferences**: Verify application context and service injection in BDD tests
3. **Router panics**: Ensure test routers properly initialize their internal maps
4. **Race conditions**: Use `-race` flag and check for concurrent access to shared data structures

Common fixes:
- Add nil checks before map assignments: `if m == nil { m = make(map[string]string) }`
- Initialize test contexts properly in BDD scenarios
- Use panic recovery for external service calls in tests
- Ensure proper cleanup in test teardown methods