# Modular Go Framework

A structured way to create modular applications in Go with module lifecycle management, dependency injection, configuration management, and multi-tenancy support.

## Development Commands

### Testing
```bash
# Core framework tests
go test ./... -v

# Module tests (each module has its own go.mod)
for module in modules/*/; do
  if [ -f "$module/go.mod" ]; then
    echo "Testing $module"
    cd "$module" && go test ./... -v && cd -
  fi
done

# Example tests (each example has its own go.mod)
for example in examples/*/; do
  if [ -f "$example/go.mod" ]; then
    echo "Testing $example"
    cd "$example" && go test ./... -v && cd -
  fi
done

# CLI tests
cd cmd/modcli && go test ./... -v

# Parallel BDD tests (faster feedback)
chmod +x scripts/run-module-bdd-parallel.sh
scripts/run-module-bdd-parallel.sh 6
```

### Code Quality
```bash
# Format code
go fmt ./...

# Lint code
golangci-lint run

# Run single test
go test -run TestSpecificFunction ./path/to/package -v
```

### Build
```bash
# Build CLI tool
cd cmd/modcli && go build -o modcli

# Build examples (each has own go.mod)
cd examples/basic-app && GOWORK=off go build
```

## Architecture Overview

### Core Framework Structure
- **Root Directory**: Core application framework (`application.go`, `module.go`, service registry, configuration system)
- **`feeders/`**: Configuration feeders for various sources (env, yaml, json, toml)
- **`modules/`**: Pre-built reusable modules with individual go.mod files
- **`examples/`**: Complete working applications demonstrating usage patterns
- **`cmd/modcli/`**: CLI tool for generating modules and configurations

### Key Concepts

#### Module System
- All modules implement the core `Module` interface
- Optional interfaces: `Startable`, `Stoppable`, `TenantAwareModule`
- Dependency resolution through service registry
- Interface-based and name-based service matching

#### Service Registry
- Dependency injection system
- Services can be matched by name or interface compatibility
- Support for required and optional dependencies

#### Configuration Management
- Support for YAML, JSON, TOML formats
- Validation with struct tags (`required`, `default`, `desc`)
- Custom validation via `ConfigValidator` interface
- Multi-tenant configuration isolation

#### Multi-Tenancy
- Tenant-aware modules and configurations
- Context-based tenant propagation
- Isolated per-tenant resources and settings

### Available Modules
- **auth**: JWT, sessions, password hashing, OAuth2/OIDC
- **cache**: Redis and in-memory caching
- **chimux**: Chi router integration
- **database**: Multi-driver database connectivity with migrations
- **eventbus**: Pub/sub messaging and event handling
- **eventlogger**: Structured logging for Observer pattern events with CloudEvents
- **httpclient**: Configurable HTTP client
- **httpserver**: HTTP/HTTPS server with TLS
- **jsonschema**: JSON Schema validation services
- **letsencrypt**: SSL/TLS certificate automation
- **logmasker**: Log data masking and sanitization
- **reverseproxy**: Load balancing and circuit breaker
- **scheduler**: Cron jobs and worker pools

### Testing Guidelines
- Use `app.SetConfigFeeders()` instead of mutating global `modular.ConfigFeeders`
- Prefer parallel tests with proper isolation
- Each module/example has independent tests due to separate go.mod files

## Development Workflow

1. **Module Development**: Implement `Module` interface, provide configuration, register services
2. **Configuration**: Use struct tags for validation, implement `ConfigValidator` for custom logic
3. **Testing**: Write unit tests, integration tests, and BDD tests where applicable
4. **Documentation**: Update module README files with usage examples

## Code Standards
- Go 1.25+ required (toolchain uses 1.25.0)
- Format with `gofmt`
- Lint with `golangci-lint run`
- All tests must pass before commit
- Follow established interface patterns
- Maintain backwards compatibility when possible