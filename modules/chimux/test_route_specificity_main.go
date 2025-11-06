//go:build ignore
// +build ignore

package main

import (
"fmt"
"net/http"
"net/http/httptest"
"github.com/go-chi/chi/v5"
)

func main() {
r := chi.NewRouter()

// Register param route first
r.Get("/api/v1/items/{id}", func(w http.ResponseWriter, r *http.Request) {
if r.Header.Get("Authorization") == "" {
http.Error(w, "Authorization header required", http.StatusUnauthorized)
return
}
w.Header().Set("Content-Type", "application/json")
id := chi.URLParam(r, "id")
fmt.Fprintf(w, `{"kind":"protected","id":"%s"}`, id)
})

// Register more specific public route
r.Get("/api/v1/items/{id}/public", func(w http.ResponseWriter, r *http.Request) {
w.Header().Set("Content-Type", "application/json")
id := chi.URLParam(r, "id")
fmt.Fprintf(w, `{"kind":"public","id":"%s"}`, id)
})

// Test the routes
testRequest := func(path string) {
req := httptest.NewRequest("GET", path, nil)
w := httptest.NewRecorder()
r.ServeHTTP(w, req)
fmt.Printf("\nPath: %s\nStatus: %d\nBody: %s\n", path, w.Code, w.Body.String())
}

testRequest("/api/v1/items/123")
testRequest("/api/v1/items/123/public")

// Now try with reversed registration order
fmt.Println("\n=== Reversed Registration Order ===")
r2 := chi.NewRouter()

// Register more specific public route first
r2.Get("/api/v1/items/{id}/public", func(w http.ResponseWriter, r *http.Request) {
w.Header().Set("Content-Type", "application/json")
id := chi.URLParam(r, "id")
fmt.Fprintf(w, `{"kind":"public","id":"%s"}`, id)
})

// Register param route
r2.Get("/api/v1/items/{id}", func(w http.ResponseWriter, r *http.Request) {
if r.Header.Get("Authorization") == "" {
http.Error(w, "Authorization header required", http.StatusUnauthorized)
return
}
w.Header().Set("Content-Type", "application/json")
id := chi.URLParam(r, "id")
fmt.Fprintf(w, `{"kind":"protected","id":"%s"}`, id)
})

testRequest2 := func(path string) {
req := httptest.NewRequest("GET", path, nil)
w := httptest.NewRecorder()
r2.ServeHTTP(w, req)
fmt.Printf("\nPath: %s\nStatus: %d\nBody: %s\n", path, w.Code, w.Body.String())
}

testRequest2("/api/v1/items/123")
testRequest2("/api/v1/items/123/public")
}
