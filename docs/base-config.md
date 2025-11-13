# Base Configuration Guide

The Modular framework supports a base configuration approach that allows you to manage configuration across multiple environments efficiently, minimizing duplication and simplifying environment-specific overrides.

## Overview

Base configuration provides a hierarchical configuration system where:

1. **Base Configuration**: Common settings shared across all environments
2. **Environment Overrides**: Environment-specific values that override base settings
3. **Deep Merging**: Intelligent merging of nested configuration structures

This approach follows the DRY (Don't Repeat Yourself) principle and makes it easy to maintain configurations across multiple deployment environments.

## Directory Structure

The recommended directory structure for base configuration is:

```
config/
├── base/
│   └── default.yaml              # Baseline config shared across all environments
└── environments/
    ├── prod/
    │   └── overrides.yaml        # Production-specific overrides
    ├── staging/
    │   └── overrides.yaml        # Staging-specific overrides
    └── dev/
        └── overrides.yaml        # Development-specific overrides
```

## How It Works

### Base Configuration File

The `config/base/default.yaml` file contains all shared configuration that applies across all environments. This typically includes:

- Application structure and defaults
- Module configurations with sensible defaults
- Common service endpoints
- Default feature flags
- Standard timeouts and limits

**Example: `config/base/default.yaml`**

```yaml
app:
  name: "My Application"
  version: "1.0.0"

database:
  driver: "postgres"
  port: 5432
  max_connections: 25
  timeout: 30s
  ssl_mode: "require"

server:
  host: "localhost"
  port: 8080
  read_timeout: 30s
  write_timeout: 30s

features:
  caching: true
  debug: false
  logging: true
  metrics: false

external_services:
  redis:
    enabled: false
    port: 6379
  rabbitmq:
    enabled: false
    port: 5672
```

### Environment Override Files

Each environment directory contains an `overrides.yaml` file that specifies only the values that differ from the base configuration. The framework performs deep merging, so you only need to specify what changes.

**Example: `config/environments/dev/overrides.yaml`**

```yaml
database:
  host: "localhost"
  name: "dev_db"
  password: "dev_password"

server:
  port: 8080

features:
  debug: true
  caching: false
```

**Example: `config/environments/staging/overrides.yaml`**

```yaml
database:
  host: "staging-db.internal"
  name: "staging_db"
  password: "staging_password"

server:
  port: 443

features:
  metrics: true

external_services:
  redis:
    enabled: true
    host: "staging-redis.internal"
```

**Example: `config/environments/prod/overrides.yaml`**

```yaml
database:
  host: "prod-db.example.com"
  name: "prod_db"
  password: "secure_prod_password"

server:
  port: 443

features:
  debug: false
  metrics: true

external_services:
  redis:
    enabled: true
    host: "prod-redis.example.com"
  rabbitmq:
    enabled: true
    host: "prod-rabbitmq.example.com"
```

## Using Base Configuration

### In Your Application

To enable base configuration support in your application:

```go
package main

import (
    "github.com/CrisisTextLine/modular"
    "log/slog"
    "os"
)

func main() {
    // Get environment from environment variable or command line
    environment := os.Getenv("APP_ENVIRONMENT")
    if environment == "" {
        environment = "dev" // Default to dev
    }
    
    // Set base configuration support
    modular.SetBaseConfig("config", environment)
    
    // Create logger
    logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
    
    // Create application config provider
    config := &AppConfig{}
    configProvider := modular.NewStdConfigProvider(config)
    
    // Create application
    app := modular.NewStdApplication(configProvider, logger)
    
    // Register modules...
    
    // Run application
    if err := app.Run(); err != nil {
        logger.Error("Application error", "error", err)
        os.Exit(1)
    }
}
```

### Environment Selection

You can specify the environment in several ways:

**1. Command Line Argument:**
```bash
./myapp prod
./myapp staging
./myapp dev
```

**2. Environment Variables:**
```bash
APP_ENVIRONMENT=prod ./myapp
ENVIRONMENT=staging ./myapp
ENV=dev ./myapp
```

**3. Configuration File:**
```yaml
app:
  environment: prod
```

## Deep Merging Behavior

The framework performs intelligent deep merging of configuration structures:

### Simple Values
Simple values in overrides completely replace base values:

**Base:**
```yaml
server:
  port: 8080
```

**Override:**
```yaml
server:
  port: 443
```

**Result:**
```yaml
server:
  port: 443
```

### Nested Objects
Nested objects are merged recursively:

**Base:**
```yaml
database:
  host: "localhost"
  port: 5432
  max_connections: 25
```

**Override:**
```yaml
database:
  host: "prod-db.example.com"
  password: "secure_password"
```

**Result:**
```yaml
database:
  host: "prod-db.example.com"      # Overridden
  port: 5432                        # From base
  max_connections: 25               # From base
  password: "secure_password"       # Added
```

### Maps
Map fields merge by key:

**Base:**
```yaml
features:
  caching: true
  debug: false
  logging: true
```

**Override:**
```yaml
features:
  debug: true
  metrics: true
```

**Result:**
```yaml
features:
  caching: true       # From base
  debug: true         # Overridden
  logging: true       # From base
  metrics: true       # Added
```

### Arrays
Arrays in overrides completely replace base arrays (no merging):

**Base:**
```yaml
allowed_origins:
  - "http://localhost:3000"
  - "http://localhost:8080"
```

**Override:**
```yaml
allowed_origins:
  - "https://app.example.com"
```

**Result:**
```yaml
allowed_origins:
  - "https://app.example.com"
```

## Tenant Configuration with Base Config

Base configuration works seamlessly with the framework's multi-tenancy support. You can combine base/environment configs with tenant-specific overrides:

```
config/
├── base/
│   └── default.yaml
├── environments/
│   └── prod/
│       └── overrides.yaml
└── tenants/
    ├── tenant-a.yaml
    └── tenant-b.yaml
```

The merging order is:
1. Base configuration
2. Environment overrides
3. Tenant-specific overrides

This allows you to:
- Define common settings in base
- Set environment-specific values (dev/staging/prod)
- Override specific values per tenant

## Working Example

The framework includes a complete working example in `examples/base-config-example/`:

```bash
cd examples/base-config-example

# Run with different environments
go run main.go dev
go run main.go staging
go run main.go prod

# Or use environment variables
APP_ENVIRONMENT=prod go run main.go
```

The example demonstrates:
- Base configuration with common settings
- Environment-specific overrides
- Deep merging of configurations
- Runtime environment selection

See [examples/base-config-example/README.md](../examples/base-config-example/README.md) for full documentation.

## Best Practices

### 1. Keep Base Configuration Complete
Your base configuration should include all possible configuration fields with sensible defaults. This makes it clear what can be configured.

### 2. Override Only What Changes
Environment overrides should only specify values that differ from base. This keeps overrides small and easy to understand.

### 3. Use Descriptive Environment Names
Use clear, consistent environment names:
- `dev` or `development`
- `staging` or `stage`
- `prod` or `production`

### 4. Document Environment Differences
Add comments to override files explaining why values differ:

```yaml
# Production uses managed database service
database:
  host: "prod-db.example.com"
  
# Production requires HTTPS
server:
  port: 443
```

### 5. Validate Configurations
Use configuration validation to catch errors early:

```go
type AppConfig struct {
    Environment string `yaml:"environment" required:"true"`
    Database    DatabaseConfig `yaml:"database"`
}

func (c *AppConfig) Validate() error {
    validEnvs := map[string]bool{"dev": true, "staging": true, "prod": true}
    if !validEnvs[c.Environment] {
        return fmt.Errorf("invalid environment: %s", c.Environment)
    }
    return nil
}
```

### 6. Secure Sensitive Values
Never commit sensitive values to version control:
- Use environment variables for secrets
- Use secret management services (AWS Secrets Manager, HashiCorp Vault)
- Keep production passwords in secure storage

### 7. Test Configuration Loading
Write tests to verify configuration loading:

```go
func TestConfigurationLoading(t *testing.T) {
    modular.SetBaseConfig("config", "prod")
    
    config := &AppConfig{}
    provider := modular.NewStdConfigProvider(config)
    
    // Verify merged configuration
    assert.Equal(t, "prod-db.example.com", config.Database.Host)
    assert.Equal(t, true, config.Features.Metrics)
}
```

## Migration Guide

### Migrating from Single Config Files

If you currently use a single config file per environment:

**Before:**
```
config/
├── dev.yaml
├── staging.yaml
└── prod.yaml
```

**After:**
```
config/
├── base/
│   └── default.yaml      # Common settings from all files
└── environments/
    ├── dev/
    │   └── overrides.yaml  # Only dev-specific values
    ├── staging/
    │   └── overrides.yaml  # Only staging-specific values
    └── prod/
        └── overrides.yaml  # Only prod-specific values
```

**Steps:**

1. **Create directory structure:**
   ```bash
   mkdir -p config/base config/environments/{dev,staging,prod}
   ```

2. **Extract common settings:**
   - Identify settings that are the same across all environments
   - Move these to `config/base/default.yaml`

3. **Create environment overrides:**
   - For each environment, create `config/environments/{env}/overrides.yaml`
   - Include only values that differ from base

4. **Update application code:**
   ```go
   // Add before creating application
   modular.SetBaseConfig("config", environment)
   ```

5. **Test each environment:**
   ```bash
   APP_ENVIRONMENT=dev go run main.go
   APP_ENVIRONMENT=staging go run main.go
   APP_ENVIRONMENT=prod go run main.go
   ```

6. **Remove old config files** once verified

## Troubleshooting

### Configuration Not Loading

**Problem:** Base configuration not being loaded

**Solution:**
- Ensure `modular.SetBaseConfig()` is called before creating the application
- Verify directory structure matches expected layout
- Check environment variable is set correctly

### Values Not Overriding

**Problem:** Environment overrides not taking effect

**Solution:**
- Verify YAML structure matches base configuration
- Check for typos in field names
- Ensure YAML indentation is correct
- Test with debug logging enabled

### Missing Required Fields

**Problem:** Required fields showing as missing

**Solution:**
- Ensure base configuration includes all required fields
- Verify required fields have values in base or environment override
- Check `required:"true"` tags are on correct fields

## Summary

Base configuration support provides:

- **DRY Principle**: Define common configuration once
- **Environment Management**: Easy management of environment-specific settings
- **Deep Merging**: Intelligent merging of nested structures
- **Maintainability**: Clear separation of base and environment-specific configs
- **Flexibility**: Works with tenant-aware configurations

For a complete working example, see [examples/base-config-example/](../examples/base-config-example/).
