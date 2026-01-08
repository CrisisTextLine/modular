# Map/Reduce Composite Routes Guide

## Overview

The Map/Reduce strategy enables sophisticated data aggregation patterns where responses from one backend can feed into another, or where multiple backends' responses are intelligently merged based on common identifiers. This is essential for scenarios where you need to combine data from multiple microservices that don't directly communicate with each other.

## Use Cases

### 1. Sequential Pattern: List with Enrichment
**Problem**: You have a list endpoint that returns basic conversation data, but follow-up information is in a separate service. You want to enrich the conversation list with follow-up details in a single API call.

**Solution**: Use sequential map/reduce to:
1. Query conversations backend for the list
2. Extract conversation IDs from the response
3. Send those IDs to the follow-ups backend
4. Merge the follow-up data back into the conversation list

### 2. Parallel Pattern: Multi-Source Join
**Problem**: You need to display a unified view combining data from multiple independent services (e.g., conversation details, participant info, and activity logs), all keyed by a common ID.

**Solution**: Use parallel map/reduce to:
1. Query all backends concurrently
2. Join responses based on a common field (like `id` or `conversation_id`)
3. Return a unified response with all data merged per item

## Configuration

### Sequential Map/Reduce

```yaml
reverseproxy:
  backend_services:
    conversations: "http://conversations-service:8080"
    followups: "http://followups-service:8080"
  
  composite_routes:
    "/api/conversations":
      pattern: "/api/conversations"
      backends:
        - "conversations"
        - "followups"
      strategy: "map-reduce"
      map_reduce:
        type: "sequential"
        source_backend: "conversations"
        target_backend: "followups"
        mapping:
          extract_path: "conversations"        # Path in source response to extract from
          extract_field: "id"                  # Field to extract from each item
          target_request_field: "conversation_ids"  # Field name in target request
          target_request_path: "/bulk"         # Path to send request to on target
          target_request_method: "POST"        # HTTP method for target request
          merge_into_field: "followup_info"    # Where to place target data in result
        merge_strategy: "enrich"               # How to merge responses
        allow_empty_responses: true            # Return source data even if target fails
```

**Request Flow**:
```
Client → /api/conversations
  ↓
Proxy → conversations backend → {"conversations": [{"id": "1", ...}, {"id": "2", ...}]}
  ↓
Proxy extracts: ["1", "2"]
  ↓
Proxy → followups backend POST /bulk {"conversation_ids": ["1", "2"]}
  ↓
Proxy ← followups backend ← {"followups": [...]}
  ↓
Proxy merges and enriches
  ↓
Client ← {"conversations": [...], "followup_info": {...}}
```

### Parallel Map/Reduce with Join

```yaml
reverseproxy:
  backend_services:
    conversations: "http://conversations-service:8080"
    participants: "http://participants-service:8080"
    activity: "http://activity-service:8080"
  
  composite_routes:
    "/api/conversations/detailed":
      pattern: "/api/conversations/detailed"
      backends:
        - "conversations"
        - "participants"
        - "activity"
      strategy: "map-reduce"
      map_reduce:
        type: "parallel"
        backends:
          - "conversations"
          - "participants"
          - "activity"
        mapping:
          join_field: "conversation_id"        # Common field across all backends
          merge_into_field: "participant_info" # Where to nest merged data (optional)
        merge_strategy: "join"                 # Join by common field
        allow_empty_responses: true
        filter_on_empty: false                 # Keep items even without ancillary data
```

**Request Flow**:
```
Client → /api/conversations/detailed
  ↓
Proxy → conversations, participants, activity (in parallel)
  ↓
Proxy receives:
  - conversations: [{"conversation_id": "1", "title": "..."}, ...]
  - participants: [{"conversation_id": "1", "users": [...]}, ...]
  - activity: [{"conversation_id": "1", "last_activity": "..."}, ...]
  ↓
Proxy joins by conversation_id
  ↓
Client ← [{"conversation_id": "1", "title": "...", "users": [...], "last_activity": "..."}, ...]
```

## Mapping Configuration

### Extract Path
The `extract_path` uses dot notation to navigate nested JSON structures:

```yaml
# Source response:
{
  "data": {
    "items": [
      {"id": "1", "name": "Item 1"},
      {"id": "2", "name": "Item 2"}
    ]
  }
}

# Configuration:
extract_path: "data.items"  # Navigates to the items array
extract_field: "id"          # Extracts ["1", "2"]
```

### Merge Strategies

#### 1. **Enrich** (Sequential)
Adds target response data into the source response:
```json
// Source
{"user": {"id": 123, "name": "John"}}

// Target
{"analytics": {"views": 100}}

// Result
{
  "user": {"id": 123, "name": "John"},
  "analytics": {"analytics": {"views": 100}}
}
```

#### 2. **Nested** (Sequential)
Creates separate top-level keys for each backend:
```json
{
  "source_backend": {...},
  "target_backend": {...}
}
```

#### 3. **Flat** (Sequential)
Merges all fields into a single flat structure:
```json
// Source
{"user_id": 123, "name": "John"}

// Target
{"views": 100, "last_login": "2024-01-01"}

// Result
{"user_id": 123, "name": "John", "views": 100, "last_login": "2024-01-01"}
```

#### 4. **Join** (Parallel)
Joins multiple backend responses by a common field:
```json
// Backend 1
[{"id": "1", "title": "Conv 1"}, {"id": "2", "title": "Conv 2"}]

// Backend 2
[{"id": "1", "participants": ["user1"]}, {"id": "2", "participants": ["user2", "user3"]}]

// Result (with merge_into_field not set)
[
  {"id": "1", "title": "Conv 1", "participants": ["user1"]},
  {"id": "2", "title": "Conv 2", "participants": ["user2", "user3"]}
]
```

## Advanced Features

### Filtering on Empty Responses

When `filter_on_empty: true`, items that don't have corresponding ancillary data are removed from the result:

```yaml
map_reduce:
  type: "parallel"
  filter_on_empty: true  # Remove items without ancillary data
```

**Example**:
```
Base backend: [{"id": "1"}, {"id": "2"}, {"id": "3"}]
Ancillary backend: [{"id": "1"}, {"id": "3"}]  # Missing id "2"

Result: [{"id": "1"}, {"id": "3"}]  # Item "2" filtered out
```

### Custom HTTP Methods and Paths

The target backend request can be customized:

```yaml
mapping:
  target_request_path: "/api/v2/bulk/process"
  target_request_method: "PUT"
```

This sends the extracted data to the specified path with the specified HTTP method.

### Handling Empty Source Data

Configure behavior when the source backend returns no data:

```yaml
allow_empty_responses: true   # Return source as-is if extraction yields nothing
allow_empty_responses: false  # Return 204 No Content if extraction yields nothing
```

## Error Handling

### Source Backend Failure
- Returns `502 Bad Gateway`
- Error message indicates source backend failure
- Target backend is not called

### Target Backend Failure with `allow_empty_responses: true`
- Returns `200 OK`
- Returns source response unchanged
- Graceful degradation for optional enrichment data

### Target Backend Failure with `allow_empty_responses: false`
- Returns `502 Bad Gateway`
- Error message indicates target backend failure

### Parallel Backend Partial Failure
- With `allow_empty_responses: true`: Returns data from successful backends
- With `allow_empty_responses: false`: Returns `502 Bad Gateway`

## Performance Considerations

### Sequential Pattern
- Total time = source request + extraction + target request
- Suitable for: <1000 items per request
- Consider caching if this is a frequently accessed endpoint

### Parallel Pattern
- Total time = max(backend response times)
- Much faster than sequential when backends are independent
- All requests execute concurrently
- Suitable for: Real-time aggregation of distributed data

### Large Datasets
For datasets >1000 items:
- Consider pagination at the source
- Implement request timeout tuning
- Monitor memory usage during join operations
- Consider async/background processing for very large datasets

## Example: Conversation List with Follow-ups

This is the canonical use case described in the requirements:

```yaml
composite_routes:
  "/api/conversations/list":
    pattern: "/api/conversations/list"
    strategy: "map-reduce"
    map_reduce:
      type: "sequential"
      source_backend: "conversations"
      target_backend: "followups"
      mapping:
        extract_path: "conversations"
        extract_field: "id"
        target_request_field: "conversation_ids"
        target_request_path: "/bulk/followups"
        target_request_method: "POST"
        merge_into_field: "followup_data"
      merge_strategy: "enrich"
      allow_empty_responses: true
```

**Conversations Backend** (`GET /api/conversations`):
```json
{
  "conversations": [
    {"id": "conv1", "title": "Support Request", "status": "open"},
    {"id": "conv2", "title": "Billing Question", "status": "active"},
    {"id": "conv3", "title": "Bug Report", "status": "closed"}
  ]
}
```

**Followups Backend** receives `POST /bulk/followups`:
```json
{
  "conversation_ids": ["conv1", "conv2", "conv3"]
}
```

**Followups Backend** responds:
```json
{
  "followups": [
    {"conversation_id": "conv1", "is_followup": true, "parent_id": "orig1"},
    {"conversation_id": "conv3", "is_followup": false}
  ]
}
```

**Final Response** to client:
```json
{
  "conversations": [
    {"id": "conv1", "title": "Support Request", "status": "open"},
    {"id": "conv2", "title": "Billing Question", "status": "active"},
    {"id": "conv3", "title": "Bug Report", "status": "closed"}
  ],
  "followup_data": {
    "followups": [
      {"conversation_id": "conv1", "is_followup": true, "parent_id": "orig1"},
      {"conversation_id": "conv3", "is_followup": false}
    ]
  }
}
```

## Testing

See the comprehensive test suite:
- Unit tests: `mapreduce_test.go`
- BDD tests: `bdd_mapreduce_test.go` and `features/mapreduce_composite.feature`
- Example: `examples/reverse-proxy` with live map/reduce routes

To test the example:
```bash
cd examples/reverse-proxy
go run main.go

# In another terminal
# Sequential map/reduce
curl http://localhost:8080/api/composite/mapreduce/conversations

# Parallel map/reduce with join
curl http://localhost:8080/api/composite/mapreduce/parallel-join
```

## Migration from Custom Transformers

If you're currently using custom `ResponseTransformer` functions for similar scenarios, consider migrating to map/reduce configuration:

**Before** (Custom Transformer):
```go
proxyModule.SetResponseTransformer("/api/enriched", func(responses map[string]*http.Response) (*http.Response, error) {
    // Complex custom merging logic
    // ...
})
```

**After** (Map/Reduce Config):
```yaml
composite_routes:
  "/api/enriched":
    strategy: "map-reduce"
    map_reduce:
      type: "sequential"
      # Declarative configuration
```

**Benefits**:
- Configuration-driven (no code changes)
- Testable through BDD scenarios
- Consistent error handling
- Built-in performance optimizations
- Easier to understand and maintain

Custom transformers are still available for scenarios requiring complex business logic that can't be expressed declaratively.
