# Testing Scenarios Example

This example demonstrates comprehensive testing scenarios for reverse proxy and API gateway functionality using the modular framework. It supports all common testing patterns needed for production-ready API gateway systems.

## Supported Testing Scenarios

### 1. Health Check Testing
- Backend availability monitoring
- Custom health endpoints per backend
- DNS resolution testing
- HTTP connectivity testing
- Configurable health check intervals and timeouts

### 2. Load Testing
- High-concurrency request handling
- Connection pooling validation
- Resource utilization monitoring
- Performance baseline establishment

### 3. Failover/Circuit Breaker Testing
- Backend failure simulation
- Circuit breaker state transitions
- Fallback behavior validation
- Recovery time testing

### 4. Feature Flag Testing
- A/B deployment testing
- Gradual rollout scenarios
- Tenant-specific feature flags
- Dynamic routing based on flags

### 5. Multi-Tenant Testing
- Tenant isolation validation
- Tenant-specific routing
- Cross-tenant security testing
- Configuration isolation

### 6. Security Testing
- Authentication testing
- Authorization validation
- Rate limiting testing
- Header security validation

### 7. Performance Testing
- Latency measurement
- Throughput testing
- Response time validation
- Caching effectiveness

### 8. Configuration Testing
- Dynamic configuration updates
- Configuration validation
- Environment-specific configs
- Hot reloading validation

### 9. Error Handling Testing
- Error propagation testing
- Custom error responses
- Retry mechanism testing
- Graceful degradation

### 10. Monitoring/Metrics Testing
- Metrics collection validation
- Log aggregation testing
- Performance metrics
- Health status reporting

## Quick Start

```bash
cd examples/testing-scenarios

# Build the application
go build -o testing-scenarios .

# Run with basic configuration
./testing-scenarios

# Run specific test scenario
./testing-scenarios --scenario health-check
./testing-scenarios --scenario load-test
./testing-scenarios --scenario failover
```

## Configuration

The example uses `config.yaml` for comprehensive configuration covering all testing scenarios:

```yaml
reverseproxy:
  # Multiple backend services for different test scenarios
  backend_services:
    primary: "http://localhost:9001"
    secondary: "http://localhost:9002"
    canary: "http://localhost:9003"
    legacy: "http://localhost:9004"
    monitoring: "http://localhost:9005"
  
  # Health check configuration
  health_check:
    enabled: true
    interval: "10s"
    timeout: "3s"
    
  # Circuit breaker configuration
  circuit_breaker:
    enabled: true
    failure_threshold: 3
    success_threshold: 2
    open_timeout: "30s"
    
  # Feature flag support
  feature_flags:
    enabled: true
    
  # Multi-tenant configuration
  tenant_id_header: "X-Tenant-ID"
  require_tenant_id: false
```

## Testing Scripts

Each scenario includes automated test scripts:

- `test-health-checks.sh` - Health check scenarios
- `test-load.sh` - Load testing scenarios  
- `test-failover.sh` - Failover and circuit breaker scenarios
- `test-feature-flags.sh` - Feature flag scenarios
- `test-multi-tenant.sh` - Multi-tenant scenarios
- `test-security.sh` - Security scenarios
- `test-performance.sh` - Performance scenarios
- `test-config.sh` - Configuration scenarios
- `test-errors.sh` - Error handling scenarios
- `test-monitoring.sh` - Monitoring scenarios
- `test-all.sh` - Run all test scenarios

## Architecture

```
Client → Testing Proxy → Scenario Selector → Backend Pool
           ↓                  ↓                ↓
      Monitoring         Feature Flags    Health Checks
           ↓                  ↓                ↓
      Metrics            Circuit Breaker   Load Balancer
```

## Use Cases

This example is designed to validate:
- Production readiness of reverse proxy configurations
- API gateway behavior under various conditions
- Performance characteristics and bottlenecks
- Security posture and threat response
- Monitoring and observability capabilities
- Multi-tenant isolation and routing
- Feature rollout and deployment strategies

## Running Individual Scenarios

Each scenario can be run independently for focused testing:

```bash
# Health check testing
./testing-scenarios --scenario=health-check --duration=60s

# Load testing with custom parameters
./testing-scenarios --scenario=load-test --connections=100 --duration=120s

# Failover testing with backend simulation
./testing-scenarios --scenario=failover --backend=primary --failure-rate=0.5

# Feature flag testing with tenant isolation
./testing-scenarios --scenario=feature-flags --tenant=test-tenant --flag=new-api

# Performance testing with detailed metrics
./testing-scenarios --scenario=performance --metrics=detailed --export=json
```

This comprehensive testing example ensures that your reverse proxy configuration is production-ready and handles all common operational scenarios.