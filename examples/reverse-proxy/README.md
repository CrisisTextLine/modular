# Reverse Proxy Example

This example demonstrates the advanced composite routing capabilities of the modular framework's reverse proxy module, showcasing **all composite route response strategies** including the new **map/reduce patterns** for intelligent data aggregation, plus **custom response transformers**.

## What it demonstrates

### 🆕 Map/Reduce Composite Routes

The example now includes **map/reduce patterns** that enable sophisticated data aggregation:

1. **Sequential Map/Reduce** (`map-reduce` with `type: sequential`)
   - Query one backend for a list
   - Extract IDs or fields from the response
   - Send extracted data to another backend
   - Merge the enriched data back into the response
   - **Use case**: Conversation list enriched with follow-up information
   - **Example**: List conversations → extract IDs → query follow-ups → merge

2. **Parallel Map/Reduce** (`map-reduce` with `type: parallel`)
   - Query multiple backends in parallel
   - Join responses based on a common field (like ID)
   - Filter or merge results intelligently
   - **Use case**: Unified view across independent microservices
   - **Example**: Conversations + participants + activity → joined by ID

**See the [Map/Reduce Guide](../../modules/reverseproxy/MAPREDUCE_GUIDE.md) for complete documentation.**

### Traditional Composite Route Strategies

This example also demonstrates **three traditional strategies** for combining responses from multiple backend services:

1. **First-Success Strategy** (`first-success`)
   - Tries backends sequentially until one succeeds
   - Returns the first successful response
   - **Use case**: High-availability setups with primary and fallback backends
   - **Example**: Try primary server first, fall back to backup if unavailable

2. **Merge Strategy** (`merge`)
   - Executes all backend requests in parallel
   - Merges JSON responses from all backends into a single JSON object
   - **Use case**: Aggregating data from multiple microservices
   - **Example**: Fetch user data, orders, and preferences simultaneously and combine them

3. **Sequential Strategy** (`sequential`)
   - Executes requests one at a time in order
   - Returns the last successful response
   - **Use case**: Multi-step workflows where later steps may depend on earlier ones completing
   - **Example**: Auth → Processing → Finalization pipeline

### Custom Response Transformers

Beyond the built-in strategies, this example shows how to create **custom response transformers** that can:
- Intelligently merge data from multiple backends
- Augment response data from one backend with data from another
- Create entirely new response structures
- Apply business logic during response composition

The example includes a transformer that enriches user profile data with analytics information, demonstrating tight integration between backend responses.

### Additional Features

- **Tenant-Specific Routing**: Different tenants can have different default backend services
- **Response Header Rewriting**: Override and consolidate response headers (e.g., CORS headers)
- **Dynamic Response Modification**: Custom callback functions to modify response headers dynamically
- **Multiple Mock Backends**: 14 self-contained mock servers demonstrating real-world scenarios

## Architecture

```
                                    ┌─────────────────────────┐
                                    │   Composite Routes       │
                                    └─────────────────────────┘
                                              │
                    ┌─────────────────────────┼─────────────────────────┐
                    │                         │                         │
         ┌──────────▼──────────┐   ┌─────────▼────────┐   ┌───────────▼──────────┐
         │  First-Success      │   │     Merge        │   │    Sequential        │
         │  Strategy           │   │     Strategy     │   │    Strategy          │
         └──────────┬──────────┘   └─────────┬────────┘   └───────────┬──────────┘
                    │                         │                         │
         Try in order, return     Parallel requests,       Sequential requests,
         first successful         merge all JSON          return last successful
                    │                         │                         │
         ┌──────────▼──────────┐   ┌─────────▼────────┐   ┌───────────▼──────────┐
         │  Primary Backend    │   │  Users Backend   │   │  Auth Backend        │
         │  Fallback Backend   │   │  Orders Backend  │   │  Processing Backend  │
         └─────────────────────┘   │  Prefs Backend   │   │  Final Backend       │
                                   └──────────────────┘   └──────────────────────┘
```

## Running the Example

```bash
cd examples/reverse-proxy

# Build the application
go build -o reverse-proxy .

# Run the reverse proxy server
./reverse-proxy
```

The server will start on `localhost:8080` and automatically launch **14 mock backend servers** on ports 9001-9014 to demonstrate the various strategies.

## Testing the Composite Route Strategies

### Quick Test Script

Test all composite routes (including map/reduce) with one command:

```bash
./test-mapreduce.sh
```

### 🆕 Map/Reduce Strategies

#### 1. Sequential Map/Reduce: Conversations with Follow-ups

Test the conversation list enrichment scenario from the requirements:

```bash
curl http://localhost:8080/api/composite/mapreduce/conversations
```

**What happens**:
1. Proxy queries conversations backend → gets list of 5 conversations
2. Proxy extracts conversation IDs: `["conv1", "conv2", "conv3", "conv4", "conv5"]`
3. Proxy sends POST to followups backend `/bulk` with extracted IDs
4. Followups backend returns follow-up info for 3 conversations
5. Proxy enriches the original response with follow-up data

**Expected response structure**:
```json
{
  "conversations": [
    {"id": "conv1", "title": "Customer Support Request", "status": "open", ...},
    {"id": "conv2", "title": "Billing Inquiry", "status": "active", ...},
    ...
  ],
  "followup_info": {
    "followups": [
      {"conversation_id": "conv1", "is_followup": true, "parent_id": "conv_original_1", ...},
      {"conversation_id": "conv3", "is_followup": false},
      {"conversation_id": "conv4", "is_followup": true, "parent_id": "conv_original_4", ...}
    ]
  }
}
```

#### 2. Parallel Map/Reduce: Join by ID

Test parallel backend queries with ID-based joining:

```bash
curl http://localhost:8080/api/composite/mapreduce/parallel-join
```

**What happens**:
1. Proxy queries conversations and participants backends in parallel
2. Proxy receives:
   - Conversations: 5 items with IDs
   - Participants: 3 items with matching IDs
3. Proxy joins by `id` field
4. Returns unified array with participant info merged into each conversation

**Expected response** (array of joined items):
```json
[
  {
    "id": "conv1",
    "title": "Customer Support Request",
    "status": "open",
    "participant_info": {
      "id": "conv1",
      "participants": ["user123", "agent456"],
      "participant_count": 2
    }
  },
  {
    "id": "conv2",
    "title": "Billing Inquiry",
    "status": "active",
    "participant_info": {
      "id": "conv2",
      "participants": ["user789"],
      "participant_count": 1
    }
  },
  ...
]
```

### Traditional Composite Strategies

#### 1. First-Success Strategy

Test the high-availability pattern with automatic fallback:

```bash
# Make multiple requests to see primary/fallback behavior
# Primary backend fails every 3rd request, triggering fallback
curl http://localhost:8080/api/composite/first-success
curl http://localhost:8080/api/composite/first-success
curl http://localhost:8080/api/composite/first-success  # This one will use fallback
curl http://localhost:8080/api/composite/first-success
```

**Expected behavior**:
- Most requests: Primary backend response
- Every 3rd request: Fallback backend response (when primary "fails")

**Response example (primary)**:
```json
{"backend":"primary-backend","status":"success","request_count":1}
```

**Response example (fallback)**:
```json
{"backend":"fallback-backend","status":"success","message":"fallback activated"}
```

### 2. Merge Strategy

Test parallel data aggregation from multiple microservices:

```bash
# Single request fetches data from 3 backends in parallel
curl http://localhost:8080/api/composite/merge
```

**Expected response** (merged JSON from all backends):
```json
{
  "users-backend": {
    "user_id": 123,
    "username": "john_doe",
    "email": "john@example.com"
  },
  "orders-backend": {
    "total_orders": 42,
    "recent_orders": [
      {"id": 1, "amount": 99.99},
      {"id": 2, "amount": 149.99}
    ]
  },
  "preferences-backend": {
    "theme": "dark",
    "notifications": true,
    "language": "en"
  }
}
```

### 3. Sequential Strategy

Test ordered execution of multi-step workflows:

```bash
# Requests execute in order: auth → processing → finalization
curl http://localhost:8080/api/composite/sequential
```

**Expected response** (from the last backend in sequence):
```json
{
  "status": "completed",
  "result": "success",
  "message": "All steps completed",
  "step": "3_finalize"
}
```

### 4. Custom Response Transformer

Test intelligent data merging with custom business logic:

```bash
# Custom transformer enriches profile with analytics
curl http://localhost:8080/api/composite/profile-with-analytics
```

**Expected response** (custom merged structure):
```json
{
  "profile": {
    "user_id": 789,
    "name": "Alice Smith",
    "bio": "Software Engineer",
    "joined": "2023-01-15"
  },
  "analytics": {
    "page_views": 1523,
    "session_duration": "45m",
    "last_login": "2024-01-20T10:30:00Z"
  },
  "total_page_views": 1523,
  "enriched": true,
  "timestamp": "2024-01-20T15:30:45Z"
}
```

Notice how the transformer:
- Keeps both responses organized under `profile` and `analytics` keys
- Extracts `page_views` to the top level as `total_page_views`
- Adds metadata like `enriched` flag and `timestamp`

## Testing Tenant-Specific Routing

The example also maintains backward compatibility with tenant-specific routing:

```bash
# Test tenant1 routing (goes to tenant1-backend)
curl -H "X-Tenant-ID: tenant1" http://localhost:8080/test

# Test tenant2 routing (goes to tenant2-backend)  
curl -H "X-Tenant-ID: tenant2" http://localhost:8080/test

# Test without tenant header (goes to global-default)
curl http://localhost:8080/test
```

Or run the comprehensive test script:
```bash
./test-tenant-routing.sh
```

## Configuration

The reverse proxy is configured through `config.yaml`. Here's how each strategy is configured:

### First-Success Strategy Configuration

```yaml
composite_routes:
  "/api/composite/first-success":
    pattern: "/api/composite/first-success"
    backends:
      - "primary-backend"      # Tried first
      - "fallback-backend"     # Tried if primary fails
    strategy: "first-success"
```

### Merge Strategy Configuration

```yaml
composite_routes:
  "/api/composite/merge":
    pattern: "/api/composite/merge"
    backends:
      - "users-backend"        # All executed in parallel
      - "orders-backend"
      - "preferences-backend"
    strategy: "merge"
```

### Sequential Strategy Configuration

```yaml
composite_routes:
  "/api/composite/sequential":
    pattern: "/api/composite/sequential"
    backends:
      - "auth-backend"         # Executed first
      - "processing-backend"   # Executed second
      - "finalization-backend" # Executed last (response returned)
    strategy: "sequential"
```

### Custom Response Transformer

Response transformers are set programmatically in `main.go`:

```go
proxyModule.SetResponseTransformer("/api/composite/profile-with-analytics", 
    func(responses map[string]*http.Response) (*http.Response, error) {
        // Custom logic to merge/transform responses
        // ...
        return customResponse, nil
    })
```

## Implementation Details

### Code Structure

The example consists of:

1. **main.go**: Application setup with:
   - Module registration and configuration
   - Custom response transformer implementation
   - 14 mock backend servers demonstrating various scenarios

2. **config.yaml**: Complete configuration showing:
   - All backend service definitions
   - All three composite route strategies
   - Response header rewriting rules

3. **README.md**: Comprehensive documentation

### Mock Backend Servers

The example includes 17 self-contained mock servers:

| Port | Backend | Purpose |
|------|---------|---------|
| 9001 | global-default | Default routing |
| 9002 | tenant1-backend | Tenant-specific routing |
| 9003 | tenant2-backend | Tenant-specific routing |
| 9004 | specific-api | CORS header override demo |
| 9005 | primary-backend | First-success: primary (occasionally fails) |
| 9006 | fallback-backend | First-success: backup |
| 9007 | users-backend | Merge: user data |
| 9008 | orders-backend | Merge: order data |
| 9009 | preferences-backend | Merge: preferences |
| 9010 | auth-backend | Sequential: step 1 (auth) |
| 9011 | processing-backend | Sequential: step 2 (process) |
| 9012 | finalization-backend | Sequential: step 3 (finalize) |
| 9013 | profile-backend | Transformer: profile data |
| 9014 | analytics-backend | Transformer: analytics data |
| 9015 | conversations-backend | 🆕 Map/reduce: conversation list |
| 9016 | followups-backend | 🆕 Map/reduce: follow-up data |
| 9017 | participants-backend | 🆕 Map/reduce: participant info |

## Use Cases

### First-Success Strategy

Perfect for:
- **High-availability setups**: Primary and backup backend servers
- **Graceful degradation**: Try expensive operation first, fall back to cached/simpler version
- **Multi-region deployments**: Try nearest region first, fall back to others
- **Service migration**: Try new service, fall back to legacy if unavailable

### Merge Strategy

Perfect for:
- **Microservice aggregation**: Combine data from multiple services (users, orders, inventory)
- **Dashboard APIs**: Fetch multiple metrics/stats in parallel
- **Profile enrichment**: User + preferences + settings in one call
- **Report generation**: Gather data from multiple sources simultaneously

### Sequential Strategy

Perfect for:
- **Multi-step workflows**: Auth → business logic → finalization
- **Pipeline processing**: Data validation → transformation → storage
- **Dependent operations**: Where later steps need earlier steps to complete
- **State transitions**: Ordered status changes

### Custom Transformers

Perfect for:
- **Complex data merging**: Business logic for combining responses
- **Data augmentation**: Enrich response A with calculated fields from response B
- **Response reshaping**: Create custom structures from backend data
- **Cross-service enrichment**: Add related data from multiple sources

## Key Modules Used

1. **ChiMux Module**: Provides HTTP routing with Chi router and CORS middleware
2. **ReverseProxy Module**: Handles composite routing with multiple strategies
3. **HTTPServer Module**: Manages the HTTP server lifecycle

## Benefits of Composite Routing

1. **Reduced Client Complexity**: One API call instead of multiple
2. **Improved Performance**: Parallel requests reduce total latency
3. **Better Reliability**: Automatic failover and retries
4. **Simplified Orchestration**: Server-side composition vs. client-side
5. **Consistent Error Handling**: Centralized handling of partial failures
6. **Flexible Integration**: Mix different backend types and protocols

## Advanced Topics

### Circuit Breakers

The reverse proxy module includes circuit breaker support for all strategies. When a backend repeatedly fails, the circuit breaker opens and stops sending requests to that backend temporarily, allowing it to recover.

### Response Caching

Responses can be cached based on configurable TTL values, reducing load on backend services.

### Health Checking

Backends can be automatically health-checked, and unhealthy backends are temporarily removed from rotation.

### Feature Flags

Composite routes can be controlled by feature flags, allowing gradual rollout or A/B testing of new aggregation patterns.

## Troubleshooting

### All backends return 502

- Ensure all mock backends started successfully (check console output)
- Verify no port conflicts (ports 9001-9014 should be available)
- Check that the proxy server is running on port 8080

### Merge strategy returns partial data

- This is normal if some backends fail
- The merge strategy combines all successful responses
- Check individual backend logs for errors

### Sequential strategy returns unexpected backend

- Sequential returns the **last** successful response
- If early backends fail, you'll get a response from a later backend
- This is by design for resilience

## Learn More

- [Reverse Proxy Module Documentation](../../modules/reverseproxy/README.md)
- [Path Rewriting Guide](../../modules/reverseproxy/PATH_REWRITING_GUIDE.md)
- [Per-Backend Configuration](../../modules/reverseproxy/PER_BACKEND_CONFIGURATION_GUIDE.md)
