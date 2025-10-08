# NATS EventBus Example

This example demonstrates using the EventBus module with NATS as the messaging backend. It shows two Go services communicating asynchronously through NATS pub/sub messaging.

## Overview

This example includes:
- **NATS Server**: Lightweight, high-performance messaging system
- **Publisher Service**: Simulates order creation and publishes events
- **Subscriber Service**: Listens to events and processes them
- **Event Types**: Orders, Analytics, and Notifications

## Architecture

```
┌─────────────────────┐         ┌──────────────┐         ┌──────────────────────┐
│ Publisher Service   │         │              │         │ Subscriber Service   │
│                     │         │     NATS     │         │                      │
│ - Order Events     ├────────►│   Message    │────────►│ - Order Handler      │
│ - Analytics Events │         │   Broker     │         │ - Analytics Handler  │
│ - Notifications    │         │              │         │ - Notification       │
└─────────────────────┘         └──────────────┘         └──────────────────────┘
```

## Features Demonstrated

- **NATS Integration**: EventBus module configured with NATS engine
- **Pub/Sub Pattern**: Multiple services communicating via events
- **Wildcard Subscriptions**: Subscribe to topic patterns like `order.*`
- **Async Processing**: Asynchronous event handlers for heavy operations
- **Graceful Shutdown**: Proper cleanup and service termination
- **Health Monitoring**: NATS health checks and monitoring

## Prerequisites

- Go 1.25 or later
- Docker and Docker Compose
- Ports available:
  - `4222` - NATS client connections
  - `8222` - NATS HTTP monitoring
  - `6222` - NATS cluster connections

## Quick Start

### Option 1: Use the Run Script (Recommended)

```bash
# Run the complete demo (starts NATS and the application)
./run-demo.sh run

# Or start services separately
./run-demo.sh start    # Start NATS
go run .               # Run the application

# Stop services when done
./run-demo.sh stop

# Clean up everything
./run-demo.sh cleanup
```

### Option 2: Manual Setup

1. **Start NATS server**:
   ```bash
   docker-compose up -d
   ```

2. **Wait for NATS to be ready**:
   ```bash
   # Check NATS health
   curl http://localhost:8222/healthz
   ```

3. **Run the application**:
   ```bash
   go run main.go
   ```

4. **Stop services when done**:
   ```bash
   docker-compose down
   ```

## Configuration

The example uses the following NATS configuration:

```yaml
eventbus:
  engines:
    - name: nats-primary
      type: nats
      config:
        url: "nats://localhost:4222"
        connectionName: "nats-eventbus-demo"
        maxReconnects: 10
        reconnectWait: 2
        allowReconnect: true
        pingInterval: 20
        maxPingsOut: 2
        subscribeTimeout: 5
```

### Configuration Options

- **url**: NATS server URL (default: `nats://localhost:4222`)
- **connectionName**: Client connection name for monitoring
- **maxReconnects**: Maximum reconnection attempts (0 = unlimited)
- **reconnectWait**: Wait time between reconnection attempts (seconds)
- **allowReconnect**: Enable automatic reconnection
- **pingInterval**: Interval for ping requests (seconds)
- **maxPingsOut**: Maximum outstanding pings before disconnect
- **subscribeTimeout**: Timeout for subscription operations (seconds)

## Event Flow

1. **Publisher Service** (runs every 3 seconds):
   - Publishes order creation events to `order.created`
   - Publishes analytics events to `analytics.order`
   - Publishes notification events to `notification.system`

2. **Subscriber Service**:
   - Listens to `order.*` (wildcard) - processes all order events
   - Listens to `analytics.*` (async) - records analytics
   - Listens to `notification.*` - sends notifications

## Expected Output

```
🚀 Started NATS EventBus Demo in development environment
📊 NATS EventBus Configuration:
  - NATS server: localhost:4222
  - All topics routed through NATS

🔍 Checking NATS service availability:
  ✅ NATS service is reachable on localhost:4222
  ✅ Ready for pub/sub messaging

📤 Publisher Service started
📨 Subscriber Service started
✅ All subscriptions active

🔄 Services are running. Press Ctrl+C to stop...

📤 [PUBLISHED] order.created: ORDER-1 (amount: $100.99)
📤 [PUBLISHED] analytics.order: ORDER-1
📨 [ORDER SERVICE] Processing order: ORDER-1
📨 [ANALYTICS SERVICE] Recording event: order_created

📤 [PUBLISHED] order.created: ORDER-2 (amount: $101.99)
📤 [PUBLISHED] analytics.order: ORDER-2
📤 [PUBLISHED] notification.system: Processed 2 orders
📨 [ORDER SERVICE] Processing order: ORDER-2
📨 [NOTIFICATION SERVICE] Sending notification: Processed 2 orders
📨 [ANALYTICS SERVICE] Recording event: order_created
```

## NATS Monitoring

Access NATS monitoring dashboard at: http://localhost:8222

Available endpoints:
- `/varz` - General server information
- `/connz` - Connection information
- `/routez` - Route information
- `/subsz` - Subscription information
- `/healthz` - Health check

Example monitoring commands:
```bash
# Check server health
curl http://localhost:8222/healthz

# View server info
curl http://localhost:8222/varz

# View connections
curl http://localhost:8222/connz

# View subscriptions
curl http://localhost:8222/subsz
```

## Troubleshooting

### NATS Service Not Available

If you see "❌ NATS service not reachable":

1. **Check if Docker is running**: `docker --version`
2. **Start NATS**: `./run-demo.sh start`
3. **Check service status**: `./run-demo.sh status`
4. **View logs**: `./run-demo.sh logs`

### Port Conflicts

If ports 4222, 8222, or 6222 are in use:

```bash
# Check what's using the ports
netstat -tlnp | grep :4222
netstat -tlnp | grep :8222

# Modify docker-compose.yml to use different ports
```

### Connection Errors

If you see connection errors:

1. **Verify NATS is healthy**: `curl http://localhost:8222/healthz`
2. **Check logs**: `docker logs nats-eventbus`
3. **Restart NATS**: `./run-demo.sh restart`

### Services Taking Too Long to Start

- NATS usually starts in 5-10 seconds
- Use `./run-demo.sh status` to monitor startup progress
- Check `./run-demo.sh logs` for any startup errors

## Key Concepts

### NATS vs Other Message Brokers

**NATS Advantages**:
- **Lightweight**: Minimal resource footprint
- **Fast**: High throughput and low latency
- **Simple**: Easy to deploy and operate
- **Cloud-Native**: Designed for distributed systems
- **Resilient**: Built-in reconnection and failover

**When to Use NATS**:
- Real-time messaging
- Microservices communication
- IoT applications
- Event streaming
- Service mesh data plane

### EventBus with NATS

The EventBus module abstracts NATS details:
- Automatic connection management
- Reconnection handling
- Topic pattern conversion (`.* → .>`)
- Graceful shutdown
- Error handling

### Topic Patterns

NATS uses hierarchical subjects:
```
order.created
order.updated
order.cancelled
user.registered
user.updated
```

Wildcards:
- `order.*` - matches `order.created`, `order.updated`, etc.
- `>` - matches everything (multi-level wildcard)

The EventBus automatically converts:
- `order.*` → `order.>` (NATS format)
- `*` → `>` (catch-all)

## Advanced Usage

### Authentication

To use NATS with authentication:

```yaml
eventbus:
  engines:
    - name: nats-primary
      type: nats
      config:
        url: "nats://localhost:4222"
        username: "myuser"
        password: "mypassword"
```

Or with token authentication:

```yaml
eventbus:
  engines:
    - name: nats-primary
      type: nats
      config:
        url: "nats://localhost:4222"
        token: "mytoken"
```

### Multiple NATS Servers

For high availability:

```yaml
eventbus:
  engines:
    - name: nats-primary
      type: nats
      config:
        url: "nats://server1:4222,nats://server2:4222,nats://server3:4222"
```

### JetStream (Persistent Messaging)

This example uses core NATS (in-memory). For persistent messaging with JetStream:

1. NATS is already started with JetStream enabled (`-js` flag)
2. Use JetStream for:
   - Message persistence
   - At-least-once delivery
   - Consumer groups
   - Stream replay

## Production Considerations

1. **High Availability**: Deploy NATS cluster with 3+ nodes
2. **Monitoring**: Use Prometheus/Grafana with NATS exporter
3. **Security**: Enable TLS and authentication
4. **Resource Limits**: Configure connection and subscription limits
5. **JetStream**: Use for critical events requiring persistence
6. **Observability**: Implement structured logging and tracing

## Testing

Build and test the example:

```bash
# Build
GOWORK=off go build -o nats-demo .

# Test (requires NATS running)
./run-demo.sh start
./nats-demo
```

## Cleaning Up

```bash
# Stop all services and clean up
./run-demo.sh cleanup

# Or manually
docker-compose down -v
rm -f nats-demo
```

## Learn More

- [NATS Documentation](https://docs.nats.io/)
- [NATS Go Client](https://github.com/nats-io/nats.go)
- [EventBus Module](../../modules/eventbus/README.md)
- [Modular Framework](../../README.md)

## Next Steps

1. Explore [multi-engine-eventbus](../multi-engine-eventbus) for using multiple backends
2. Try [observer-pattern](../observer-pattern) for event-driven architecture
3. Implement custom event handlers and error handling
4. Add message persistence with JetStream
5. Set up NATS clustering for production use
