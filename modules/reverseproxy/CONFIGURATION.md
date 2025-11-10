# Reverse Proxy Module - Configuration Guide

This document provides comprehensive configuration guidance for the Reverse Proxy module, including all features, configuration inheritance, and tenant-specific overrides.

## Table of Contents

- [Feature Summary](#feature-summary)
- [Configuration Reference](#configuration-reference)
- [Routing, Load Balancing, and Tenants](#routing-load-balancing-and-tenants)
- [Composite Routes and Response Caching](#composite-routes-and-response-caching)
- [Feature Flags and Dry Runs](#feature-flags-and-dry-runs)
- [Metrics, Health, Debug, and Events](#metrics-health-debug-and-events)
- [Operational Checklist](#operational-checklist)
- [Example Configuration](#example-configuration)

## Feature Summary

- Multi-tenant aware routing that merges tenant configuration on top of global defaults without duplicating the full tree.
- Direct backend routing, composite fan-out, and custom aggregation endpoints that can rewrite headers and paths per backend or endpoint.
- Built-in circuit breaker integration, health checking, metrics, debug endpoints, and structured CloudEvents-compatible event emission.
- Extensible feature flag evaluation with automatic discovery of `FeatureFlagEvaluator` services plus a file-based fallback.
- Optional dry-run comparisons that execute primary and alternative backends side-by-side and capture detailed diffs.
- Experimental response caching for composite handlers with TTL controls and tenant overrides.

## Configuration Reference

| Field | Purpose | Notes |
| --- | --- | --- |
| `backend_services` | Map backend IDs to base URLs used when constructing reverse proxies. | Empty strings are allowed so that tenants can supply URLs while globals stay blank. |
| `routes` | Route patterns mapped to backend IDs (or comma-separated backend groups). | Patterns use glob matching; comma-delimited values enable simple round-robin with load-balancing events. |
| `route_configs["pattern"]` | Per-route behaviour overrides. | Supports feature flag gating, alternative backends, per-route timeouts, and enabling dry-run comparisons on only the routes that need them. |
| `default_backend` | Catch-all backend when no specific route matches. | Registered as `/*` but skipped for health, metrics, and debug endpoints. Must be present in `backend_services`. |
| `composite_routes` | Multi-backend aggregation definitions. | Each entry provides a pattern, ordered backend list, optional strategy string, optional `feature_flag_id`, and `alternative_backend` fallback. |
| `backend_configs["id"]` | Backend-specific tuning. | Allows path/header rewriting, endpoint-specific overrides, retry and connection pool tuning, backend health configuration, and its own feature flag/alternative backend logic. |
| `cache_enabled` / `cache_ttl` | Governs the in-memory response cache checked by composite handlers. | TTL defaults to 60s when enabled. Only GET responses that return 200 are cached. Tenant configs can increase TTL, but once the global config enables caching tenants cannot turn it back off because of the current merge semantics. |
| `tenant_id_header` / `require_tenant_id` | Tenant enforcement. | The module rejects requests with HTTP 400 when the header is required but missing. Default header is `X-Tenant-ID`. |
| `request_timeout` | Default timeout for outbound backend requests. | Individual routes may override via `route_configs[*].timeout`. |
| `metrics_enabled` / `metrics_endpoint` & `metrics_config` | JSON metrics exposure. | Enables `/metrics` style JSON output and automatically wires a `.../health` endpoint when the health checker runs. |
| `health_check` | Background probing configuration. | Handles per-backend overrides, integrates with circuit breaker status, and emits backend healthy/unhealthy events. |
| `feature_flags` | Built-in file-backed evaluator defaults. | When enabled the module registers a tenant-aware evaluator and also aggregates any external `FeatureFlagEvaluator` services by weight. |
| `dry_run` | Module-level dry-run settings. | Controls parallel comparisons, max body size, headers to compare/ignore, and which backend's response is ultimately returned. |
| `debug_endpoints` | Diagnostic HTTP endpoints. | Exposes `/debug` (customisable `base_path`) routes for flags, info, backends, circuit breakers, and health checks. Can optionally require an auth token. |
| `circuit_breaker` / `backend_circuit_breakers` | Failure isolation. | Provides global defaults plus per-backend overrides; state changes trigger CloudEvents so you can observe openings, closures, and half-open transitions. |

> **Note:** The `global_timeout` configuration value is currently persisted with the module but is not yet consumed in request handling; keep per-route `timeout` and the global `request_timeout` aligned with your SLOs until the orchestration logic starts honouring it.

## Routing, Load Balancing, and Tenants

During `Start`, the module:

- Loads tenant configs via `mergeConfigs`, overlaying tenant-provided data over the global struct. Map fields (`routes`, `composite_routes`, `backend_configs`, `backend_circuit_breakers`) merge with tenant entries replacing matching global keys.
- Validates that required services exist (router, optional `httpclient`, optional `featureFlagEvaluator`) and constructs `httputil.ReverseProxy` instances for every global backend plus each tenant-specific override.
- Registers handlers for every explicit route. When a `routes` value contains commas (for example `"api-a, api-b"`), `selectBackendFromGroup` rotates through the candidates and emits `com.modular.reverseproxy.loadbalance.*` events on every decision.
- Falls back to `default_backend` for unmatched traffic while deliberately skipping `/health`, debug, and metrics endpoints so internal handlers can respond locally.
- Enforces tenant presence when `require_tenant_id` is true by returning HTTP 400 before proxying.
- Applies backend and endpoint path/header rewriting based on the relevant `backend_configs` and `endpoint` overrides for each request, preserving tenant-specific substitutions.

Because `mergeConfigs` declares `CacheEnabled` and `MetricsEnabled` as opt-in flags, a tenant can switch those features on even when they are disabled globally, but cannot turn them off once the global config enables them. Plan config hierarchies accordingly.

## Composite Routes and Response Caching

Composite routes use `CompositeHandler`, which fans out requests (parallel by default) using the module's configured HTTP client. The handler:

- Replays request bodies to each backend, merges the first successful response based on the configured backend order, and returns JSON/headers taken from the first winning backend.
- Supports custom aggregation with `RegisterCustomEndpoint`, letting you define per-endpoint HTTP method, query rewrites, and a `ResponseTransformer` to build the final payload.
- Consults the optional in-memory response cache. Only GET requests with 200 responses are cached by default. The cache key incorporates method, URL, and select headers (Accept / Accept-Encoding).

`cache_enabled` simply validates configuration today and exposes `cache_ttl` to the handler. The handler only caches responses when the module has been initialised with a `responseCache` instance (tests populate this explicitly). Until the automatic wiring lands, treat caching as experimental and verify behaviour with integration tests or by checking for the `Cleaned up response cache` shutdown log.

Tenants inherit the global cache settings automatically; specifying a tenant-level `cache_ttl` increases the TTL just for that tenant's merged configuration.

## Feature Flags and Dry Runs

Feature flag checks run through the aggregator in this order:

1. External services that implement `FeatureFlagEvaluator`, sorted by their optional `Weight()` (lower numbers run first).
2. The built-in file-backed evaluator (weight 1000) when `feature_flags.enabled` is true, which honours tenant overrides stored in the Modular config provider.

Route-level controls (`route_configs[*].feature_flag_id`) can swap traffic to an `alternative_backend` or a list of alternatives when the flag resolves to false. Backend-level feature flags inside `backend_configs` behave the same way. Composite routes can also attach a flag; when disabled the request is routed to `alternative_backend` or a 404 is returned.

Dry-run mode builds on these controls: enabling `dry_run` at the route level causes the module to send the request to both the primary backend and either `dry_run_backend` or the alternative backend. The `DryRunHandler` logs comparison details (status, headers, bodies, timing) and returns either the primary or secondary response based on `dry_run.default_response_backend`. Use this to compare legacy and new services without affecting callers.

## Metrics, Health, Debug, and Events

- **Metrics**: when `metrics_enabled` or `metrics_config.enabled` is true, the module registers a JSON endpoint (default `/metrics`) containing per-backend counters and latencies. If the health checker runs, an additional `.../health` endpoint surfaces aggregate health status (HTTP 200/503).
- **Health checks**: `health_check.enabled` starts a background probe that reuses the module's HTTP client, respects `recent_request_threshold`, and emits `backend.healthy` / `backend.unhealthy` events.
- **Debug endpoints**: setting `debug_endpoints.enabled` registers routes under `base_path` (default `/debug`) for flags, backends, info, circuit breakers, and health checks. Optional auth tokens gate access.
- **Event emission**: every request emits `request.received` and either `request.proxied` or `request.failed`. Circuit breaker transitions emit `circuitbreaker.open/closed/halfopen`, load balancer decisions emit `loadbalance.decision`, and lifecycle hooks emit `module.started`, `proxy.started`, `proxy.stopped`, and `module.stopped`. Subscribe via the observer infrastructure to stream these CloudEvents.

## Operational Checklist

- Register a router service that satisfies the `routerService` interface before starting the application.
- Provide an `httpclient` service if you need custom transport or TLS settings; otherwise the module creates a pooled client with sane defaults.
- Set `backend_services` for every backend referenced by `routes`, `composite_routes`, or tenant overrides. Tenant configs that rely on unique URLs must be available through the `TenantService`.
- If you enable caching, confirm the response cache has been initialised (look for cache logs during shutdown or write a quick integration test) before depending on it in production.
- Decide whether observers need CloudEvents and register them before `Init()` so they capture configuration and lifecycle events.
- Expose debug and metrics endpoints behind appropriate auth or network controls, especially when `RequireAuth` is false.

## Example Configuration

```yaml
reverseproxy:
    backend_services:
        catalog:  "https://catalog.internal:8443"
        reviews:  "https://reviews.internal:8443"
        fallback: "https://legacy.internal:8443"

    routes:
        /api/catalog/*: catalog
        /api/reviews/*: reviews
        /api/legacy/*: fallback

    default_backend: fallback

    route_configs:
        /api/reviews/*:
            feature_flag_id: enable-reviews
            alternative_backend: fallback
            timeout: 750ms
            dry_run: true
            dry_run_backend: fallback

    composite_routes:
        /api/product/*:
            backends: [catalog, reviews]
            strategy: "first-success"
            feature_flag_id: enable-product-composite
            alternative_backend: catalog

    backend_configs:
        catalog:
            header_rewriting:
                hostname_handling: use_backend
        reviews:
            path_rewriting:
                strip_base_path: /api/reviews
                base_path_rewrite: /v1

    cache_enabled: true
    cache_ttl: 120s
    require_tenant_id: true
    tenant_id_header: X-Tenant-ID

    feature_flags:
        enabled: true
        flags:
            enable-reviews: true
            enable-product-composite: false

    dry_run:
        enabled: true
        log_responses: true
        default_response_backend: primary

    circuit_breaker:
        enabled: true
        failure_threshold: 5
        open_timeout: 30s

    backend_circuit_breakers:
        reviews:
            enabled: true
            failure_threshold: 3
            recovery_timeout: 15s

    health_check:
        enabled: true
        interval: 30s
        timeout: 5s
        health_endpoints:
            reviews: /internal/health

    metrics_enabled: true
    metrics_endpoint: /metrics/reverseproxy

    debug_endpoints:
        enabled: true
        base_path: /debug/reverseproxy
        require_auth: true
        auth_token: super-secret-token
```

Tenant-specific overrides can now focus on the deltas, for example enabling caching with a shorter TTL, pointing a subset of backends at regional URLs, or flipping a feature flag without duplicating the whole structure.
