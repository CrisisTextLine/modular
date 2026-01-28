# Configuration System

The Modular framework provides a flexible and powerful configuration system that supports:

- Multiple configuration sources (YAML, JSON, TOML, environment variables)
- Configuration validation with defaults and required fields
- Module-aware environment variable resolution
- Instance-aware configuration for managing multiple instances
- Custom validation logic
- Sample configuration generation

## Table of Contents

- [Configuration Providers](#configuration-providers)
- [Configuration Validation](#configuration-validation)
- [Configuration Feeders](#configuration-feeders)
- [Module-Aware Environment Variables](#module-aware-environment-variables)
- [Instance-Aware Configuration](#instance-aware-configuration)
- [Sample Configuration Generation](#sample-configuration-generation)

## Configuration Providers

Config Providers are responsible for supplying configuration values to modules. The basic interface is simple:

```go
type ConfigProvider interface {
    GetConfig() any
}
```

The standard implementation, `StdConfigProvider`, wraps a Go struct:

```go
config := &MyConfig{}
provider := modular.NewStdConfigProvider(config)
```

### Registering Configuration

Modules can register configuration sections during the configuration phase:

```go
func (m *MyModule) RegisterConfig(app modular.Application) error {
    m.config = &MyConfig{
        Port: 8080, // Default value
    }
    app.RegisterConfigSection(m.Name(), modular.NewStdConfigProvider(m.config))
    return nil
}
```

## Configuration Validation

Modular supports comprehensive configuration validation through struct tags and custom validation logic.

### Default Values

Default values are specified using the `default` struct tag:

```go
type ServerConfig struct {
    Host string `yaml:"host" default:"localhost"`
    Port int    `yaml:"port" default:"8080"`
}
```

These values are automatically applied during configuration loading if the field is empty or zero.

### Required Fields

Fields can be marked as required using the `required` tag:

```go
type DatabaseConfig struct {
    User     string `yaml:"user" required:"true"`
    Password string `yaml:"password" required:"true"`
}
```

If these fields are not provided, configuration loading will fail with an appropriate error.

### Field Descriptions

Use the `desc` tag to document configuration options:

```go
type AppConfig struct {
    Name    string `yaml:"name" default:"MyApp" desc:"Application name"`
    Version string `yaml:"version" required:"true" desc:"Application version"`
    Debug   bool   `yaml:"debug" default:"false" desc:"Enable debug mode"`
}
```

### Custom Validation Logic

For more complex validation, implement the `ConfigValidator` interface:

```go
type ConfigValidator interface {
    Validate() error
}
```

**Example:**

```go
func (c *ServerConfig) Validate() error {
    if c.Port < 1024 || c.Port > 65535 {
        return fmt.Errorf("%w: port must be between 1024 and 65535", 
            modular.ErrConfigValidationFailed)
    }
    return nil
}
```

## Configuration Feeders

Feeders provide a way to load configuration from different sources. The framework supports multiple feeder types that can be chained together.

### Available Feeders

```go
// YAML file
yamlFeeder := feeders.NewYAMLFeeder("config.yaml")

// JSON file
jsonFeeder := feeders.NewJSONFeeder("config.json")

// TOML file
tomlFeeder := feeders.NewTOMLFeeder("config.toml")

// Environment variables
envFeeder := feeders.NewEnvFeeder("MYAPP_")

// .env file
dotEnvFeeder := feeders.NewDotEnvFeeder(".env")
```

### Using Feeders

Apply feeders to configuration:

```go
config := &AppConfig{}

// Apply YAML feeder
err := yamlFeeder.Feed(config)
if err != nil {
    return err
}

// Override with environment variables
err = envFeeder.Feed(config)
if err != nil {
    return err
}
```

Multiple feeders can be chained, with later feeders overriding values from earlier ones.

### Feeder Priority

Configure the application to use multiple feeders in priority order:

```go
app.SetConfigFeeders([]modular.Feeder{
    feeders.NewYAMLFeeder("config.yaml"),  // Base configuration
    feeders.NewEnvFeeder("APP_"),          // Override with env vars
})
```

## Module-Aware Environment Variables

The framework includes intelligent environment variable resolution that automatically searches for module-specific environment variables to prevent naming conflicts between modules.

### How It Works

When a module registers configuration with `env` tags, the framework searches for environment variables in priority order:

1. `MODULENAME_ENV_VAR` (module name prefix - highest priority)
2. `ENV_VAR_MODULENAME` (module name suffix - medium priority)
3. `ENV_VAR` (original variable name - lowest priority)

This allows different modules to use the same configuration field names without conflicts.

### Example

Consider a module with this configuration:

```go
type DatabaseConfig struct {
    Host    string `env:"HOST"`
    Port    int    `env:"PORT"`
    Timeout int    `env:"TIMEOUT"`
}
```

The framework will search for environment variables in this order:

```bash
# For the database module's HOST field:
DATABASE_HOST=db.example.com          # Highest priority
HOST_DATABASE=alt.example.com         # Medium priority
HOST=fallback.example.com             # Lowest priority
```

### Benefits

- **No Naming Conflicts**: Different modules can use the same field names safely
- **Module-Specific Overrides**: Easily configure specific modules without affecting others
- **Backward Compatibility**: Existing environment variable configurations continue to work
- **Automatic Resolution**: No code changes required in modules
- **Predictable Patterns**: Consistent naming conventions across all modules

### Multiple Modules Example

```bash
# Database module configuration
DATABASE_HOST=db.internal.example.com
DATABASE_PORT=5432
DATABASE_TIMEOUT=120

# HTTP server module configuration
HTTPSERVER_HOST=api.external.example.com
HTTPSERVER_PORT=8080
HTTPSERVER_TIMEOUT=30

# Fallback values
HOST=localhost
PORT=8000
TIMEOUT=60
```

### Module Name Resolution

The module name used for environment variable prefixes comes from the module's `Name()` method and is automatically converted to uppercase:

- Module name `"database"` → Environment prefix `DATABASE_`
- Module name `"httpserver"` → Environment prefix `HTTPSERVER_`
- Module name `"reverseproxy"` → Environment prefix `REVERSEPROXY_`

## Instance-Aware Configuration

Instance-aware configuration allows you to manage multiple instances of the same configuration type using environment variables with instance-specific prefixes.

### Overview

Traditional configuration approaches struggle with multiple instances because they rely on fixed environment variable names. Instance-aware configuration solves this by using instance-specific prefixes:

```bash
# Single instance (backward compatible)
DRIVER=postgres
DSN=postgres://localhost/db

# Multiple instances with prefixes
DB_PRIMARY_DRIVER=postgres
DB_PRIMARY_DSN=postgres://localhost/primary
DB_SECONDARY_DRIVER=mysql
DB_SECONDARY_DSN=mysql://localhost/secondary
```

### InstanceAwareEnvFeeder

The `InstanceAwareEnvFeeder` handles environment variable feeding for multiple instances:

```go
// Create an instance-aware feeder with a prefix function
feeder := modular.NewInstanceAwareEnvFeeder(func(instanceKey string) string {
    return "DB_" + strings.ToUpper(instanceKey) + "_"
})

// Feed a single instance
config := &database.ConnectionConfig{}
err := feeder.FeedKey("primary", config)
// Looks for DB_PRIMARY_DRIVER, DB_PRIMARY_DSN, etc.
```

### InstanceAwareConfigProvider

The `InstanceAwareConfigProvider` wraps configuration objects and associates them with instance prefix functions:

```go
// Create instance-aware config provider
prefixFunc := func(instanceKey string) string {
    return "DB_" + strings.ToUpper(instanceKey) + "_"
}

config := &database.Config{
    Connections: map[string]database.ConnectionConfig{
        "primary":   {},
        "secondary": {},
    },
}

provider := modular.NewInstanceAwareConfigProvider(config, prefixFunc)
app.RegisterConfigSection("database", provider)
```

### Module Integration

Modules can implement the `InstanceAwareConfigSupport` interface to enable automatic instance-aware configuration:

```go
type InstanceAwareConfigSupport interface {
    GetInstanceConfigs() map[string]interface{}
}
```

**Example implementation:**

```go
func (c *Config) GetInstanceConfigs() map[string]interface{} {
    instances := make(map[string]interface{})
    for name, connection := range c.Connections {
        connCopy := connection
        instances[name] = &connCopy
    }
    return instances
}
```

### Environment Variable Patterns

Instance-aware configuration supports consistent naming patterns:

```bash
# Pattern: <PREFIX><INSTANCE_KEY>_<FIELD_NAME>

# Database connections
DB_PRIMARY_DRIVER=postgres
DB_PRIMARY_DSN=postgres://user:pass@localhost/primary
DB_PRIMARY_MAX_OPEN_CONNECTIONS=25

DB_SECONDARY_DRIVER=mysql
DB_SECONDARY_DSN=mysql://user:pass@localhost/secondary
DB_SECONDARY_MAX_OPEN_CONNECTIONS=10

# Cache instances
CACHE_SESSION_DRIVER=redis
CACHE_SESSION_ADDR=localhost:6379
CACHE_SESSION_DB=0

CACHE_OBJECTS_DRIVER=redis
CACHE_OBJECTS_ADDR=localhost:6379
CACHE_OBJECTS_DB=1
```

### Configuration Struct Requirements

Configuration structs must have `env` struct tags:

```go
type ConnectionConfig struct {
    Driver             string `env:"DRIVER"`
    DSN                string `env:"DSN"`
    MaxOpenConnections int    `env:"MAX_OPEN_CONNECTIONS"`
    MaxIdleConnections int    `env:"MAX_IDLE_CONNECTIONS"`
}
```

The `env` tag specifies the environment variable name that will be combined with the instance prefix.

### Complete Example

```go
package main

import (
    "github.com/CrisisTextLine/modular"
    "github.com/CrisisTextLine/modular/modules/database"
    "os"
)

func main() {
    // Set up environment variables
    os.Setenv("DB_PRIMARY_DRIVER", "postgres")
    os.Setenv("DB_PRIMARY_DSN", "postgres://localhost/primary")
    os.Setenv("DB_SECONDARY_DRIVER", "mysql")
    os.Setenv("DB_SECONDARY_DSN", "mysql://localhost/secondary")

    // Create application
    app := modular.NewStdApplication(
        modular.NewStdConfigProvider(&AppConfig{}),
        logger,
    )

    // Register database module
    app.RegisterModule(database.NewModule())

    // Initialize
    err := app.Init()
    if err != nil {
        panic(err)
    }

    // Access different connections
    var dbManager *database.Module
    app.GetService("database.manager", &dbManager)
    
    primaryDB, _ := dbManager.GetConnection("primary")
    secondaryDB, _ := dbManager.GetConnection("secondary")
}
```

### Best Practices

1. **Consistent Naming**: Use consistent prefix patterns across your application
   ```bash
   DB_<INSTANCE>_<FIELD>
   CACHE_<INSTANCE>_<FIELD>
   HTTP_<INSTANCE>_<FIELD>
   ```

2. **Uppercase Instance Keys**: Convert instance keys to uppercase for environment variables
   ```go
   prefixFunc := func(instanceKey string) string {
       return "DB_" + strings.ToUpper(instanceKey) + "_"
   }
   ```

3. **Environment Variable Documentation**: Document expected environment variables

4. **Graceful Defaults**: Provide sensible defaults for non-critical configuration
   ```go
   type ConnectionConfig struct {
       Driver             string `env:"DRIVER"`
       DSN                string `env:"DSN"`
       MaxOpenConnections int    `env:"MAX_OPEN_CONNECTIONS" default:"25"`
   }
   ```

5. **Validation**: Implement validation for instance configurations
   ```go
   func (c *ConnectionConfig) Validate() error {
       if c.Driver == "" {
           return errors.New("driver is required")
       }
       if c.DSN == "" {
           return errors.New("DSN is required")
       }
       return nil
   }
   ```

## Sample Configuration Generation

Modular can generate sample configuration files with all default values pre-populated:

```go
// Generate a sample configuration file
cfg := &AppConfig{}
err := modular.SaveSampleConfig(cfg, "yaml", "config-sample.yaml")
if err != nil {
    log.Fatalf("Error generating sample config: %v", err)
}
```

### Supported Formats

- `yaml` - YAML format
- `json` - JSON format
- `toml` - TOML format

### Command-Line Integration

```go
func main() {
    // Generate sample config if requested
    if len(os.Args) > 1 && os.Args[1] == "--generate-config" {
        format := "yaml"
        if len(os.Args) > 2 {
            format = os.Args[2]
        }
        outputFile := "config-sample." + format
        if len(os.Args) > 3 {
            outputFile = os.Args[3]
        }
        
        cfg := &AppConfig{}
        if err := modular.SaveSampleConfig(cfg, format, outputFile); err != nil {
            fmt.Printf("Error: %v\n", err)
            os.Exit(1)
        }
        fmt.Printf("Sample config generated: %s\n", outputFile)
        os.Exit(0)
    }
    
    // Normal application startup...
}
```

### Usage

```bash
# Generate YAML sample
./myapp --generate-config yaml config-sample.yaml

# Generate JSON sample
./myapp --generate-config json config-sample.json

# Generate TOML sample
./myapp --generate-config toml config-sample.toml
```

## See Also

- [Base Configuration Guide](base-config.md) - Managing multi-environment configurations
- [Multi-Tenancy](multi-tenancy.md) - Tenant-aware configuration
- [Module Lifecycle](module-lifecycle.md) - Configuration in module lifecycle
