package chimux

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/CrisisTextLine/modular"
	"github.com/go-chi/chi/v5"
)

// TestSpecificRouteAfterGeneric verifies that a more specific route
// registered after a generic parameterized route is correctly matched.
// This ensures that /api/v1/items/{id}/public takes precedence over
// /api/v1/items/{id} when the request path matches the specific route.
func TestSpecificRouteAfterGeneric(t *testing.T) {
	app := setupTestApp(t)
	module := setupChiMuxModule(t, app)

	// Register generic param route first
	module.Get("/api/v1/items/{id}", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "" {
			http.Error(w, "Authorization header required", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		id := chi.URLParam(r, "id")
		fmt.Fprintf(w, `{"kind":"protected","id":"%s"}`, id)
	})

	// Register more specific public route after
	module.Get("/api/v1/items/{id}/public", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		id := chi.URLParam(r, "id")
		fmt.Fprintf(w, `{"kind":"public","id":"%s"}`, id)
	})

	// Test that the protected route requires auth
	t.Run("protected route requires auth", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/items/123", nil)
		w := httptest.NewRecorder()
		module.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
		}
	})

	// Test that the public route works without auth
	t.Run("public route works without auth", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/items/123/public", nil)
		w := httptest.NewRecorder()
		module.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
		}

		expected := `{"kind":"public","id":"123"}`
		if w.Body.String() != expected {
			t.Errorf("Expected body %q, got %q", expected, w.Body.String())
		}

		contentType := w.Header().Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", contentType)
		}
	})

	// Test that protected route works with auth
	t.Run("protected route works with auth", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/items/abc", nil)
		req.Header.Set("Authorization", "Bearer token")
		w := httptest.NewRecorder()
		module.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		expected := `{"kind":"protected","id":"abc"}`
		if w.Body.String() != expected {
			t.Errorf("Expected body %q, got %q", expected, w.Body.String())
		}
	})
}

// TestSpecificRouteBeforeGeneric verifies that route matching works correctly
// regardless of registration order. The more specific route should still be
// matched even if registered before the generic route.
func TestSpecificRouteBeforeGeneric(t *testing.T) {
	app := setupTestApp(t)
	module := setupChiMuxModule(t, app)

	// Register more specific public route FIRST
	module.Get("/api/v1/wishlists/{id}/public", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		id := chi.URLParam(r, "id")
		fmt.Fprintf(w, `{"kind":"public","id":"%s"}`, id)
	})

	// Register generic param route AFTER
	module.Get("/api/v1/wishlists/{id}", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "" {
			http.Error(w, "Authorization header required", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		id := chi.URLParam(r, "id")
		fmt.Fprintf(w, `{"kind":"protected","id":"%s"}`, id)
	})

	// Test that public route still works without auth (same as previous test)
	t.Run("public route works without auth", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/wishlists/456/public", nil)
		w := httptest.NewRecorder()
		module.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
		}

		expected := `{"kind":"public","id":"456"}`
		if w.Body.String() != expected {
			t.Errorf("Expected body %q, got %q", expected, w.Body.String())
		}
	})

	// Test that protected route still requires auth
	t.Run("protected route requires auth", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/wishlists/456", nil)
		w := httptest.NewRecorder()
		module.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
		}
	})

	// Test that protected route works with auth
	t.Run("protected route works with auth", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/wishlists/xyz", nil)
		req.Header.Set("Authorization", "Bearer token")
		w := httptest.NewRecorder()
		module.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		expected := `{"kind":"protected","id":"xyz"}`
		if w.Body.String() != expected {
			t.Errorf("Expected body %q, got %q", expected, w.Body.String())
		}
	})
}

// TestCatchAllDoesNotOverrideApi verifies that catch-all routes (/*) don't
// intercept specific API routes. This ensures proper route priority handling.
func TestCatchAllDoesNotOverrideApi(t *testing.T) {
	app := setupTestApp(t)
	module := setupChiMuxModule(t, app)

	// Register specific API routes
	module.Get("/api/v1/users", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"users":[]}`)
	})

	module.Get("/api/v1/users/{id}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		id := chi.URLParam(r, "id")
		fmt.Fprintf(w, `{"user":{"id":"%s"}}`, id)
	})

	// Register catch-all route that serves HTML
	module.Get("/*", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, "<!DOCTYPE html><html><body>Shell</body></html>")
	})

	// Test that API routes return JSON, not HTML
	t.Run("list users returns JSON", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/users", nil)
		w := httptest.NewRecorder()
		module.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		contentType := w.Header().Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s. Body: %s", contentType, w.Body.String())
		}

		expected := `{"users":[]}`
		if w.Body.String() != expected {
			t.Errorf("Expected body %q, got %q", expected, w.Body.String())
		}
	})

	t.Run("get user by id returns JSON", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/users/789", nil)
		w := httptest.NewRecorder()
		module.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		contentType := w.Header().Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", contentType)
		}

		expected := `{"user":{"id":"789"}}`
		if w.Body.String() != expected {
			t.Errorf("Expected body %q, got %q", expected, w.Body.String())
		}
	})

	t.Run("unknown route returns HTML from catch-all", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/unknown/path", nil)
		w := httptest.NewRecorder()
		module.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		contentType := w.Header().Get("Content-Type")
		if contentType != "text/html" {
			t.Errorf("Expected Content-Type text/html, got %s", contentType)
		}
	})

	t.Run("root route returns HTML from catch-all", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()
		module.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		contentType := w.Header().Get("Content-Type")
		if contentType != "text/html" {
			t.Errorf("Expected Content-Type text/html, got %s", contentType)
		}
	})
}

// TestMultiLevelParameterizedRoutes tests complex routing scenarios with
// multiple levels of parameterized paths to ensure proper specificity handling.
func TestMultiLevelParameterizedRoutes(t *testing.T) {
	app := setupTestApp(t)
	module := setupChiMuxModule(t, app)

	// Register routes with different levels of specificity
	module.Get("/api/v1/orgs/{orgId}/projects/{projectId}/public", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		orgId := chi.URLParam(r, "orgId")
		projectId := chi.URLParam(r, "projectId")
		fmt.Fprintf(w, `{"kind":"public","org":"%s","project":"%s"}`, orgId, projectId)
	})

	module.Get("/api/v1/orgs/{orgId}/projects/{projectId}", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "" {
			http.Error(w, "Authorization required", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		orgId := chi.URLParam(r, "orgId")
		projectId := chi.URLParam(r, "projectId")
		fmt.Fprintf(w, `{"kind":"protected","org":"%s","project":"%s"}`, orgId, projectId)
	})

	module.Get("/api/v1/orgs/{orgId}/projects/{projectId}/members", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		orgId := chi.URLParam(r, "orgId")
		projectId := chi.URLParam(r, "projectId")
		fmt.Fprintf(w, `{"members":[],"org":"%s","project":"%s"}`, orgId, projectId)
	})

	t.Run("public endpoint works", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/orgs/org1/projects/proj1/public", nil)
		w := httptest.NewRecorder()
		module.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
		}

		expected := `{"kind":"public","org":"org1","project":"proj1"}`
		if w.Body.String() != expected {
			t.Errorf("Expected body %q, got %q", expected, w.Body.String())
		}
	})

	t.Run("members endpoint works", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/orgs/org2/projects/proj2/members", nil)
		w := httptest.NewRecorder()
		module.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
		}

		expected := `{"members":[],"org":"org2","project":"proj2"}`
		if w.Body.String() != expected {
			t.Errorf("Expected body %q, got %q", expected, w.Body.String())
		}
	})

	t.Run("protected endpoint requires auth", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/orgs/org3/projects/proj3", nil)
		w := httptest.NewRecorder()
		module.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
		}
	})
}

// TestRouteWithMiddleware verifies that specific routes work correctly
// even when middleware is applied, ensuring middleware doesn't interfere
// with route matching.
func TestRouteWithMiddleware(t *testing.T) {
	app := setupTestApp(t)
	module := setupChiMuxModule(t, app)

	// Add middleware that adds a header
	module.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Middleware", "applied")
			next.ServeHTTP(w, r)
		})
	})

	// Register routes
	module.Get("/api/v1/resources/{id}/public", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		id := chi.URLParam(r, "id")
		fmt.Fprintf(w, `{"public":true,"id":"%s"}`, id)
	})

	module.Get("/api/v1/resources/{id}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		id := chi.URLParam(r, "id")
		fmt.Fprintf(w, `{"public":false,"id":"%s"}`, id)
	})

	t.Run("public route with middleware", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/resources/res1/public", nil)
		w := httptest.NewRecorder()
		module.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		// Check middleware was applied
		if w.Header().Get("X-Middleware") != "applied" {
			t.Error("Expected middleware to be applied")
		}

		expected := `{"public":true,"id":"res1"}`
		if w.Body.String() != expected {
			t.Errorf("Expected body %q, got %q", expected, w.Body.String())
		}
	})

	t.Run("generic route with middleware", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/resources/res2", nil)
		w := httptest.NewRecorder()
		module.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		// Check middleware was applied
		if w.Header().Get("X-Middleware") != "applied" {
			t.Error("Expected middleware to be applied")
		}

		expected := `{"public":false,"id":"res2"}`
		if w.Body.String() != expected {
			t.Errorf("Expected body %q, got %q", expected, w.Body.String())
		}
	})
}

// Helper functions

func setupTestApp(t *testing.T) modular.TenantApplication {
	t.Helper()
	app := NewMockApplication()
	return app
}

func setupChiMuxModule(t *testing.T, app modular.TenantApplication) *ChiMuxModule {
	t.Helper()

	module := NewChiMuxModule().(*ChiMuxModule)

	// Register config
	err := module.RegisterConfig(app)
	if err != nil {
		t.Fatalf("Failed to register config: %v", err)
	}

	// Initialize module
	err = module.Init(app)
	if err != nil {
		t.Fatalf("Failed to initialize module: %v", err)
	}

	return module
}
