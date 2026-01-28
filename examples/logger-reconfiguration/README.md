# Logger Reconfiguration Example

This example demonstrates how to use the `OnConfigLoaded` hook to reconfigure the logger based on configuration file settings before modules initialize.

## Problem

When applications need to configure the logger based on settings loaded from configuration files, modules that initialize during `app.Init()` receive and cache a reference to the initially-provided logger before any configuration-based reconfiguration can occur.

## Solution

The `OnConfigLoaded` hook allows you to reconfigure dependencies (like the logger) after configuration is loaded but before modules are initialized. This ensures that all modules receive the correctly configured logger.

## Features Demonstrated

- ✅ Logger reconfiguration based on configuration settings
- ✅ Hook execution after config loading, before module initialization
- ✅ Modules receiving the reconfigured logger
- ✅ Support for different log formats (text, JSON)
- ✅ Support for different log levels (debug, info, warn, error)

## Running the Example

### With Text Format (Default)

```bash
# Use text format with info level (default)
go run main.go
```

### With JSON Format

```bash
# Edit config.yaml to set logFormat: json
go run main.go
```

### With Debug Level

```bash
# Edit config.yaml to set logLevel: debug
go run main.go
```

### Using Environment Variables

Environment variables override the config file:

```bash
# Override format to JSON via environment
LOGFORMAT=json go run main.go

# Override level to debug via environment
LOGLEVEL=debug go run main.go

# Override both
LOGFORMAT=json LOGLEVEL=debug go run main.go
```

## Expected Output

When run with `logFormat: json` and `logLevel: debug`, you should see:

```
{"time":"...","level":"INFO","msg":"Starting application with initial logger"}
{"time":"...","level":"INFO","msg":"Logger reconfigured from configuration","format":"json","level":"debug"}
{"time":"...","level":"INFO","msg":"LoggingModule initialized","module":"logging"}
{"time":"...","level":"DEBUG","msg":"This debug message will only appear if log level is debug","module":"logging"}
{"time":"...","level":"INFO","msg":"ServiceModule initialized","module":"service","status":"ready"}
{"time":"...","level":"DEBUG","msg":"Service module debug information","feature":"logger_reconfiguration","working":true}
{"time":"...","level":"INFO","msg":"Application initialized successfully"}
{"time":"...","level":"DEBUG","msg":"Debug logging is enabled","config_loaded":true}

=== Logger Reconfiguration Example Complete ===
The logger was successfully reconfigured based on configuration before modules initialized
All modules received the reconfigured logger instance
```

## Key Points

1. **Hook Registration**: The `OnConfigLoaded` hook is registered using `modular.WithOnConfigLoaded()` in the application builder
2. **Timing**: The hook executes after configuration is loaded but before any module `Init()` methods are called
3. **Logger Access**: The hook can access the loaded configuration via `app.ConfigProvider().GetConfig()`
4. **Logger Reconfiguration**: The hook creates a new logger based on configuration and calls `app.SetLogger(newLogger)`
5. **Module Caching**: Modules cache the logger during their `Init()` method and receive the reconfigured instance

## Configuration File

The `config.yaml` file controls logger behavior:

```yaml
# Logger format: text or json
logFormat: json

# Log level: debug, info, warn, error
logLevel: debug
```

## Code Structure

- `main.go`: Application setup with OnConfigLoaded hook
- `config.yaml`: Configuration file with logger settings
- `AppConfig`: Configuration struct with logger settings
- `LoggingModule`: Example module that caches the logger
- `ServiceModule`: Another module demonstrating logger usage

## What This Solves

Without the `OnConfigLoaded` hook, you would need to:
1. Read configuration files manually before creating the application
2. Duplicate config parsing logic
3. Bypass the framework's config feeder system
4. Maintain separate config reading code

With the `OnConfigLoaded` hook:
- ✅ Clean separation of concerns
- ✅ Leverage framework's config infrastructure
- ✅ Consistent configuration approach
- ✅ No code duplication

## Related Features

This pattern can be used for other config-driven dependencies:
- Metrics collectors configuration
- Tracing provider setup
- Database connection configuration
- Feature flag initialization
- Any dependency that needs config-based initialization before modules start
