# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

@AGENTS.md

## Claude-Specific Instructions

### Testing Commands
- **Run core tests**: `go test ./... -v`
- **Run module tests**: `for module in modules/*/; do [ -f "$module/go.mod" ] && (cd "$module" && go test ./... -v); done`
- **Run example tests**: `for example in examples/*/; do [ -f "$example/go.mod" ] && (cd "$example" && go test ./... -v); done`
- **Run CLI tests**: `cd cmd/modcli && go test ./... -v`
- **Format code**: `go fmt ./...`
- **Lint code**: `golangci-lint run`

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
go test ./... -v
# Test modules
for module in modules/*/; do [ -f "$module/go.mod" ] && (cd "$module" && go test ./... -v); done
# Test examples
for example in examples/*/; do [ -f "$example/go.mod" ] && (cd "$example" && go test ./... -v); done
# Test CLI
cd cmd/modcli && go test ./... -v
```