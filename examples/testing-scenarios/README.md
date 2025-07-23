# Testing Scenarios Example

This example demonstrates comprehensive testing scenarios for reverse proxy and API gateway functionality using the modular framework. It supports all common testing patterns needed for production-ready API gateway systems.

## Supported Testing Scenarios

### 1. Health Check Testing ✅
- Backend availability monitoring
- Custom health endpoints per backend  
- DNS resolution testing
- HTTP connectivity testing
- Configurable health check intervals and timeouts

### 2. Load Testing ✅
- High-concurrency request handling
- Connection pooling validation
- Resource utilization monitoring
- Performance baseline establishment

### 3. Failover/Circuit Breaker Testing ✅
- Backend failure simulation
- Circuit breaker state transitions
- Fallback behavior validation
- Recovery time testing

### 4. Feature Flag Testing ✅
- A/B deployment testing
- Gradual rollout scenarios
- Tenant-specific feature flags
- Dynamic routing based on flags

### 5. Multi-Tenant Testing ✅
- Tenant isolation validation
- Tenant-specific routing
- Cross-tenant security testing
- Configuration isolation

### 6. Security Testing ✅
- Authentication testing
- Authorization validation
- Rate limiting testing
- Header security validation

### 7. Performance Testing ✅
- Latency measurement
- Throughput testing
- Response time validation
- Caching effectiveness

### 8. Configuration Testing ✅
- Dynamic configuration updates
- Configuration validation
- Environment-specific configs
- Hot reloading validation

### 9. Error Handling Testing ✅
- Error propagation testing
- Custom error responses
- Retry mechanism testing
- Graceful degradation

### 10. Monitoring/Metrics Testing ✅
- Metrics collection validation
- Log aggregation testing
- Performance metrics
- Health status reporting

## Quick Start

```bash
cd examples/testing-scenarios

# Build the application
go build -o testing-scenarios .

# Run demonstration of all key scenarios
./demo.sh

# Run with basic configuration
./testing-scenarios

# Run specific test scenario
./testing-scenarios --scenario health-check
./testing-scenarios --scenario load-test
./testing-scenarios --scenario failover
```

## Individual Scenario Testing

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

## Automated Test Scripts

Each scenario includes automated test scripts:

- `demo.sh` - **Quick demonstration of all key scenarios**
- `test-all.sh` - Comprehensive test suite for all scenarios
- `test-health-checks.sh` - Health check scenarios
- `test-load.sh` - Load testing scenarios  
- `test-feature-flags.sh` - Feature flag scenarios

### Running Automated Tests

```bash
# Quick demonstration (recommended first run)
./demo.sh

# Comprehensive testing
./test-all.sh

# Specific scenario testing
./test-health-checks.sh
./test-load.sh --requests 200 --concurrency 20
./test-feature-flags.sh

# All tests with custom parameters
./test-all.sh --verbose --timeout 10
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
    unstable: "http://localhost:9006"    # For circuit breaker testing
    slow: "http://localhost:9007"        # For performance testing
  
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
  route_configs:
    "/api/v1/*":
      feature_flag_id: "api-v1-enabled"
      alternative_backend: "legacy"
    
  # Multi-tenant configuration
  tenant_id_header: "X-Tenant-ID"
  require_tenant_id: false
```

## Architecture

```
Client → Testing Proxy → Scenario Selector → Backend Pool
           ↓                  ↓                ↓
      Monitoring         Feature Flags    Health Checks
           ↓                  ↓                ↓
      Metrics            Circuit Breaker   Load Balancer
```

## Mock Backend System

The application automatically starts 7 mock backends:

- **Primary** (port 9001): Main backend for standard testing
- **Secondary** (port 9002): Secondary backend for failover testing
- **Canary** (port 9003): Canary backend for feature flag testing
- **Legacy** (port 9004): Legacy backend with `/status` endpoint
- **Monitoring** (port 9005): Monitoring backend with metrics
- **Unstable** (port 9006): Unstable backend for circuit breaker testing
- **Slow** (port 9007): Slow backend for performance testing

Each backend can be configured with:
- Custom failure rates
- Response delays
- Different health endpoints
- Request counting and metrics

## Testing Features

### Health Check Testing
- Tests all backend health endpoints
- Validates health check routing through proxy
- Tests tenant-specific health checks
- Monitors health check stability over time

### Load Testing
- Sequential and concurrent request testing
- Configurable request counts and concurrency
- Response time measurement
- Success rate calculation
- Throughput measurement

### Failover Testing  
- Simulates backend failures
- Tests circuit breaker behavior
- Validates fallback mechanisms
- Tests recovery scenarios

### Feature Flag Testing
- Tests enabled/disabled routing
- Tenant-specific feature flags
- Dynamic flag changes
- Fallback behavior validation

### Multi-Tenant Testing
- Tenant isolation validation
- Tenant-specific routing
- Concurrent tenant testing
- Default behavior testing

## Production Readiness Validation

This example validates:
- ✅ High availability configurations
- ✅ Performance characteristics and bottlenecks
- ✅ Security posture and threat response
- ✅ Monitoring and observability capabilities
- ✅ Multi-tenant isolation and routing
- ✅ Feature rollout and deployment strategies
- ✅ Error handling and recovery mechanisms
- ✅ Circuit breaker and failover behavior

## Use Cases

Perfect for validating:
- **API Gateway Deployments**: Ensure production readiness
- **Performance Tuning**: Identify bottlenecks and optimize settings
- **Resilience Testing**: Validate failure handling and recovery
- **Multi-Tenant Systems**: Ensure proper isolation and routing
- **Feature Rollouts**: Test gradual deployment strategies
- **Monitoring Setup**: Validate observability and alerting

## Example Output

```bash
$ ./demo.sh
╔══════════════════════════════════════════════════════════════╗
║           Testing Scenarios Demonstration               ║
╚══════════════════════════════════════════════════════════════╝

Test 1: Health Check Scenarios
  General health check... ✓ PASS
  API v1 health... ✓ PASS
  Legacy health... ✓ PASS

Test 2: Multi-Tenant Scenarios  
  Alpha tenant... ✓ PASS
  Beta tenant... ✓ PASS
  No tenant (default)... ✓ PASS

Test 3: Feature Flag Scenarios
  API v1 with feature flag... ✓ PASS
  API v2 routing... ✓ PASS
  Canary endpoint... ✓ PASS

✓ All scenarios completed successfully
```

This comprehensive testing example ensures that your reverse proxy configuration is production-ready and handles all common operational scenarios.