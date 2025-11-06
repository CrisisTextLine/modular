# Route Specificity in ChiMux Module

## Overview

This document explains how the ChiMux module handles route specificity and parameter matching, addressing concerns about parameterized routes overshadowing more specific sibling routes.

## Summary

The ChiMux module uses the [go-chi/chi](https://github.com/go-chi/chi) router, which implements a **radix tree-based routing algorithm** that automatically handles route specificity correctly. More specific routes are always matched before less specific ones, **regardless of registration order**.

## Key Behaviors

### 1. Static Segments Have Priority Over Parameters

```go
router.Get("/api/users/admin", adminHandler)    // Higher priority
router.Get("/api/users/{id}", userHandler)      // Lower priority
```

A request to `/api/users/admin` will always match the first route, never the second.

### 2. More Specific Routes Are Matched First

```go
router.Get("/api/items/{id}", itemHandler)        // Less specific
router.Get("/api/items/{id}/public", publicHandler) // More specific
```

A request to `/api/items/123/public` will always match the second route, never the first, **even if the first route was registered first**.

### 3. Catch-All Routes Have Lowest Priority

```go
router.Get("/api/users", listUsersHandler)
router.Get("/api/users/{id}", getUserHandler)
router.Get("/*", spaHandler)  // Lowest priority
```

The catch-all route only matches paths that don't match any more specific route.

### 4. Registration Order Is Irrelevant

Chi's radix tree algorithm ensures that routes are matched by specificity, not registration order:

```go
// Order 1: Generic first, then specific
router.Get("/api/v1/wishlists/{id}", protectedHandler)
router.Get("/api/v1/wishlists/{id}/public", publicHandler)

// Order 2: Specific first, then generic  
router.Get("/api/v1/wishlists/{id}/public", publicHandler)
router.Get("/api/v1/wishlists/{id}", protectedHandler)
```

Both registration orders produce identical routing behavior.

## Common Use Cases

### Public vs Protected Endpoints

A common pattern is having both authenticated and unauthenticated endpoints for the same resource:

```go
// Protected endpoint - requires authentication
router.Get("/api/v1/wishlists/{id}", func(w http.ResponseWriter, r *http.Request) {
    if r.Header.Get("Authorization") == "" {
        http.Error(w, "Authorization required", http.StatusUnauthorized)
        return
    }
    // Return full data including private fields
})

// Public endpoint - no authentication
router.Get("/api/v1/wishlists/{id}/public", func(w http.ResponseWriter, r *http.Request) {
    // Return only public data
})
```

Requests to `/api/v1/wishlists/123/public` will always route to the public handler without requiring authentication.

### SPA with API Routes

Single-page applications often need a catch-all route for client-side routing:

```go
// API routes
router.Get("/api/v1/users", listUsersHandler)
router.Get("/api/v1/users/{id}", getUserHandler)
router.Get("/api/v1/posts", listPostsHandler)

// Catch-all for SPA
router.Get("/*", func(w http.ResponseWriter, r *http.Request) {
    http.ServeFile(w, r, "static/index.html")
})
```

All `/api/v1/*` routes will be handled by their specific handlers, while other paths like `/about`, `/contact`, etc., will serve the SPA.

## Testing

The `route_specificity_test.go` file contains comprehensive tests that verify:

1. **TestSpecificRouteAfterGeneric**: Specific routes work correctly when registered after generic routes
2. **TestSpecificRouteBeforeGeneric**: Specific routes work correctly when registered before generic routes
3. **TestCatchAllDoesNotOverrideApi**: Catch-all routes don't intercept API routes
4. **TestMultiLevelParameterizedRoutes**: Complex multi-level parameterized paths work correctly
5. **TestRouteWithMiddleware**: Route specificity is maintained even with middleware

All tests pass, confirming that the Chi router handles route specificity correctly.

## Why This Works

Chi uses a **radix tree (trie) data structure** for route matching:

1. When a route is registered, it's inserted into the tree based on its path segments
2. Static segments create distinct branches in the tree
3. Parameter segments (`{id}`) are stored as wildcard branches
4. During request matching, Chi traverses the tree:
   - It tries to match static segments first
   - Only uses parameter segments when no static match exists
   - Returns the most specific match found

This algorithm guarantees O(log n) lookup time and correct precedence handling.

## Comparison to Naive Implementations

A naive router implementation might simply iterate through routes in registration order and match the first pattern that fits. This would cause the overshadowing problem described in the issue.

For example, a buggy implementation:

```go
// BUGGY - Don't do this!
func matchPattern(pattern, path string) bool {
    patternSegs := strings.Split(pattern, "/")
    pathSegs := strings.Split(path, "/")
    
    if len(pathSegs) < len(patternSegs) {
        return false
    }
    
    for i := range patternSegs {
        if isParam(patternSegs[i]) {
            continue  // Any segment matches a parameter
        }
        if patternSegs[i] != pathSegs[i] {
            return false
        }
    }
    
    return true  // BUG: Should check len(pathSegs) == len(patternSegs)
}
```

This buggy implementation would match `/api/items/{id}` against `/api/items/123/public` because it doesn't verify that all path segments are consumed.

**The ChiMux module does NOT have this problem** because it uses Chi's proven routing algorithm.

## Conclusion

The ChiMux module correctly handles route specificity through Chi's radix tree-based routing. Developers can confidently:

- Register routes in any order
- Use both generic and specific parameterized routes
- Implement public and protected endpoints for the same resource
- Use catch-all routes without interfering with API routes

No additional configuration or workarounds are needed.
